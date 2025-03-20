package main

import (
	"net/http"

	"go.hasen.dev/vbolt"
)

func RegisterAdminPages(mux *http.ServeMux) {
	mux.Handle("GET /admin", AuthHandler(ContextFunc(adminPage)))
	mux.Handle("GET /admin/users", AuthHandler(ContextFunc(usersAdminPage)))
	mux.Handle("GET /admin/families", AuthHandler(ContextFunc(familiesAdminPage)))
	mux.Handle("GET /admin/people", AuthHandler(ContextFunc(peopleAdminPage)))
}

func adminPage(context ResponseContext) {
	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		RenderAdminTemplateWithData(context, "home", map[string]any{
			"Users":    GetAllUsers(tx),
			"People":   GetAllPeople(tx),
			"Families": GetAllFamilies(tx),
		})
	})
}

func usersAdminPage(context ResponseContext) {
	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		RenderAdminTemplateWithData(context, "users", map[string]any{
			"Users": GetAllUsers(tx),
		})
	})
}

func familiesAdminPage(context ResponseContext) {
	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		RenderAdminTemplateWithData(context, "families", map[string]any{
			"Families": GetAllFamilies(tx),
		})
	})
}

func peopleAdminPage(context ResponseContext) {
	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		RenderAdminTemplateWithData(context, "people", map[string]any{
			"People": GetAllPeople(tx),
		})
	})
}
