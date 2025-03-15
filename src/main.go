package main

import (
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.hasen.dev/vbolt"
)

const dbFile = "family.db"

var db *vbolt.DB
var Info vbolt.Info // define once

type ResponseContext struct {
	w        http.ResponseWriter
	r        *http.Request
	userId   int
	username string
	isAdmin  bool
}

type ContextFunc func(ResponseContext)

var funcMap = template.FuncMap{
	"formatDate": func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.Format("Jan 2, 2006")
	},
	"formatDateForInput": func(t time.Time) string {
		if t.IsZero() {
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
	"formatMilestoneType": func(milestoneType MilestoneType) string {
		return parseMilestoneTypeLabel(milestoneType)
	},
}

var templatePaths map[string]string
var adminPaths map[string]string

func preloadTemplates() error {
	templatePaths = make(map[string]string)
	adminPaths = make(map[string]string)

	files, err := filepath.Glob("html/**/*.html")
	if err != nil {
		return err
	}

	for _, file := range files {
		base := strings.TrimSuffix(filepath.Base(file), ".html")
		templatePaths[base] = file
	}

	adminFiles, err := filepath.Glob("html/admin/*.html")
	if err != nil {
		return err
	}

	for _, file := range adminFiles {
		base := strings.TrimSuffix(filepath.Base(file), ".html")
		adminPaths[base] = file
	}

	return nil
}

func RenderNoBaseTemplate(context ResponseContext, templateName string) {
	RenderNoBaseTemplateWithData(context, templateName, map[string]any{})
}

func RenderNoBaseTemplateWithData(context ResponseContext, templateName string, data map[string]interface{}) {
	internalRenderTemplateWithData(context, []string{templateName}, data)
}

func RenderTemplate(context ResponseContext, templateName string) {
	RenderTemplateWithData(context, templateName, map[string]any{})
}

func RenderTemplateWithData(context ResponseContext, templateName string, data map[string]interface{}) {
	internalRenderTemplateWithData(context, []string{"base", templateName}, data)
}

var ErrInvalidTemplate = errors.New("InvalidTemplate")

func getTemplatePaths(templateNames []string) ([]string, error) {
	paths := make([]string, len(templateNames))
	for index, templateName := range templateNames {
		path, exists := templatePaths[templateName]
		if !exists {
			return nil, ErrInvalidTemplate
		}
		paths[index] = path
	}

	return paths, nil
}
func internalRenderTemplateWithData(context ResponseContext, templateNames []string, data map[string]interface{}) {
	paths, err := getTemplatePaths(templateNames)
	if err != nil {
		log.Printf("Template failure: %v", err)
		http.Error(context.w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl := template.New(filepath.Base(paths[0])).Funcs(funcMap)
	for _, path := range paths {
		tmpl, err = tmpl.ParseFiles(path)
		if err != nil {
			log.Printf("Template failure: %v", err)
			http.Error(context.w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if context.username != "" {
		data["Username"] = context.username
		if context.isAdmin {
			data["isAdmin"] = true
		}
	}

	err = tmpl.Execute(context.w, data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(context.w, err.Error(), http.StatusInternalServerError)
	}
}

func RenderAdminTemplate(context ResponseContext, templateName string) {
	RenderAdminTemplateWithData(context, templateName, map[string]any{})
}

func RenderAdminTemplateWithData(context ResponseContext, templateName string, data map[string]interface{}) {
	path, exists := adminPaths[templateName]
	if !exists {
		log.Printf("Template not found: %v", templateName)
		http.Error(context.w, "Template not found", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("admin.html").Funcs(funcMap).ParseFiles("html/admin/admin.html", path)
	if err != nil {
		log.Printf("Template failure: %v", err)
		http.Error(context.w, err.Error(), http.StatusInternalServerError)
		return
	}

	if context.username == "" {
		http.Error(context.w, "auth failure", http.StatusInternalServerError)
		return
	}
	data["Username"] = context.username
	data["UserId"] = context.userId
	if context.isAdmin {
		data["isAdmin"] = true
	} else {
		http.Error(context.w, "user not an admin", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(context.w, data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(context.w, err.Error(), http.StatusInternalServerError)
	}
}

func BuildResponseContext(w http.ResponseWriter, r *http.Request) (context ResponseContext) {
	context.w = w
	context.r = r

	cookie, err := r.Cookie("auth_token")
	if err != nil {
		return
	}

	token, err := jwt.ParseWithClaims(cookie.Value, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return jwtKey, nil
	})

	if err != nil || !token.Valid {
		return
	}

	if claims, ok := token.Claims.(*Claims); ok {
		context.username = claims.Username
		vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
			context.userId = GetUserId(tx, context.username)
			context.isAdmin = context.userId == 1
		})
	}

	return
}

func RenderTemplateBlock(context ResponseContext, templateName string, blockName string, data interface{}) {
	var template = template.Must(template.ParseFiles("html/base.html", "html/"+templateName+".html"))
	err := template.ExecuteTemplate(context.w, blockName, data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(context.w, err.Error(), http.StatusInternalServerError)
	}
}

func PublicHandler(next ContextFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		context := BuildResponseContext(w, r)
		next(context)
	})
}

func AuthHandler(next ContextFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		context := BuildResponseContext(w, r)
		if context.username == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next(context)
	})
}

func main() {
	fmt.Println("family site starting")

	if preloadTemplates() != nil {
		log.Fatal("error preloading templates")
	}

	db = vbolt.Open(dbFile)
	vbolt.InitBuckets(db, &Info)

	// migrations
	vbolt.ApplyDBProcess(db, "2025-0307-reset-user", func() {
		vbolt.WithWriteTx(db, func(tx *vbolt.Tx) {
			tx.DeleteBucket([]byte(UsersBucket.Name))
			tx.DeleteBucket([]byte(PasswordBucket.Name))
			tx.DeleteBucket([]byte(EmailBucket.Name))
			tx.CreateBucket([]byte(UsersBucket.Name))
			tx.CreateBucket([]byte(PasswordBucket.Name))
			tx.CreateBucket([]byte(EmailBucket.Name))
			vbolt.TxCommit(tx)
		})
	})

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
	RegisterMilestonesPages(mux.family)
	RegisterAdminPages(mux.family)
	RegisterDashboardPages(mux.family)

	// HTTP to HTTPS redirect handler
	go func() {
		log.Fatal(http.ListenAndServe(":8665", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host := strings.Split(r.Host, ":")[0]
			targetURL := "https://" + host + r.URL.Path
			http.Redirect(w, r, targetURL, http.StatusMovedPermanently)
		})))
	}()

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
