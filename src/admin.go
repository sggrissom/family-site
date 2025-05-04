package main

import (
	"net/http"
	"slices"
	"strconv"
	"strings"

	"go.hasen.dev/vbolt"
)

func RegisterAdminPages(mux *http.ServeMux) {
	mux.Handle("GET /admin", AdminHandler(ContextFunc(adminPage)))
	mux.Handle("GET /admin/users", AdminHandler(ContextFunc(usersAdminPage)))
	mux.Handle("GET /admin/families", AdminHandler(ContextFunc(familiesAdminPage)))
	mux.Handle("GET /admin/people", AdminHandler(ContextFunc(peopleAdminPage)))
	mux.Handle("GET /admin/user/delete/{id}", AdminHandler(ContextFunc(deleteUserId)))
	mux.Handle("GET /admin/user/delete", AdminHandler(ContextFunc(deleteUsersBulk)))
	mux.Handle("GET /admin/user/make-owner/{id}", AdminHandler(ContextFunc(makeUserOwner)))
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

func deleteUsersBulk(context ResponseContext) {
	idsString := context.r.URL.Query().Get("ids")
	idValues := strings.Split(idsString, ",")

	for _, id := range idValues {
		if id == "" {
			continue
		}
		idVal, err := strconv.Atoi(strings.TrimSpace(id))
		if err != nil {
			http.Error(context.w, "Invalid ID", http.StatusBadRequest)
			return
		}
		DeleteUser(db, idVal)
	}

	http.Redirect(context.w, context.r, "/admin/users", http.StatusFound)
}

func makeUserOwner(context ResponseContext) {
	id := context.r.PathValue("id")
	userId, _ := strconv.Atoi(id)

	id = context.r.URL.Query().Get("familyId")
	familyId, _ := strconv.Atoi(id)

	var family Family
	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		family = getFamily(tx, familyId)
	})
	if !slices.Contains(family.OwningUsers, userId) {
		family.OwningUsers = append(family.OwningUsers, userId)
		vbolt.WithWriteTx(db, func(tx *vbolt.Tx) {
			vbolt.Write(tx, FamilyBucket, family.Id, &family)
			tx.Commit()
		})
	}

	http.Redirect(context.w, context.r, "/admin/users", http.StatusFound)
}
