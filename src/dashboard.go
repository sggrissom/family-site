package main

import (
	"net/http"

	"go.hasen.dev/vbolt"
)

func RegisterDashboardPages(mux *http.ServeMux) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		authenticateUser(w, r)
		username := w.Header().Get("username")
		if username == "" {
			RenderNoBaseTemplate(w, "welcome")
		}

		vbolt.WithReadTx(db, func(tx *vbolt.Tx) {
			families := GetAllFamilies(tx)

			if len(families) > 0 {
				RenderTemplate(w, "dashboard")
			} else {
				RenderTemplate(w, "landing")
			}
		})

	})
}
