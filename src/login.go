package main

import (
	"net/http"
)

func RegisterLoginPages(mux *http.ServeMux) {
	mux.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
		RenderTemplate(w, "login", nil)
	})
}
