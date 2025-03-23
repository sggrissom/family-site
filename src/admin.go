package main

import (
	"net/http"
	"strconv"

	"go.hasen.dev/vbolt"
)

func RegisterAdminPages(mux *http.ServeMux) {
	mux.Handle("GET /admin", AdminHandler(ContextFunc(adminPage)))
	mux.Handle("GET /admin/users", AdminHandler(ContextFunc(usersAdminPage)))
	mux.Handle("GET /admin/families", AdminHandler(ContextFunc(familiesAdminPage)))
	mux.Handle("GET /admin/people", AdminHandler(ContextFunc(peopleAdminPage)))
	mux.Handle("GET /admin/user/delete/{id}", AdminHandler(ContextFunc(deleteUserId)))
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

func deleteUserId(context ResponseContext) {
	id := context.r.PathValue("id")
	idVal, _ := strconv.Atoi(id)
	DeleteUser(db, idVal)

	http.Redirect(context.w, context.r, "/admin/users", http.StatusFound)
}
