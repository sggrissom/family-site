package backend

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.hasen.dev/vbeam"
	"go.hasen.dev/vbolt"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var jwtKey []byte
var oauthConf *oauth2.Config
var oauthStateString string
var ErrLoginFailure = errors.New("LoginFailure")

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

var appDb *vbolt.DB

func SetupAuth(app *vbeam.Application) {
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

	app.HandleFunc("/api/login", loginHandler)
	app.HandleFunc("/api/logout", logoutHandler)

	appDb = app.DB
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		vbeam.RespondError(w, errors.New("login call must be POST"))
		return
	}

	var credentials AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		vbeam.RespondError(w, ErrLoginFailure)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var user User
	var passHash []byte

	vbolt.WithReadTx(appDb, func(tx *vbolt.Tx) {
		userId := GetUserId(tx, credentials.Email)
		if userId == 0 {
			json.NewEncoder(w).Encode(LoginResponse{Success: false})
			return
		}
		user = GetUser(tx, userId)
		passHash = GetPassHash(tx, userId)
	})

	err := bcrypt.CompareHashAndPassword(passHash, []byte(credentials.Password))
	if err != nil {
		json.NewEncoder(w).Encode(LoginResponse{Success: false})
		return
	}

	err = generateAuthJwt(user, w)
	if err != nil {
		json.NewEncoder(w).Encode(LoginResponse{Success: false})
		return
	}

	json.NewEncoder(w).Encode(LoginResponse{Success: true})
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "authToken",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Unix(0, 0),
	})
}

func generateToken(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
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
		Name:     "authToken",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   60 * 15,
	})

	vbolt.WithWriteTx(appDb, func(tx *vbolt.Tx) {
		user.LastLogin = time.Now()
		vbolt.Write(tx, UsersBkt, user.Id, &user)
		vbolt.TxCommit(tx)
	})

	return
}
