package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	"go.hasen.dev/vbolt"
	"go.hasen.dev/vpack"
)

type PersonWeight struct {
	Id         int
	PersonId   int
	Pounds     float64
	Date       time.Time
	DateString string
	Age        float64
	PersonName string
}

func PackPersonWeight(self *PersonWeight, buf *vpack.Buffer) {
	vpack.Version(1, buf)
	vpack.Int(&self.Id, buf)
	vpack.Int(&self.PersonId, buf)
	vpack.Float64(&self.Pounds, buf)
	vpack.Time(&self.Date, buf)
}

var PersonWeightsBucket = vbolt.Bucket(&Info, "personWeight", vpack.FInt, PackPersonWeight)

// PersonWeightIdx term: person id, priority: timestamp, target: person weight id
var PersonWeightIdx = vbolt.IndexExt(&Info, "weights_by", vpack.FInt, vpack.UnixTimeKey, vpack.FInt)

func updateWeightIndex(tx *vbolt.Tx, entry PersonWeight) {
	priority := entry.Date
	vbolt.SetTargetSingleTermExt(
		tx,              // transaction
		PersonWeightIdx, // index reference
		entry.Id,        // target
		priority,        // priority (same for all terms)
		entry.PersonId,  // terms (slice)
	)
}

func QueryWeights(personId int) (weights []PersonWeight) {
	vbolt.WithReadTx(db, func(tx *bolt.Tx) {
		people := getAllPeopleMap(tx)

		var window vbolt.Window
		var weightIds []int
		vbolt.ReadTermTargets(tx, PersonWeightIdx, personId, &weightIds, window)
		vbolt.ReadSlice(tx, PersonWeightsBucket, weightIds, &weights)
		for i := range weights {
			weights[i].DateString = weights[i].Date.Format("January 02, 2006")
			weights[i].Age = weights[i].Date.Sub(people[weights[i].PersonId].BirthdayRaw).Hours() / (365.25 * 24) // Age in years
			weights[i].PersonName = people[weights[i].PersonId].Name
		}
	})
	return
}

func RegisterWeightPage(mux *http.ServeMux) {
	mux.HandleFunc("GET /weight", func(w http.ResponseWriter, r *http.Request) {
		vbolt.WithReadTx(db, func(tx *bolt.Tx) {
			RenderTemplate(w, "weight", struct {
				People []Person
			}{
				People: getAllPeople(tx),
			})
		})
	})
	mux.HandleFunc("GET /weight/{personId}", func(w http.ResponseWriter, r *http.Request) {
		personId, _ := strconv.Atoi(r.PathValue("personId"))
		RenderTemplate(w, "weight", QueryWeights(personId))
	})
	mux.HandleFunc("GET /weight/add", func(w http.ResponseWriter, r *http.Request) {
		vbolt.WithReadTx(db, func(tx *bolt.Tx) {
			RenderTemplate(w, "weight-add", struct {
				Weight PersonWeight
				People []Person
			}{
				People: getAllPeople(tx),
			})
		})
	})
	mux.HandleFunc("POST /weight/add", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		measureDate := r.FormValue("measureDate")
		pounds, _ := strconv.ParseFloat(r.FormValue("pounds"), 64)
		personId, _ := strconv.Atoi(r.FormValue("personId"))
		id, _ := strconv.Atoi(r.FormValue("id"))

		measureDateTime, _ := time.Parse("2006-01-02", measureDate)

		entry := PersonWeight{
			Id:       id,
			Date:     measureDateTime,
			PersonId: personId,
			Pounds:   pounds,
		}
		vbolt.WithWriteTx(db, func(tx *bolt.Tx) {
			if entry.Id == 0 {
				entry.Id = vbolt.NextIntId(tx, PersonWeightsBucket)
			}
			vbolt.Write(tx, PersonWeightsBucket, entry.Id, &entry)
			updateWeightIndex(tx, entry)
			vbolt.TxCommit(tx)
		})

		http.Redirect(w, r, "/weight", http.StatusFound)
	})
	mux.HandleFunc("GET /api/weight/{id}", func(w http.ResponseWriter, r *http.Request) {
		personId, _ := strconv.Atoi(r.PathValue("id"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(QueryWeights(personId))
	})
}
