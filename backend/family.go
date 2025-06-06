package backend

import (
	"family/db"

	"go.hasen.dev/generic"
	"go.hasen.dev/vbeam"
	"go.hasen.dev/vbolt"
	"go.hasen.dev/vpack"
)

func RegisterFamilyMethods(app *vbeam.Application) {
	vbeam.RegisterProc(app, AddFamily)
	vbeam.RegisterProc(app, ListFamilies)
	vbeam.RegisterProc(app, GetFamilyInfo)
}

type Empty struct{}

type Family struct {
	Id          int
	Name        string
	Description string
	CreatorId   int
}

func PackFamily(self *Family, buf *vpack.Buffer) {
	vpack.Version(1, buf)
	vpack.Int(&self.Id, buf)
	vpack.String(&self.Name, buf)
	vpack.String(&self.Description, buf)
	vpack.Int(&self.CreatorId, buf)
}

var FamilyBucket = vbolt.Bucket(&db.Info, "family", vpack.FInt, PackFamily)

func GetAllFamilies(tx *vbolt.Tx) (families []Family) {
	vbolt.IterateAll(tx, FamilyBucket, func(key int, value Family) bool {
		generic.Append(&families, value)
		return true
	})
	return families
}

func AddFamilyTx(tx *vbolt.Tx, req AddFamilyRequest, creatorId int) (family Family) {
	if req.Id > 0 {
		family.Id = req.Id
	} else {
		family.Id = vbolt.NextIntId(tx, FamilyBucket)
	}
	family.Name = req.Name
	family.Description = req.Description
	family.CreatorId = creatorId

	vbolt.Write(tx, FamilyBucket, family.Id, &family)
	return
}

type AddFamilyRequest struct {
	Id          int
	Name        string
	Description string
}

type FamilyListResponse struct {
	AllFamilyNames []string
}

type FamilyDataResponse struct {
	AuthUserId int
	Family     Family
	Members    []Person
}

func AddFamily(ctx *vbeam.Context, req AddFamilyRequest) (resp FamilyListResponse, err error) {
	user, err := GetAuthUser(ctx)
	if err != nil {
		return
	}
	if user.Id == 0 {
		return
	}
	vbeam.UseWriteTx(ctx)
	AddFamilyTx(ctx.Tx, req, user.Id)
	vbolt.TxCommit(ctx.Tx)

	resp.AllFamilyNames = []string{}

	return
}

func ListFamilies(ctx *vbeam.Context, req Empty) (resp FamilyListResponse, err error) {
	allFamilies := GetAllFamilies(ctx.Tx)
	resp.AllFamilyNames = []string{}
	for _, family := range allFamilies {
		resp.AllFamilyNames = append(resp.AllFamilyNames, family.Name)
	}
	return
}

func GetFamilyInfo(ctx *vbeam.Context, req Empty) (resp FamilyDataResponse, err error) {
	user, authErr := GetAuthUser(ctx)
	if authErr != nil {
		return
	}
	if user.Id == 0 {
		return
	}
	resp.AuthUserId = user.Id
	allFamilies := GetAllFamilies(ctx.Tx)
	if len(allFamilies) < 1 {
		resp.Members = []Person{}
		return
	}
	resp.Family = allFamilies[0]
	resp.Members = GetAllPeople(ctx.Tx)
	return
}
