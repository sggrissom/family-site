package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.hasen.dev/vbolt"
)

const dbFile = "family.db"

var db *vbolt.DB
var Info vbolt.Info // define once

func RenderTemplate(w http.ResponseWriter, templateName string, data map[string]interface{}) {
	funcMap := template.FuncMap{
		"formatDate": func(t time.Time) string {
			if t.Year() == 1 {
				return ""
			}
			return t.Format("Jan 2, 2006")
		},
		"formatDateForInput": func(t time.Time) string {
			if t.Year() == 1 {
				return ""
			}
			return t.Format("2006-01-02")
		},
		"formatNumber": func(n float64) string {
			return fmt.Sprintf("%.2f", n)
		},
		"formatAge": func(age float64) string {
			if age == 0 {
				return "birth"
			}

			years := int(age)
			months := int((age - float64(years)) * 12)

			parts := []string{}
			if years > 0 {
				parts = append(parts, fmt.Sprintf("%d years", years))
			}
			if months > 0 {
				parts = append(parts, fmt.Sprintf("%d months", months))
			}
			if len(parts) == 0 {
				return "< 1 month"
			}
			return strings.Join(parts, ", ")
		},
		"displayHtml": func(content string) template.HTML {
			return template.HTML(content)
		},
	}

	tmpl := template.Must(template.New("base.html").Funcs(funcMap).ParseFiles(
		"html/base.html",
		"html/"+templateName+".html",
	))

	err := tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func isUserAuthenticated(w http.ResponseWriter, r *http.Request) bool {
	// Retrieve the JWT from the cookie
	cookie, err := r.Cookie("auth_token")
	if err != nil {
		return false
	}

	// Parse and validate the JWT
	token, err := jwt.ParseWithClaims(cookie.Value, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return jwtKey, nil
	})

	if err != nil || !token.Valid {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}

	return true
}

func RenderTemplateBlock(w http.ResponseWriter, templateName string, blockName string, data interface{}) {
	var template = template.Must(template.ParseFiles("html/base.html", "html/"+templateName+".html"))
	err := template.ExecuteTemplate(w, blockName, data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func AuthHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isUserAuthenticated(w, r) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	fmt.Println("family site starting")

	// Initialize the database
	db = vbolt.Open(dbFile)
	vbolt.InitBuckets(db, &Info)
	defer db.Close()

	mux := &Mux{
		family: http.NewServeMux(),
		maia:   http.NewServeMux(),
	}

	mux.family.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.family.Handle("/maia/", http.StripPrefix("/maia/", http.FileServer(http.Dir("../maia/html"))))
	mux.maia.Handle("/", http.FileServer(http.Dir("../maia/html")))

	RegisterMeasurementsPages(mux.family)
	RegisterChildrenPage(mux.family)
	RegisterPostPages(mux.family)
	RegisterLoginPages(mux.family)

	// HTTP to HTTPS redirect handler
	go func() {
		log.Fatal(http.ListenAndServe(":8665", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host := strings.Split(r.Host, ":")[0]
			targetURL := "https://" + host + r.URL.Path
			http.Redirect(w, r, targetURL, http.StatusMovedPermanently)
		})))
	}()

	// HTTPS server
	mux.family.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		RenderTemplate(w, "home", map[string]interface{}{})
	})

	useTLS := flag.Bool("tls", false, "Enable TLS (HTTPS)")
	flag.Parse()

	addr := "localhost:8666"
	log.Printf("Starting server on %s\n", addr)

	// Conditionally enable TLS
	if *useTLS {
		log.Println("TLS enabled. Using cert.pem and privkey.pem")
		log.Fatal(http.ListenAndServeTLS(addr, "cert.pem", "privkey.pem", mux))
	} else {
		log.Println("TLS disabled. Running HTTP only")
		log.Fatal(http.ListenAndServe(addr, mux))
	}
}

type Mux struct {
	family, maia *http.ServeMux
}

func (mux *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	domainParts := strings.Split(r.Host, ".")
	if domainParts[0] == "maia" {
		mux.maia.ServeHTTP(w, r)
	} else {
		mux.family.ServeHTTP(w, r)
	}
}
