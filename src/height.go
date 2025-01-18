package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	"go.hasen.dev/vbolt"
	"go.hasen.dev/vpack"
)

type PersonHeight struct {
	Id         int
	PersonId   int
	Inches     float64
	Date       time.Time
	DateString string
	Age        float64
	PersonName string
}

type MilestoneResponse struct {
	People     []Person
	Milestones []MilestoneAges
}

type MilestoneAges struct {
	MilestoneAge float64
	Average      float64
	Inches       []float64
}

func PackPersonHeight(self *PersonHeight, buf *vpack.Buffer) {
	vpack.Version(1, buf)
	vpack.Int(&self.Id, buf)
	vpack.Int(&self.PersonId, buf)
	vpack.Float64(&self.Inches, buf)
	vpack.Time(&self.Date, buf)
}

var PersonHeightBucket = vbolt.Bucket(&Info, "personHeight", vpack.FInt, PackPersonHeight)

// PersonHeightIdx term: person id, priority: timestamp, target: person height id
var PersonHeightIdx = vbolt.IndexExt(&Info, "heights_by", vpack.FInt, vpack.UnixTimeKey, vpack.FInt)

func updateIndex(tx *vbolt.Tx, entry PersonHeight) {
	priority := entry.Date
	vbolt.SetTargetSingleTermExt(
		tx,              // transaction
		PersonHeightIdx, // index reference
		entry.Id,        // target
		priority,        // priority (same for all terms)
		entry.PersonId,  // terms (slice)
	)
}

func QueryHeights(personId int) (heights []PersonHeight) {
	vbolt.WithReadTx(db, func(tx *bolt.Tx) {
		people := getAllPeopleMap(tx)

		var window vbolt.Window
		var heightIds []int
		vbolt.ReadTermTargets(tx, PersonHeightIdx, personId, &heightIds, window)
		vbolt.ReadSlice(tx, PersonHeightBucket, heightIds, &heights)
		for i := range heights {
			heights[i].DateString = heights[i].Date.Format("January 02, 2006")
			heights[i].Age = heights[i].Date.Sub(people[heights[i].PersonId].BirthdayRaw).Hours() / (365.25 * 24) // Age in years
			heights[i].PersonName = people[heights[i].PersonId].Name
		}
	})
	return
}

// Linear interpolation
func interpolate(x1, y1, x2, y2, x float64) float64 {
	return y1 + (y2-y1)/(x2-x1)*(x-x1)
}

func getMilestones(heights []PersonHeight, milestones []float64) (milestoneInches []float64) {
	milestoneInches = make([]float64, 0, len(milestones))
	milestoneIdx, heightIdx := 0, 0

	for milestoneIdx < len(milestones) {
		nextMilestone := milestones[milestoneIdx]

		if heightIdx >= len(heights) {
			milestoneInches = append(milestoneInches, 0)
			milestoneIdx++
			continue
		}

		if heights[heightIdx].Age == nextMilestone {
			milestoneInches = append(milestoneInches, heights[heightIdx].Inches)
			milestoneIdx++
		} else if heights[heightIdx].Age > nextMilestone && heightIdx > 0 {
			x1, y1 := heights[heightIdx-1].Age, heights[heightIdx-1].Inches
			x2, y2 := heights[heightIdx].Age, heights[heightIdx].Inches
			interpolated := interpolate(x1, y1, x2, y2, nextMilestone)
			milestoneInches = append(milestoneInches, interpolated)
			milestoneIdx++
		} else {
			heightIdx++
		}
	}

	return milestoneInches
}

func RegisterHeightPage(mux *http.ServeMux) {
	mux.HandleFunc("GET /height", func(w http.ResponseWriter, r *http.Request) {
		vbolt.WithReadTx(db, func(tx *bolt.Tx) {
			RenderTemplate(w, "height", struct {
				People []Person
			}{
				People: getAllPeople(tx),
			})
		})
	})
	mux.HandleFunc("GET /height/add", func(w http.ResponseWriter, r *http.Request) {
		vbolt.WithReadTx(db, func(tx *bolt.Tx) {
			RenderTemplate(w, "height-add", struct {
				Height PersonHeight
				People []Person
			}{
				People: getAllPeople(tx),
			})
		})
	})
	mux.HandleFunc("POST /height/add", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		measureDate := r.FormValue("measureDate")
		inches, _ := strconv.ParseFloat(r.FormValue("inches"), 64)
		personId, _ := strconv.Atoi(r.FormValue("personId"))
		id, _ := strconv.Atoi(r.FormValue("id"))

		measureDateTime, _ := time.Parse("2006-01-02", measureDate)

		entry := PersonHeight{
			Id:       id,
			Date:     measureDateTime,
			PersonId: personId,
			Inches:   inches,
		}
		vbolt.WithWriteTx(db, func(tx *bolt.Tx) {
			if entry.Id == 0 {
				entry.Id = vbolt.NextIntId(tx, PersonHeightBucket)
			}
			vbolt.Write(tx, PersonHeightBucket, entry.Id, &entry)
			updateIndex(tx, entry)
			vbolt.TxCommit(tx)
		})

		http.Redirect(w, r, "/height", http.StatusFound)
	})
	mux.HandleFunc("GET /api/height/{id}", func(w http.ResponseWriter, r *http.Request) {
		personId, _ := strconv.Atoi(r.PathValue("id"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(QueryHeights(personId))
	})

	mux.HandleFunc("GET /height/table/{id}", func(w http.ResponseWriter, r *http.Request) {
		personId, _ := strconv.Atoi(r.PathValue("id"))
		RenderTemplate(w, "height-table", QueryHeights(personId))
	})
	mux.HandleFunc("GET /api/height/table", func(w http.ResponseWriter, r *http.Request) {
		milestones := []float64{0, 1.0 / 12, 2.0 / 12, 3.0 / 12,
			4.0 / 12, 5.0 / 12, 6.0 / 12, 7.0 / 12, 8.0 / 12,
			9.0 / 12, 10.0 / 12, 11.0 / 12, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13}
		personIDs := r.URL.Query()["ids"]
		var response MilestoneResponse

		vbolt.WithReadTx(db, func(tx *bolt.Tx) {
			people := getAllPeopleMap(tx)
			for _, personId := range personIDs {
				id, _ := strconv.Atoi(personId)
				response.People = append(response.People, people[id])
			}
		})

		response.Milestones = make([]MilestoneAges, 0, len(milestones))
		for i := range milestones {
			response.Milestones = append(response.Milestones, MilestoneAges{
				MilestoneAge: milestones[i],
				Inches:       make([]float64, len(personIDs)),
			})
		}
		for i := range personIDs {
			personId, _ := strconv.Atoi(personIDs[i])
			personHeights := QueryHeights(personId)
			personMilestones := getMilestones(personHeights, milestones)
			for j := range personMilestones {
				response.Milestones[j].Inches[i] = personMilestones[j]
				response.Milestones[j].Average += personMilestones[j]
			}
		}

		for i := range milestones {
			dataPointCount := 0
			for j := range personIDs {
				if response.Milestones[i].Inches[j] > 0 {
					dataPointCount++
				}
			}
			if dataPointCount > 0 {
				response.Milestones[i].Average = response.Milestones[i].Average / float64(dataPointCount)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Printf("Error encoding response: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})
}
