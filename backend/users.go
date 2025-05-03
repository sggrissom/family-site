package backend

import (
	"errors"
	"family/db"
	"time"

	"go.hasen.dev/generic"
	"go.hasen.dev/vbeam"
	"go.hasen.dev/vbolt"
	"go.hasen.dev/vpack"
	"golang.org/x/crypto/bcrypt"
)

type AddUserRequest struct {
	Email     string
	Password  string
	FirstName string
	LastName  string
}

type UserListResponse struct {
	Users []User
}

func fetchUsers(tx *vbolt.Tx) (users []User) {
	vbolt.IterateAll(tx, UsersBkt, func(key int, value User) bool {
		generic.Append(&users, value)
		return true
	})
	return
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

func isPasswordValid(pwd string) bool {
	return len(pwd) >= 8 && len(pwd) <= 72
}

var EmailTaken = errors.New("EmailTaken")
var PasswordInvalid = errors.New("PasswordInvalid")

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
		return EmailTaken
	}

	if !isPasswordValid(req.Password) {
		return PasswordInvalid
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

	resp.Users = fetchUsers(ctx.Tx)
	generic.EnsureSliceNotNil(&resp.Users)

	vbolt.TxCommit(ctx.Tx)

	return
}

func ListUsers(ctx *vbeam.Context, req Empty) (resp UserListResponse, err error) {
	resp.Users = fetchUsers(ctx.Tx)
	generic.EnsureSliceNotNil(&resp.Users)
	return
}
