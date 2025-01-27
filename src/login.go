package main

import (
	"net/http"
)

func RegisterLoginPages(mux *http.ServeMux) {
	mux.HandleFunc("GET /login", loginPage)
}

func loginPage(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, "login", nil)
}
