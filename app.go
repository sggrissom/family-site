package family

import (
	"family/backend"
	"family/cfg"
	"family/db"

	"go.hasen.dev/vbeam"
	"go.hasen.dev/vbolt"
)

func OpenDB(dbpath string) *vbolt.DB {
	dbConnection := vbolt.Open(dbpath)
	vbolt.InitBuckets(dbConnection, &db.Info)
	return dbConnection
}

func MakeApplication() *vbeam.Application {
	vbeam.RunBackServer(cfg.Backport)
	dbConnection := OpenDB(cfg.DBPath)
	var app = vbeam.NewApplication("FamilySite", dbConnection)
	backend.SetupAuth(app)
	backend.RegisterUserMethods(app)
	backend.RegisterPersonMethods(app)
	backend.RegisterFamilyMethods(app)
	return app
}
