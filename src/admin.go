package main

import (
	"net/http"

	"go.hasen.dev/vbolt"
)

func RegisterAdminPages(mux *http.ServeMux) {
	mux.Handle("GET /admin", AuthHandler(ContextFunc(adminPage)))
	mux.Handle("GET /admin/users", AuthHandler(ContextFunc(usersPage)))
	mux.Handle("GET /admin/families", AuthHandler(ContextFunc(familiesPage)))
}

func adminPage(context ResponseContext) {
	RenderAdminTemplate(context, "home")
}

func usersPage(context ResponseContext) {
	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		RenderAdminTemplateWithData(context, "users", map[string]any{
			"Users": GetAllUsers(tx),
		})
	})
}

func familiesPage(context ResponseContext) {
	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		RenderAdminTemplateWithData(context, "families", map[string]any{
			"Families": GetAllFamilies(tx),
		})
	})
}
