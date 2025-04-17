package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"time"

	"github.com/boltdb/bolt"
	"github.com/golang-jwt/jwt/v5"
	"go.hasen.dev/generic"
	"go.hasen.dev/vbolt"
	"go.hasen.dev/vpack"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var jwtKey []byte
var oauthConf *oauth2.Config
var oauthStateString string

// Models

type StatusType int

const (
	Active StatusType = iota
	Suspended
)

type User struct {
	Id              int
	Email           string
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

// token => user id
var ResetPasswordBucket = vbolt.Bucket(&Info, "password-token", vpack.String, vpack.FInt)

// token => user id
var RefreshBucket = vbolt.Bucket(&Info, "login-token", vpack.String, vpack.FInt)

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
var ErrInvalidToken = errors.New("InvalidToken")
var ErrNoUser = errors.New("NoUser")

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

func GetUserIdFromToken(tx *vbolt.Tx, token string) (userId int) {
	vbolt.Read(tx, ResetPasswordBucket, token, &userId)
	return userId
}

func GetUserIdFromRefreshToken(tx *vbolt.Tx, token string) (userId int) {
	vbolt.Read(tx, RefreshBucket, token, &userId)
	vbolt.Delete(tx, RefreshBucket, token)
	return userId
}

func AddUserTx(tx *vbolt.Tx, req AddUserRequest, hash []byte) User {
	var user User
	user.Id = vbolt.NextIntId(tx, UsersBucket)
	user.Email = req.Email
	user.Status = Active
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

func DeleteUser(dbHandle *bolt.DB, userId int) (err error) {
	var user User
	vbolt.WithReadTx(dbHandle, func(tx *vbolt.Tx) {
		user = GetUser(tx, userId)
	})
	vbolt.WithWriteTx(dbHandle, func(tx *vbolt.Tx) {
		vbolt.Delete(tx, UsersBucket, user.Id)
		vbolt.Delete(tx, PasswordBucket, user.Id)
		vbolt.Delete(tx, EmailBucket, user.Email)
		vbolt.TxCommit(tx)
	})
	return
}

func ResetUser(dbHandle *bolt.DB, token string, req AddUserRequest) (err error) {
	if !isPasswordValid(req.Password) {
		return ErrPasswordInvalid
	}

	var user User
	vbolt.WithReadTx(dbHandle, func(tx *vbolt.Tx) {
		userId := GetUserIdFromToken(tx, token)
		user = GetUser(tx, userId)
	})

	if user.Id == 0 {
		return ErrInvalidToken
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	vbolt.WithWriteTx(dbHandle, func(tx *vbolt.Tx) {
		vbolt.Write(tx, PasswordBucket, user.Id, &hash)
		vbolt.Delete(tx, ResetPasswordBucket, token)
		vbolt.TxCommit(tx)
	})
	return
}

func RegisterLoginPages(mux *http.ServeMux) {
	mux.Handle("GET /login", PublicHandler(ContextFunc(loginPage)))
	mux.Handle("GET /login/google", PublicHandler(ContextFunc(googleLogin)))
	mux.Handle("GET /google/callback", PublicHandler(ContextFunc(googleCallback)))
	mux.Handle("GET /logout", PublicHandler(ContextFunc(logout)))
	mux.Handle("POST /login", PublicHandler(ContextFunc(authenticateLogin)))
	mux.Handle("GET /register", PublicHandler(ContextFunc(registerPage)))
	mux.Handle("POST /register", PublicHandler(ContextFunc(createUser)))
	mux.Handle("GET /profile", AuthHandler(ContextFunc(profilePage)))
	mux.Handle("GET /forgot", PublicHandler(ContextFunc(forgotEmail)))
	mux.Handle("GET /reset-password-sent", PublicHandler(ContextFunc(resetEmailSent)))
	mux.Handle("GET /reset-password", PublicHandler(ContextFunc(resetPassword)))
	mux.Handle("POST /reset-password", PublicHandler(ContextFunc(resetPasswordPost)))

	oauthConf = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("SITE_ROOT") + "/google/callback",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
	token, err := generateToken(20)
	if err != nil {
		log.Fatal("error generating oauth token")
	}
	oauthStateString = token

	jwtKey = []byte(os.Getenv("JWT_SECRET_KEY"))
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
	honeypot := context.r.PostFormValue("honeypot")
	if honeypot != "" {
		http.Error(context.w, "invalid submission", http.StatusUnauthorized)
		return
	}

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

func generateAuthJwt(user User, w http.ResponseWriter) (err error) {
	expirationTime := time.Now().Add(15 * time.Minute)
	claims := &Claims{
		Username: user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   60 * 15,
	})

	vbolt.WithWriteTx(db, func(writeTx *vbolt.Tx) {
		user.LastLogin = time.Now()
		vbolt.Write(writeTx, UsersBucket, user.Id, &user)
		vbolt.TxCommit(writeTx)
	})

	return nil
}

func generateAuthRefreshToken(userId int, w http.ResponseWriter) (err error) {
	refreshToken, err := generateToken(20)
	if err != nil {
		return
	}
	saveRefreshToken(refreshToken, userId)

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   60 * 60 * 24 * 30,
	})
	return nil
}

func authenticateForUser(userId int, w http.ResponseWriter) (err error) {
	var user User
	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		user = GetUser(tx, userId)
	})

	if user.Id == 0 {
		return ErrNoUser
	}

	err = generateAuthJwt(user, w)
	if err != nil {
		return
	}
	err = generateAuthRefreshToken(userId, w)
	return
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
			err = generateAuthJwt(user, context.w)
			if err != nil {
				http.Error(context.w, "Error generating token", http.StatusInternalServerError)
				return
			}
			err = generateAuthRefreshToken(userId, context.w)
			if err != nil {
				http.Error(context.w, "Error generating token", http.StatusInternalServerError)
				return
			}

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

	cookie, err := context.r.Cookie("refresh_token")
	if err == nil {
		vbolt.WithWriteTx(db, func(tx *vbolt.Tx) {
			vbolt.Delete(tx, RefreshBucket, cookie.Value)
			tx.Commit()
		})
		http.SetCookie(context.w, &http.Cookie{
			Name:     "refresh_token",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Expires:  time.Unix(0, 0),
		})
	}

	http.Redirect(context.w, context.r, "/", http.StatusFound)
}

func generateToken(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func saveToken(token string, email string) {
	var userId int
	vbolt.WithReadTx(db, func(readTx *vbolt.Tx) {
		userId = GetUserId(readTx, email)
	})
	vbolt.WithWriteTx(db, func(writeTx *vbolt.Tx) {
		vbolt.Write(writeTx, ResetPasswordBucket, token, &userId)
		writeTx.Commit()
	})
}

func saveRefreshToken(token string, userId int) {
	vbolt.WithWriteTx(db, func(writeTx *vbolt.Tx) {
		vbolt.Write(writeTx, RefreshBucket, token, &userId)
		writeTx.Commit()
	})
}

func forgotEmail(context ResponseContext) {
	accountEmail := context.r.URL.Query().Get("email")
	token, err := generateToken(20)
	if err != nil {
		http.Error(context.w, err.Error(), http.StatusUnprocessableEntity)
	}
	saveToken(token, accountEmail)

	email := os.Getenv("EMAIL")
	appPassword := os.Getenv("APP_PASSWORD")
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"
	recipient := accountEmail

	resetLink := os.Getenv("SITE_ROOT") + "/reset-password?token=" + token

	message := []byte("Subject: Reset Your Password\r\n" +
		"\r\n" +
		"Hello,\r\n\r\n" +
		"To reset your password, please click the link below:\r\n" +
		resetLink + "\r\n\r\n" +
		"If you did not request a password reset, please ignore this email.\r\n")

	auth := smtp.PlainAuth("", email, appPassword, smtpHost)

	err = smtp.SendMail(smtpHost+":"+smtpPort, auth, email, []string{recipient}, message)
	if err != nil {
		log.Fatalf("Failed to send email: %v", err)
	}

	http.Redirect(context.w, context.r, "/reset-password-sent", http.StatusFound)
}

func resetEmailSent(context ResponseContext) {
	RenderTemplate(context, "forgot-password-sent")
}

func resetPassword(context ResponseContext) {
	RenderTemplateWithData(context, "reset-password", map[string]any{
		"Token": context.r.URL.Query().Get("token"),
	})
}

func resetPasswordPost(context ResponseContext) {
	token := context.r.FormValue("token")
	addUserRequest := AddUserRequest{
		Password: context.r.PostFormValue("password"),
	}
	err := ResetUser(db, token, addUserRequest)
	if err != nil {
		http.Error(context.w, err.Error(), http.StatusUnauthorized)
	}

	http.Redirect(context.w, context.r, "/login", http.StatusFound)
}

func googleLogin(context ResponseContext) {
	url := oauthConf.AuthCodeURL(oauthStateString, oauth2.AccessTypeOffline)
	http.Redirect(context.w, context.r, url, http.StatusTemporaryRedirect)
}

type UserInfo struct {
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

func googleCallback(ctx ResponseContext) {
	if ctx.r.FormValue("state") != oauthStateString {
		http.Error(ctx.w, "Invalid OAuth state", http.StatusBadRequest)
		return
	}

	code := ctx.r.FormValue("code")
	token, err := oauthConf.Exchange(context.Background(), code)
	if err != nil {
		http.Error(ctx.w, fmt.Sprintf("Code exchange failed: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	client := oauthConf.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		http.Error(ctx.w, fmt.Sprintf("Failed getting user info: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		http.Error(ctx.w, fmt.Sprintf("Failed decoding user info: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	var userId int
	vbolt.WithReadTx(db, func(readTx *vbolt.Tx) {
		userId = GetUserId(readTx, userInfo.Email)
	})
	if userId > 0 {
		authenticateForUser(userId, ctx.w)
	} else {
		addUserRequest := AddUserRequest{
			Email:     userInfo.Email,
			FirstName: userInfo.GivenName,
			LastName:  userInfo.FamilyName,
		}

		var user User
		vbolt.WithWriteTx(db, func(tx *vbolt.Tx) {
			user = AddUserTx(tx, addUserRequest, []byte{})
			vbolt.TxCommit(tx)
		})
		if user.Id > 0 {
			authenticateForUser(user.Id, ctx.w)
		}
	}

	http.Redirect(ctx.w, ctx.r, "/", http.StatusFound)
}
