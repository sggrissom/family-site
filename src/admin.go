package main

import "net/http"

func RegisterAdminPages(mux *http.ServeMux) {
	mux.Handle("GET /admin", AuthHandler(http.HandlerFunc(adminPage)))
	mux.Handle("GET /admin/users", AuthHandler(http.HandlerFunc(usersPage)))
}

func adminPage(w http.ResponseWriter, r *http.Request) {
	RenderAdminTemplate(w, r, "home")
}

func usersPage(w http.ResponseWriter, r *http.Request) {
	RenderAdminTemplate(w, r, "users")
}
