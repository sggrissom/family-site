package family

import (
	"family/backend"
	"family/cfg"

	"go.hasen.dev/vbeam"
	"go.hasen.dev/vbolt"
)

func MakeApplication() *vbeam.Application {
	vbeam.RunBackServer(cfg.Backport)
	db := vbolt.Open(cfg.DBPath)
	var app = vbeam.NewApplication("FamilySite", db)
	vbeam.RegisterProc(app, backend.AddFamily)
	vbeam.RegisterProc(app, backend.ListFamilies)
	return app
}
