package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.hasen.dev/vbolt"
	"go.hasen.dev/vpack"
	"golang.org/x/crypto/bcrypt"
)

var jwtKey = []byte("test-secret-key") //todo: secret, just testing for now

// Models

type User struct {
	Id    int
	Email string
}

func PackUser(self *User, buf *vpack.Buffer) {
	vpack.Version(1, buf)
	vpack.Int(&self.Id, buf)
	vpack.String(&self.Email, buf)
}

// Buckets

var UsersBucket = vbolt.Bucket(&Info, "users", vpack.FInt, PackUser)

// user id => hashed passwd
var PasswordBucket = vbolt.Bucket(&Info, "password", vpack.FInt, vpack.ByteSlice)

var EmailBucket = vbolt.Bucket(&Info, "email", vpack.StringZ, vpack.Int)

type AddUserRequest struct {
	Email    string
	Password string
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

func AddUserTx(tx *vbolt.Tx, req AddUserRequest, hash []byte) User {
	var user User
	user.Id = vbolt.NextIntId(tx, UsersBucket)
	user.Email = req.Email

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

func AddUser(readTx *vbolt.Tx, req AddUserRequest) (err error) {
	err = ValidateUserTx(readTx, req)
	if err != nil {
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	vbolt.WithWriteTx(db, func(tx *vbolt.Tx) {
		AddUserTx(tx, req, hash)
		vbolt.TxCommit(tx)
	})
	return
}

func RegisterLoginPages(mux *http.ServeMux) {
	mux.HandleFunc("GET /login", loginPage)
	mux.HandleFunc("GET /logout", logout)
	mux.HandleFunc("POST /login", authenticateLogin)
	mux.HandleFunc("GET /register", registerPage)
	mux.HandleFunc("POST /register", createUser)
	mux.Handle("GET /profile/{id}", AuthHandler(http.HandlerFunc(profilePage)))
}

func loginPage(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, "login")
}

func registerPage(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, "register")
}

func profilePage(w http.ResponseWriter, r *http.Request) {
	authenticateUser(w, r)
	RenderTemplate(w, "profile")
}

func createUser(w http.ResponseWriter, r *http.Request) {
	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		addUserRequest := AddUserRequest{
			Email:    r.PostFormValue("email"),
			Password: r.PostFormValue("password"),
		}
		err := AddUser(tx, addUserRequest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		http.Redirect(w, r, "/", http.StatusFound)
	})
}

func authenticateLogin(w http.ResponseWriter, r *http.Request) {
	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		var userId int
		var user User
		var passHash []byte
		email := r.FormValue("email")
		vbolt.Read(tx, EmailBucket, email, &userId)
		vbolt.Read(tx, UsersBucket, userId, &user)
		vbolt.Read(tx, PasswordBucket, userId, &passHash)

		err := bcrypt.CompareHashAndPassword(passHash, []byte(r.FormValue("password")))

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
				http.Error(w, "Error generating token", http.StatusInternalServerError)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:     "auth_token",
				Value:    tokenString,
				Path:     "/",
				HttpOnly: true,
			})

			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
	})

	http.Error(w, "Invalid credentials", http.StatusUnauthorized)
}

func logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Unix(0, 0),
	})

	http.Redirect(w, r, "/", http.StatusFound)
}
