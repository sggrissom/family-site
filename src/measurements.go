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

type PersonWeight struct {
	Id         int
	PersonId   int
	Pounds     float64
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
	Values       []float64
}

func PackPersonHeight(self *PersonHeight, buf *vpack.Buffer) {
	vpack.Version(1, buf)
	vpack.Int(&self.Id, buf)
	vpack.Int(&self.PersonId, buf)
	vpack.Float64(&self.Inches, buf)
	vpack.Time(&self.Date, buf)
}

func PackPersonWeight(self *PersonWeight, buf *vpack.Buffer) {
	vpack.Version(1, buf)
	vpack.Int(&self.Id, buf)
	vpack.Int(&self.PersonId, buf)
	vpack.Float64(&self.Pounds, buf)
	vpack.Time(&self.Date, buf)
}

var PersonHeightBucket = vbolt.Bucket(&Info, "personHeight", vpack.FInt, PackPersonHeight)
var PersonWeightsBucket = vbolt.Bucket(&Info, "personWeight", vpack.FInt, PackPersonWeight)

// PersonHeightIdx term: person id, priority: timestamp, target: person height id
var PersonHeightIdx = vbolt.IndexExt(&Info, "heights_by", vpack.FInt, vpack.UnixTimeKey, vpack.FInt)

// PersonWeightIdx term: person id, priority: timestamp, target: person weight id
var PersonWeightIdx = vbolt.IndexExt(&Info, "weights_by", vpack.FInt, vpack.UnixTimeKey, vpack.FInt)

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

// Linear interpolation
func interpolate(x1, y1, x2, y2, x float64) float64 {
	return y1 + (y2-y1)/(x2-x1)*(x-x1)
}

func getHeightMilestones(heights []PersonHeight, milestones []float64) (milestoneInches []float64) {
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

func getWeightMilestones(weights []PersonWeight, milestones []float64) (milestonePounds []float64) {
	milestonePounds = make([]float64, 0, len(milestones))
	milestoneIdx, weightIdx := 0, 0

	for milestoneIdx < len(milestones) {
		nextMilestone := milestones[milestoneIdx]

		if weightIdx >= len(weights) {
			milestonePounds = append(milestonePounds, 0)
			milestoneIdx++
			continue
		}

		if weights[weightIdx].Age == nextMilestone {
			milestonePounds = append(milestonePounds, weights[weightIdx].Pounds)
			milestoneIdx++
		} else if weights[weightIdx].Age > nextMilestone && weightIdx > 0 {
			x1, y1 := weights[weightIdx-1].Age, weights[weightIdx-1].Pounds
			x2, y2 := weights[weightIdx].Age, weights[weightIdx].Pounds
			interpolated := interpolate(x1, y1, x2, y2, nextMilestone)
			milestonePounds = append(milestonePounds, interpolated)
			milestoneIdx++
		} else {
			weightIdx++
		}
	}

	return milestonePounds
}

func RegisterMeasurementsPages(mux *http.ServeMux) {
	mux.HandleFunc("GET /height", mainHeightsPage)
	mux.HandleFunc("GET /weight", mainWeightsPage)
	mux.HandleFunc("GET /height/add", addHeightPage)
	mux.Handle("POST /height/add", AuthHandler(http.HandlerFunc(saveHeightPage)))
	mux.HandleFunc("GET /weight/{personId}", personWeightPage)
	mux.HandleFunc("GET /weight/add", addWeightPage)
	mux.Handle("POST /weight/add", AuthHandler(http.HandlerFunc(saveWeightPage)))
	mux.HandleFunc("GET /api/height/{id}", heightApi)
	mux.HandleFunc("GET /api/weight/{id}", weightApi)

	mux.HandleFunc("GET /height/table/{id}", heightTablePage)
	mux.HandleFunc("GET /weight/table/{id}", weightTablePage)
	mux.HandleFunc("GET /api/height/table", heightTableApi)
	mux.HandleFunc("GET /api/weight/table", weightTableApi)
}

func mainHeightsPage(w http.ResponseWriter, r *http.Request) {
	vbolt.WithReadTx(db, func(tx *bolt.Tx) {
		RenderTemplate(w, "height", map[string]interface{}{
			"People": getAllPeople(tx),
		})
	})
}
func mainWeightsPage(w http.ResponseWriter, r *http.Request) {
	vbolt.WithReadTx(db, func(tx *bolt.Tx) {
		RenderTemplate(w, "weight", map[string]interface{}{
			"People": getAllPeople(tx),
		})
	})
}

func addHeightPage(w http.ResponseWriter, r *http.Request) {
	vbolt.WithReadTx(db, func(tx *bolt.Tx) {
		RenderTemplate(w, "height-add", map[string]interface{}{
			"People": getAllPeople(tx),
		})
	})
}

func saveHeightPage(w http.ResponseWriter, r *http.Request) {
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
}

func personWeightPage(w http.ResponseWriter, r *http.Request) {
	personId, _ := strconv.Atoi(r.PathValue("personId"))
	RenderTemplate(w, "weight", map[string]interface{}{
		"Weight": QueryWeights(personId),
	})
}

func addWeightPage(w http.ResponseWriter, r *http.Request) {
	vbolt.WithReadTx(db, func(tx *bolt.Tx) {
		RenderTemplate(w, "weight-add", map[string]interface{}{
			"People": getAllPeople(tx),
		})
	})
}

func saveWeightPage(w http.ResponseWriter, r *http.Request) {
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
}
func heightApi(w http.ResponseWriter, r *http.Request) {
	personId, _ := strconv.Atoi(r.PathValue("id"))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(QueryHeights(personId))
}
func weightApi(w http.ResponseWriter, r *http.Request) {
	personId, _ := strconv.Atoi(r.PathValue("id"))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(QueryWeights(personId))
}
func heightTablePage(w http.ResponseWriter, r *http.Request) {
	personId, _ := strconv.Atoi(r.PathValue("id"))
	RenderTemplate(w, "height-table", map[string]interface{}{
		"Heights": QueryHeights(personId),
	})
}
func weightTablePage(w http.ResponseWriter, r *http.Request) {
	personId, _ := strconv.Atoi(r.PathValue("id"))
	RenderTemplate(w, "weight-table", map[string]interface{}{
		"Weights": QueryWeights(personId),
	})
}
func heightTableApi(w http.ResponseWriter, r *http.Request) {
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
			Values:       make([]float64, len(personIDs)),
		})
	}
	for i := range personIDs {
		personId, _ := strconv.Atoi(personIDs[i])
		personHeights := QueryHeights(personId)
		personMilestones := getHeightMilestones(personHeights, milestones)
		for j := range personMilestones {
			response.Milestones[j].Values[i] = personMilestones[j]
			response.Milestones[j].Average += personMilestones[j]
		}
	}

	for i := range milestones {
		dataPointCount := 0
		for j := range personIDs {
			if response.Milestones[i].Values[j] > 0 {
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
}
func weightTableApi(w http.ResponseWriter, r *http.Request) {
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
			Values:       make([]float64, len(personIDs)),
		})
	}
	for i := range personIDs {
		personId, _ := strconv.Atoi(personIDs[i])
		personWeights := QueryWeights(personId)
		personMilestones := getWeightMilestones(personWeights, milestones)
		for j := range personMilestones {
			response.Milestones[j].Values[i] = personMilestones[j]
			response.Milestones[j].Average += personMilestones[j]
		}
	}

	for i := range milestones {
		dataPointCount := 0
		for j := range personIDs {
			if response.Milestones[i].Values[j] > 0 {
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
}
