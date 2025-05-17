package backend

import (
	"errors"
	"family/db"
	"time"

	"go.hasen.dev/vbeam"
	"go.hasen.dev/vbolt"
	"go.hasen.dev/vpack"
	"golang.org/x/crypto/bcrypt"
)

func RegisterUserMethods(app *vbeam.Application) {
	vbeam.RegisterProc(app, AddUser)
	vbeam.RegisterProc(app, GetAuthContext)
}

type AddUserRequest struct {
	Email     string
	Password  string
	FirstName string
	LastName  string
}

type AuthRequest struct {
	Email    string
	Password string
}

type UserListResponse struct {
}

type LoginResponse struct {
	Success bool
	Token   string
	Auth    AuthResponse
}

type AuthResponse struct {
	Id        int
	Email     string
	FirstName string
	LastName  string
	isAdmin   bool
}

type User struct {
	Id        int
	Email     string
	Creation  time.Time
	LastLogin time.Time
	FirstName string
	LastName  string
}

func PackUser(self *User, buf *vpack.Buffer) {
	vpack.Version(1, buf)
	vpack.Int(&self.Id, buf)
	vpack.String(&self.Email, buf)
	vpack.Time(&self.Creation, buf)
	vpack.Time(&self.LastLogin, buf)
	vpack.String(&self.FirstName, buf)
	vpack.String(&self.LastName, buf)
}

// Buckets
// =============================================================================

var UsersBkt = vbolt.Bucket(&db.Info, "users", vpack.FInt, PackUser)

// user id => hashed passwd
var PasswdBkt = vbolt.Bucket(&db.Info, "passwd", vpack.FInt, vpack.ByteSlice)

// username => userid
var EmailBkt = vbolt.Bucket(&db.Info, "email", vpack.StringZ, vpack.Int)

// token => user id
var ResetPasswordBucket = vbolt.Bucket(&db.Info, "password-token", vpack.String, vpack.FInt)

// token => user id
var RefreshBucket = vbolt.Bucket(&db.Info, "login-token", vpack.String, vpack.FInt)

func isPasswordValid(pwd string) bool {
	return len(pwd) >= 8 && len(pwd) <= 72
}

var ErrEmailTaken = errors.New("EmailTaken")
var ErrPasswordInvalid = errors.New("PasswordInvalid")

func GetUserId(tx *vbolt.Tx, username string) (userId int) {
	vbolt.Read(tx, EmailBkt, username, &userId)
	return
}

func GetUser(tx *vbolt.Tx, userId int) (user User) {
	vbolt.Read(tx, UsersBkt, userId, &user)
	return
}

func GetPassHash(tx *vbolt.Tx, userId int) (hash []byte) {
	vbolt.Read(tx, PasswdBkt, userId, &hash)
	return
}

func GetUserIdFromToken(tx *vbolt.Tx, token string) (userId int) {
	vbolt.Read(tx, ResetPasswordBucket, token, &userId)
	return
}

func GetUserIdFromRefreshToken(tx *vbolt.Tx, token string) (userId int) {
	vbolt.Read(tx, RefreshBucket, token, &userId)
	vbolt.Delete(tx, RefreshBucket, token)
	return
}

func AddUserTx(tx *vbolt.Tx, req AddUserRequest, hash []byte) User {
	var user User
	user.Id = vbolt.NextIntId(tx, UsersBkt)
	user.Email = req.Email
	user.FirstName = req.FirstName
	user.LastName = req.LastName
	user.Creation = time.Now()

	vbolt.Write(tx, UsersBkt, user.Id, &user)
	vbolt.Write(tx, PasswdBkt, user.Id, &hash)
	vbolt.Write(tx, EmailBkt, user.Email, &user.Id)
	return user
}

func ValidateUserTx(tx *vbolt.Tx, req AddUserRequest) error {
	if vbolt.HasKey(tx, EmailBkt, req.Email) {
		return ErrEmailTaken
	}

	if !isPasswordValid(req.Password) {
		return ErrPasswordInvalid
	}

	return nil
}

func AddUser(ctx *vbeam.Context, req AddUserRequest) (resp UserListResponse, err error) {
	err = ValidateUserTx(ctx.Tx, req)
	if err != nil {
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	vbeam.UseWriteTx(ctx)
	AddUserTx(ctx.Tx, req, hash)

	vbolt.TxCommit(ctx.Tx)

	return
}

func GetAuthContext(ctx *vbeam.Context, req Empty) (resp AuthResponse, err error) {
	user, authErr := GetAuthUser(ctx)
	if authErr == nil && user.Id > 0 {
		resp = GetAuthResponseFromUser(user)
	}
	return
}

func GetAuthResponseFromUser(user User) (resp AuthResponse) {
	resp.Id = user.Id
	resp.Email = user.Email
	resp.FirstName = user.FirstName
	resp.LastName = user.LastName
	resp.isAdmin = user.Id == 1
	return
}
