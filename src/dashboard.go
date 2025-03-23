package main

import (
	"net/http"
	"strconv"

	"github.com/boltdb/bolt"
	"go.hasen.dev/vbolt"
)

func RegisterDashboardPages(mux *http.ServeMux) {
	mux.Handle("/", PublicHandler(ContextFunc(rootPage)))
	mux.Handle("GET /family-list", AuthHandler(ContextFunc(familiesPage)))
	mux.Handle("GET /family/favorite/{id}", AuthHandler(ContextFunc(favoriteFamily)))
}

func rootPage(context ResponseContext) {
	if context.user.Id == 0 {
		RenderNoBaseTemplate(context, "welcome")
		return
	}

	if context.user.PrimaryFamilyId > 0 {
		vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
			family := getFamily(tx, context.user.PrimaryFamilyId)
			people := getPeopleInFamily(tx, family.Id)
			RenderTemplateWithData(context, "dashboard", map[string]any{
				"Family": family,
				"People": people,
			})
		})
	} else {
		familiesPage(context)
	}
}

func familiesPage(context ResponseContext) {
	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		families := GetFamiliesForUser(tx, context.user.Id)

		if len(families) > 0 {
			RenderTemplateWithData(context, "families", map[string]any{
				"Families": families,
			})
		} else {
			RenderTemplate(context, "landing")
		}
	})
}

func favoriteFamily(context ResponseContext) {
	var user User
	vbolt.WithReadTx(db, func(tx *bolt.Tx) {
		user = GetUser(tx, context.user.Id)
	})

	vbolt.WithWriteTx(db, func(tx *bolt.Tx) {
		id := context.r.PathValue("id")
		idVal, _ := strconv.Atoi(id)
		user.PrimaryFamilyId = idVal
		vbolt.Write(tx, UsersBucket, context.user.Id, &user)
		vbolt.TxCommit(tx)
	})

	http.Redirect(context.w, context.r, "/", http.StatusFound)
}
