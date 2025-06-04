package backend

import (
	"family/db"
	"time"

	"go.hasen.dev/generic"
	"go.hasen.dev/vbeam"
	"go.hasen.dev/vbolt"
	"go.hasen.dev/vpack"
)

func RegisterPersonMethods(app *vbeam.Application) {
	vbeam.RegisterProc(app, AddPerson)
	vbeam.RegisterProc(app, ListPeople)
}

type GenderType int

const (
	Male GenderType = iota
	Female
	Undisclosed
)

type PersonType int

const (
	Parent PersonType = iota
	Child
)

type Person struct {
	Id       int
	FamilyId int
	Type     PersonType
	Gender   GenderType
	Name     string
	Birthday time.Time
	Age      string
	ImageId  int
}

func PackPerson(self *Person, buf *vpack.Buffer) {
	vpack.Version(1, buf)
	vpack.Int(&self.Id, buf)
	vpack.String(&self.Name, buf)
	vpack.Time(&self.Birthday, buf)
	vpack.Int(&self.FamilyId, buf)
	vpack.IntEnum(&self.Type, buf)
	vpack.IntEnum(&self.Gender, buf)
	vpack.Int(&self.ImageId, buf)
}

var PersonBucket = vbolt.Bucket(&db.Info, "people", vpack.FInt, PackPerson)

func GetAllPeople(tx *vbolt.Tx) (people []Person) {
	vbolt.IterateAll(tx, PersonBucket, func(key int, value Person) bool {
		generic.Append(&people, value)
		return true
	})
	return people
}

func AddPersonTx(tx *vbolt.Tx, req AddPersonRequest, birthDate time.Time) (person Person) {
	if req.Id > 0 {
		person.Id = req.Id
	} else {
		person.Id = vbolt.NextIntId(tx, PersonBucket)
	}
	person.Type = PersonType(req.PersonType)
	person.Gender = GenderType(req.Gender)
	person.Name = req.Name

	person.Birthday = birthDate

	vbolt.Write(tx, PersonBucket, person.Id, &person)
	return
}

type AddPersonRequest struct {
	Id         int
	PersonType int
	Gender     int
	Birthdate  string
	Name       string
}

type PersonListResponse struct {
	AllPersonNames []string
}

func AddPerson(ctx *vbeam.Context, req AddPersonRequest) (resp Empty, err error) {
	layout := "2006-01-02"
	parsedTime, err := time.Parse(layout, req.Birthdate)
	if err != nil {
		return
	}

	vbeam.UseWriteTx(ctx)
	AddPersonTx(ctx.Tx, req, parsedTime)
	vbolt.TxCommit(ctx.Tx)

	return
}

func ListPeople(ctx *vbeam.Context, req Empty) (resp PersonListResponse, err error) {
	allPeople := GetAllPeople(ctx.Tx)
	for _, person := range allPeople {
		resp.AllPersonNames = append(resp.AllPersonNames, person.Name)
	}
	return
}
