package main

import (
	"net/http"
)

func RegisterMilestonesPages(mux *http.ServeMux) {
	mux.Handle("GET /milestones", PublicHandler(http.HandlerFunc(milestonesPage)))
	mux.Handle("GET /milestones/add", AuthHandler(http.HandlerFunc(addMilestonesPage)))
	mux.Handle("GET /milestones/edit/{id}", AuthHandler(http.HandlerFunc(editPostPage)))
	mux.Handle("GET /milestones/delete/{id}", AuthHandler(http.HandlerFunc(deletePost)))
	mux.Handle("POST /milestones/add", AuthHandler(http.HandlerFunc(savePost)))
}

func milestonesPage(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, "milestones")
}

func addMilestonesPage(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, "milestones")
}
