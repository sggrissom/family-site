package backend

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.hasen.dev/vbeam"
	"go.hasen.dev/vbolt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var jwtKey []byte
var oauthConf *oauth2.Config
var oauthStateString string

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func SetupOauth() {
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

func generateToken(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func generateAuthJwt(ctx *vbeam.Context, user User) (newToken string, err error) {
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

	vbeam.UseWriteTx(ctx)
	user.LastLogin = time.Now()
	vbolt.Write(ctx.Tx, UsersBkt, user.Id, &user)
	vbolt.TxCommit(ctx.Tx)

	newToken = tokenString
	return
}
