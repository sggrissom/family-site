package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/boltdb/bolt"
	"github.com/golang-jwt/jwt/v5"
	"go.hasen.dev/generic"
	"go.hasen.dev/vbolt"
	"go.hasen.dev/vpack"
	"golang.org/x/crypto/bcrypt"
)

var jwtKey = []byte("test-secret-key") //todo: secret, just testing for now

// Models

type RoleType int

const (
	Admin RoleType = iota
	Owner
	Viewer
)

type StatusType int

const (
	Active StatusType = iota
	Suspended
)

type User struct {
	Id              int
	Email           string
	Role            RoleType
	Status          StatusType
	LastLogin       time.Time
	FirstName       string
	LastName        string
	PrimaryFamilyId int
}

func PackUser(self *User, buf *vpack.Buffer) {
	vpack.Version(1, buf)
	vpack.Int(&self.Id, buf)
	vpack.String(&self.Email, buf)
	vpack.IntEnum(&self.Role, buf)
	vpack.IntEnum(&self.Status, buf)
	vpack.Time(&self.LastLogin, buf)
	vpack.String(&self.FirstName, buf)
	vpack.String(&self.LastName, buf)
	vpack.Int(&self.PrimaryFamilyId, buf)
}

// Buckets

var UsersBucket = vbolt.Bucket(&Info, "users", vpack.FInt, PackUser)

// user id => hashed passwd
var PasswordBucket = vbolt.Bucket(&Info, "password", vpack.FInt, vpack.ByteSlice)

var EmailBucket = vbolt.Bucket(&Info, "email", vpack.StringZ, vpack.Int)

type AddUserRequest struct {
	Email     string
	Password  string
	FirstName string
	LastName  string
}
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func isPasswordValid(pwd string) bool {
	return len(pwd) >= 8 && len(pwd) <= 72
}

var ErrEmailTaken = errors.New("EmailTaken")
var ErrPasswordInvalid = errors.New("PasswordInvalid")

func GetAllUsers(tx *vbolt.Tx) (users []User) {
	vbolt.IterateAll(tx, UsersBucket, func(key int, value User) bool {
		generic.Append(&users, value)
		return true
	})
	return users
}

func GetUserId(tx *vbolt.Tx, username string) (userId int) {
	vbolt.Read(tx, EmailBucket, username, &userId)
	return userId
}

func GetUser(tx *vbolt.Tx, userId int) (user User) {
	vbolt.Read(tx, UsersBucket, userId, &user)
	return user
}

func AddUserTx(tx *vbolt.Tx, req AddUserRequest, hash []byte) User {
	var user User
	user.Id = vbolt.NextIntId(tx, UsersBucket)
	user.Email = req.Email
	user.Status = Active
	user.Role = Viewer
	user.FirstName = req.FirstName
	user.LastName = req.LastName

	vbolt.Write(tx, UsersBucket, user.Id, &user)
	vbolt.Write(tx, PasswordBucket, user.Id, &hash)
	vbolt.Write(tx, EmailBucket, user.Email, &user.Id)
	return user
}

func ValidateUserTx(tx *vbolt.Tx, req AddUserRequest) error {
	if vbolt.HasKey(tx, EmailBucket, req.Email) {
		return ErrEmailTaken
	}

	if !isPasswordValid(req.Password) {
		return ErrPasswordInvalid
	}

	return nil
}

func AddUser(dbHandle *bolt.DB, req AddUserRequest) (err error) {
	vbolt.WithReadTx(dbHandle, func(readTx *vbolt.Tx) {
		err = ValidateUserTx(readTx, req)
	})
	if err != nil {
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	vbolt.WithWriteTx(dbHandle, func(tx *vbolt.Tx) {
		AddUserTx(tx, req, hash)
		vbolt.TxCommit(tx)
	})
	return
}

func RegisterLoginPages(mux *http.ServeMux) {
	mux.Handle("GET /login", PublicHandler(ContextFunc(loginPage)))
	mux.Handle("GET /logout", PublicHandler(ContextFunc(logout)))
	mux.Handle("POST /login", PublicHandler(ContextFunc(authenticateLogin)))
	mux.Handle("GET /register", PublicHandler(ContextFunc(registerPage)))
	mux.Handle("POST /register", PublicHandler(ContextFunc(createUser)))
	mux.Handle("GET /profile", AuthHandler(ContextFunc(profilePage)))
}

func loginPage(context ResponseContext) {
	RenderTemplate(context, "login")
}

func registerPage(context ResponseContext) {
	RenderTemplate(context, "register")
}

func profilePage(context ResponseContext) {
	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		families := GetFamiliesForUser(tx, context.user.Id)
		RenderTemplateWithData(context, "profile", map[string]any{
			"Families": families,
		})
	})
}

func createUser(context ResponseContext) {
	addUserRequest := AddUserRequest{
		Email:     context.r.PostFormValue("email"),
		Password:  context.r.PostFormValue("password"),
		FirstName: context.r.PostFormValue("firstname"),
		LastName:  context.r.PostFormValue("lastname"),
	}
	err := AddUser(db, addUserRequest)
	if err != nil {
		http.Error(context.w, err.Error(), http.StatusUnauthorized)
		return
	}

	http.Redirect(context.w, context.r, "/", http.StatusFound)
}

func authenticateLogin(context ResponseContext) {
	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		var userId int
		var user User
		var passHash []byte
		email := context.r.FormValue("email")
		vbolt.Read(tx, EmailBucket, email, &userId)
		vbolt.Read(tx, UsersBucket, userId, &user)
		vbolt.Read(tx, PasswordBucket, userId, &passHash)

		err := bcrypt.CompareHashAndPassword(passHash, []byte(context.r.FormValue("password")))

		if err == nil {
			expirationTime := time.Now().Add(24 * time.Hour)
			claims := &Claims{
				Username: email,
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(expirationTime),
				},
			}
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

			tokenString, err := token.SignedString(jwtKey)
			if err != nil {
				http.Error(context.w, "Error generating token", http.StatusInternalServerError)
				return
			}

			http.SetCookie(context.w, &http.Cookie{
				Name:     "auth_token",
				Value:    tokenString,
				Path:     "/",
				HttpOnly: true,
			})

			vbolt.WithWriteTx(db, func(writeTx *vbolt.Tx) {
				user.LastLogin = time.Now()
				vbolt.Write(writeTx, UsersBucket, userId, &user)
				vbolt.TxCommit(writeTx)
			})

			http.Redirect(context.w, context.r, "/", http.StatusFound)
			return
		}
	})

	http.Error(context.w, "Invalid credentials", http.StatusUnauthorized)
}

func logout(context ResponseContext) {
	http.SetCookie(context.w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Unix(0, 0),
	})

	http.Redirect(context.w, context.r, "/", http.StatusFound)
}
