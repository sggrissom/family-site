package main

import (
	"net/http"

	"go.hasen.dev/vbolt"
)

func RegisterAdminPages(mux *http.ServeMux) {
	mux.Handle("GET /admin", AuthHandler(http.HandlerFunc(adminPage)))
	mux.Handle("GET /admin/users", AuthHandler(http.HandlerFunc(usersPage)))
	mux.Handle("GET /admin/families", AuthHandler(http.HandlerFunc(familiesPage)))
}

func adminPage(w http.ResponseWriter, r *http.Request) {
	RenderAdminTemplate(w, r, "home")
}

func usersPage(w http.ResponseWriter, r *http.Request) {
	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		RenderAdminTemplateWithData(w, r, "users", map[string]any{
			"Users": GetAllUsers(tx),
		})
	})
}

func familiesPage(w http.ResponseWriter, r *http.Request) {
	RenderAdminTemplate(w, r, "families")
}
