package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	"go.hasen.dev/generic"
	"go.hasen.dev/vbolt"
	"go.hasen.dev/vpack"
)

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

type VisibilityType int

const (
	Hidden VisibilityType = iota
	Public
)

func parseVisibilityLabel(t VisibilityType) string {
	switch t {
	case Hidden:
		return "hidden"
	case Public:
		return "public"
	default:
		return ""
	}
}

func parseVisibilityType(s string) (VisibilityType, error) {
	switch s {
	case "hidden":
		return Hidden, nil
	case "public":
		return Public, nil
	default:
		return 0, fmt.Errorf("unknown visibility type: %s", s)
	}
}

type Family struct {
	Id          int
	Name        string
	Description string
	ImageId     int
	OwningUsers []int
	Visibility  VisibilityType
}

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

func parsePersonType(s string) (PersonType, error) {
	switch s {
	case "parent":
		return Parent, nil
	case "child":
		return Child, nil
	default:
		return 0, fmt.Errorf("unknown type: %s", s)
	}
}

func parseGenderType(s string) (GenderType, error) {
	switch s {
	case "male":
		return Male, nil
	case "female":
		return Female, nil
	default:
		return 0, fmt.Errorf("unknown type: %s", s)
	}
}

func PackFamily(self *Family, buf *vpack.Buffer) {
	vpack.Version(1, buf)
	vpack.Int(&self.Id, buf)
	vpack.String(&self.Name, buf)
	vpack.String(&self.Description, buf)
	vpack.Slice(&self.OwningUsers, vpack.Int, buf)
	vpack.Int(&self.ImageId, buf)
	vpack.IntEnum(&self.Visibility, buf)
}

var FamilyBucket = vbolt.Bucket(&Info, "family", vpack.FInt, PackFamily)

func getFamily(tx *vbolt.Tx, id int) (family Family) {
	vbolt.Read(tx, FamilyBucket, id, &family)
	return
}

func GetAllFamilies(tx *vbolt.Tx) (families []Family) {
	vbolt.IterateAll(tx, FamilyBucket, func(key int, value Family) bool {
		generic.Append(&families, value)
		return true
	})
	return
}

func GetAllPublicFamilies(tx *vbolt.Tx) (families []Family) {
	vbolt.IterateAll(tx, FamilyBucket, func(key int, value Family) bool {
		if value.Visibility == Public {
			generic.Append(&families, value)
		}
		return true
	})
	return
}

func GetFamiliesForUser(tx *vbolt.Tx, userId int) (families []Family) {
	var familyIds []int
	vbolt.ReadTermTargets(tx, FamilyIndex, userId, &familyIds, vbolt.Window{})
	vbolt.ReadSlice(tx, FamilyBucket, familyIds, &families)
	return
}

// FamilyIndex term: family id, target: owning user ids
var FamilyIndex = vbolt.Index(&Info, "family_by", vpack.FInt, vpack.FInt)

func updateFamilyIndex(tx *vbolt.Tx, entry Family) {
	vbolt.SetTargetTermsPlain(
		tx,
		FamilyIndex,
		entry.Id,
		entry.OwningUsers,
	)
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

var PersonBucket = vbolt.Bucket(&Info, "people", vpack.FInt, PackPerson)

func GetAllPeople(tx *vbolt.Tx) (people []Person) {
	vbolt.IterateAll(tx, PersonBucket, func(key int, value Person) bool {
		prepPerson(&value)
		generic.Append(&people, value)
		return true
	})
	return people
}

func getPeopleInFamily(tx *vbolt.Tx, familyId int) (people []Person) {
	var personIds []int
	vbolt.ReadTermTargets(tx, PersonIndex, familyId, &personIds, vbolt.Window{})
	vbolt.ReadSlice(tx, PersonBucket, personIds, &people)
	for i := range people {
		prepPerson(&people[i])
	}
	return
}

// PersonIndex term: Person id, target: family id
var PersonIndex = vbolt.Index(&Info, "person_by", vpack.FInt, vpack.FInt)

func updatePersonIndex(tx *vbolt.Tx, entry Person) {
	vbolt.SetTargetTermsPlain(
		tx,
		PersonIndex,
		entry.Id,
		[]int{entry.FamilyId},
	)
}

func getAllPeopleMap(tx *vbolt.Tx) (peopleMap map[int]Person) {
	var people []Person
	vbolt.IterateAll(tx, PersonBucket, func(key int, value Person) bool {
		generic.Append(&people, value)
		return true
	})
	peopleMap = make(map[int]Person)
	for _, person := range people {
		peopleMap[person.Id] = person
	}
	return peopleMap
}

func getPerson(tx *vbolt.Tx, id int) (person Person) {
	vbolt.Read(tx, PersonBucket, id, &person)
	prepPerson(&person)
	return person
}

func prepPerson(person *Person) {
	person.Age = CalculateAge(person.Birthday, true)
}

func CalculateAge(birthday time.Time, includeMonths bool) string {
	now := time.Now()

	years := now.Year() - birthday.Year()
	months := int(now.Month()) - int(birthday.Month())

	if months < 0 || (months == 0 && now.Day() < birthday.Day()) {
		years--
		months += 12
	}

	if now.Day() < birthday.Day() {
		months--
		if months < 0 {
			months += 12
			years--
		}
	}

	if includeMonths {
		return fmt.Sprintf("%d years and %d months", years, months)
	}
	return fmt.Sprintf("%d years", years)
}

func RegisterChildrenPage(mux *http.ServeMux) {
	mux.Handle("GET /children/add", AuthHandler(ContextFunc(addPersonPage)))
	mux.Handle("GET /children/add/{id}", OwnerHandler(ContextFunc(editPersonPage)))
	mux.Handle("GET /children/delete/{id}", AuthHandler(ContextFunc(deletePerson)))
	mux.Handle("POST /children/add", AuthHandler(ContextFunc(savePerson)))

	mux.Handle("GET /family/create", AuthHandler(ContextFunc(createFamilyPage)))
	mux.Handle("GET /family/edit/{id}", OwnerHandler(ContextFunc(editFamilyPage)))
	mux.Handle("POST /family/create", AuthHandler(ContextFunc(saveFamily)))
	mux.Handle("POST /family/owner/{id}", OwnerHandler(ContextFunc(addOwnerPost)))

	mux.Handle("GET /person/{id}", PublicHandler(ContextFunc(personPage)))
}

func addPersonPage(context ResponseContext) {
	RenderTemplate(context, "children-add")
}
func editPersonPage(context ResponseContext) {
	vbolt.WithReadTx(db, func(tx *bolt.Tx) {
		id := context.r.PathValue("id")
		idVal, _ := strconv.Atoi(id)
		person := getPerson(tx, idVal)
		context.familyId = person.FamilyId
		RenderTemplateWithData(context, "children-add", map[string]any{
			"Person": person,
		})
	})
}
func deletePerson(context ResponseContext) {
	vbolt.WithWriteTx(db, func(tx *bolt.Tx) {
		id := context.r.PathValue("id")
		idVal, _ := strconv.Atoi(id)
		vbolt.Delete(tx, PersonBucket, idVal)
		vbolt.TxCommit(tx)
	})

	http.Redirect(context.w, context.r, "/", http.StatusFound)
}
func savePerson(context ResponseContext) {
	context.r.ParseForm()
	birthdate := context.r.FormValue("birthdate")
	name := context.r.FormValue("name")
	id, _ := strconv.Atoi(context.r.FormValue("id"))
	personType, _ := parsePersonType(context.r.FormValue("personType"))
	gender, _ := parseGenderType(context.r.FormValue("gender"))

	birthDateTime, _ := time.Parse("2006-01-02", birthdate)

	entry := Person{
		Birthday: birthDateTime,
		Name:     name,
		Id:       id,
		FamilyId: context.user.PrimaryFamilyId,
		Gender:   gender,
		Type:     personType,
	}
	vbolt.WithWriteTx(db, func(tx *bolt.Tx) {
		if entry.Id == 0 {
			entry.Id = vbolt.NextIntId(tx, PersonBucket)
		}
		vbolt.Write(tx, PersonBucket, entry.Id, &entry)
		updatePersonIndex(tx, entry)
		vbolt.TxCommit(tx)
	})

	http.Redirect(context.w, context.r, "/", http.StatusFound)
}

func createFamilyPage(context ResponseContext) {
	RenderTemplate(context, "family-create")
}
func editFamilyPage(context ResponseContext) {
	vbolt.WithReadTx(db, func(tx *bolt.Tx) {
		id := context.r.PathValue("id")
		idVal, _ := strconv.Atoi(id)
		context.familyId = idVal
		RenderTemplateWithData(context, "family-create", map[string]any{
			"Family": getFamily(tx, idVal),
		})
	})
}
func saveFamily(context ResponseContext) {
	context.r.ParseForm()
	name := context.r.FormValue("name")
	description := context.r.FormValue("description")
	id, err := strconv.Atoi(context.r.FormValue("id"))
	if err != nil {
		http.Error(context.w, err.Error(), http.StatusBadRequest)
		return
	}
	visibility, err := parseVisibilityType(context.r.FormValue("visibility"))
	if err != nil {
		http.Error(context.w, err.Error(), http.StatusBadRequest)
		return
	}

	entry := Family{
		Name:        name,
		Id:          id,
		Description: description,
		OwningUsers: []int{context.user.Id},
		Visibility:  visibility,
	}

	var user User
	vbolt.WithReadTx(db, func(tx *bolt.Tx) {
		user = GetUser(tx, context.user.Id)
	})

	vbolt.WithWriteTx(db, func(tx *bolt.Tx) {
		if entry.Id == 0 {
			entry.Id = vbolt.NextIntId(tx, FamilyBucket)
		}
		vbolt.Write(tx, FamilyBucket, entry.Id, &entry)
		updateFamilyIndex(tx, entry)
		if user.PrimaryFamilyId == 0 {
			user.PrimaryFamilyId = entry.Id
			vbolt.Write(tx, UsersBucket, context.user.Id, &user)
		}
		vbolt.TxCommit(tx)
	})

	http.Redirect(context.w, context.r, "/", http.StatusFound)
}

func personPage(context ResponseContext) {
	vbolt.WithReadTx(db, func(tx *bolt.Tx) {
		id := context.r.PathValue("id")
		idVal, _ := strconv.Atoi(id)
		person := getPerson(tx, idVal)
		prepPerson(&person)
		var image Image
		if person.ImageId > 0 {
			vbolt.Read(tx, ImageBucket, person.ImageId, &image)
		}
		context.familyId = person.FamilyId
		RenderTemplateWithData(context, "person", map[string]any{
			"Person": person,
			"Image":  image,
		})
	})
}

func addOwnerPost(context ResponseContext) {
	var userId int
	var family Family
	vbolt.WithReadTx(db, func(tx *bolt.Tx) {
		id := context.r.PathValue("id")
		idVal, _ := strconv.Atoi(id)
		context.familyId = idVal
		family = getFamily(tx, idVal)

		email := context.r.FormValue("owner")
		userId = GetUserId(tx, email)
	})
	if userId != 0 && family.Id != 0 {
		family.OwningUsers = append(family.OwningUsers, userId)
		vbolt.WithWriteTx(db, func(tx *bolt.Tx) {
			vbolt.Write(tx, FamilyBucket, family.Id, &family)
			tx.Commit()
		})
	}
	http.Redirect(context.w, context.r, "/", http.StatusFound)
}
