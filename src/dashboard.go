package main

import (
	"net/http"

	"go.hasen.dev/vbolt"
)

func RegisterDashboardPages(mux *http.ServeMux) {
	mux.Handle("/", PublicHandler(ContextFunc(rootPage)))
	mux.Handle("GET /family-list", AuthHandler(ContextFunc(familiesPage)))
}

func rootPage(context ResponseContext) {
	if context.user.Id == 0 {
		RenderNoBaseTemplate(context, "welcome")
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
