package main

import "net/http"

func RegisterAdminPages(mux *http.ServeMux) {
	mux.Handle("GET /admin", AuthHandler(http.HandlerFunc(adminPage)))
}

func adminPage(w http.ResponseWriter, r *http.Request) {
	RenderAdminTemplate(w, r, "admin")
}
