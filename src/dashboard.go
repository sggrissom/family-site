package main

import (
	"net/http"

	"go.hasen.dev/vbolt"
)

func RegisterDashboardPages(mux *http.ServeMux) {
	mux.Handle("/", PublicHandler(ContextFunc(rootPage)))
}

func rootPage(context ResponseContext) {
	if context.username == "" {
		RenderNoBaseTemplate(context, "welcome")
	}

	vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
		families := GetAllFamilies(tx)

		if len(families) > 0 {
			RenderTemplate(context, "dashboard")
		} else {
			RenderTemplate(context, "landing")
		}
	})
}
