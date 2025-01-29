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

type Person struct {
	Id          int
	Name        string
	BirthdayRaw time.Time
	Birthday    string
	Age         string
	Height      string
	Weight      string
}

func PackPerson(self *Person, buf *vpack.Buffer) {
	vpack.Version(1, buf)
	vpack.Int(&self.Id, buf)
	vpack.String(&self.Name, buf)
	vpack.Time(&self.BirthdayRaw, buf)
}

var PersonBucket = vbolt.Bucket(&Info, "people", vpack.FInt, PackPerson)

func getAllPeople(tx *vbolt.Tx) (people []Person) {
	vbolt.IterateAll(tx, PersonBucket, func(key int, value Person) bool {
		prepPerson(&value, "January 02, 2006")
		generic.Append(&people, value)
		return true
	})
	return people
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
	prepPerson(&person, "2006-01-02")
	return person
}

func prepPerson(person *Person, format string) {
	heights := QueryHeights(person.Id)
	if len(heights) > 0 {
		person.Height = strconv.FormatFloat(heights[len(heights)-1].Inches, 'f', -1, 64)
	} else {
		person.Height = "none"
	}
	weights := QueryWeights(person.Id)
	if len(weights) > 0 {
		person.Weight = strconv.FormatFloat(weights[len(weights)-1].Pounds, 'f', -1, 64)
	} else {
		person.Weight = "none"
	}
	person.Birthday = person.BirthdayRaw.Format(format)
	person.Age = CalculateAge(person.BirthdayRaw, true)
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
	mux.HandleFunc("/children", peoplePage)
	mux.HandleFunc("/children/admin", personAdminPage)
	mux.HandleFunc("GET /children/add", addPersonPage)
	mux.HandleFunc("GET /children/add/{id}", editPersonPage)
	mux.Handle("GET /children/delete/{id}", AuthHandler(http.HandlerFunc(deletePerson)))
	mux.Handle("POST /children/add", AuthHandler(http.HandlerFunc(savePerson)))
}

func peoplePage(w http.ResponseWriter, r *http.Request) {
	vbolt.WithReadTx(db, func(tx *bolt.Tx) {
		RenderTemplateWithData(w, "children", map[string]interface{}{
			"People": getAllPeople(tx),
		})
	})
}
func personAdminPage(w http.ResponseWriter, r *http.Request) {
	vbolt.WithReadTx(db, func(tx *bolt.Tx) {
		RenderTemplateBlock(w, "children", "childrenAdmin", getAllPeople(tx))
	})
}
func addPersonPage(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, "children-add")
}
func editPersonPage(w http.ResponseWriter, r *http.Request) {
	vbolt.WithReadTx(db, func(tx *bolt.Tx) {
		id := r.PathValue("id")
		idVal, _ := strconv.Atoi(id)
		RenderTemplateWithData(w, "children-add", map[string]interface{}{
			"Person": getPerson(tx, idVal),
		})
	})
}
func deletePerson(w http.ResponseWriter, r *http.Request) {
	vbolt.WithWriteTx(db, func(tx *bolt.Tx) {
		id := r.PathValue("id")
		idVal, _ := strconv.Atoi(id)
		vbolt.Delete(tx, PersonBucket, idVal)
		vbolt.TxCommit(tx)
	})

	http.Redirect(w, r, "/children", http.StatusFound)
}
func savePerson(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	birthdate := r.FormValue("birthdate")
	name := r.FormValue("name")
	id, _ := strconv.Atoi(r.FormValue("id"))

	birthDateTime, _ := time.Parse("2006-01-02", birthdate)

	entry := Person{
		BirthdayRaw: birthDateTime,
		Name:        name,
		Id:          id,
	}
	vbolt.WithWriteTx(db, func(tx *bolt.Tx) {
		if entry.Id == 0 {
			entry.Id = vbolt.NextIntId(tx, PersonBucket)
		}
		vbolt.Write(tx, PersonBucket, entry.Id, &entry)
		vbolt.TxCommit(tx)
	})

	http.Redirect(w, r, "/children", http.StatusFound)
}
