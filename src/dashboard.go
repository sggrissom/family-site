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
	if context.username == "" {
		RenderNoBaseTemplate(context, "welcome")
	}

	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		families := GetFamiliesForUser(tx, context.userId)

		if len(families) > 0 {
			RenderTemplateWithData(context, "dashboard", map[string]any{
				"Family": families[0],
			})
		} else {
			RenderTemplate(context, "landing")
		}
	})
}

func familiesPage(context ResponseContext) {
	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		families := GetFamiliesForUser(tx, context.userId)

		if len(families) > 0 {
			RenderTemplateWithData(context, "families", map[string]any{
				"Families": families,
			})
		} else {
			RenderTemplate(context, "landing")
		}
	})
}
