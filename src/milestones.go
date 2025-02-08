package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	"go.hasen.dev/vbolt"
	"go.hasen.dev/vpack"
)

type MilestoneType int

const (
	MilestoneHeight MilestoneType = iota
	MilestoneWeight
	MilestoneCrawling
	MilestoneWalking
	MilestoneFirstWord
)

type Milestone struct {
	Id           int
	PersonId     int
	Type         MilestoneType
	Date         time.Time
	Age          float64
	NumericValue float64
	Unit         string
	TextValue    string
	Notes        string
}

func PackMilestone(self *Milestone, buf *vpack.Buffer) {
	vpack.Version(1, buf)
	vpack.Int(&self.Id, buf)
	vpack.Int(&self.PersonId, buf)
	vpack.IntEnum(&self.Type, buf)
	vpack.Time(&self.Date, buf)
	vpack.Float64(&self.Age, buf)
	vpack.Float64(&self.NumericValue, buf)
	vpack.String(&self.Unit, buf)
	vpack.String(&self.TextValue, buf)
	vpack.String(&self.Notes, buf)
}

var MilestoneBucket = vbolt.Bucket(&Info, "personMilestones", vpack.FInt, PackMilestone)

// MilestoneIndex term: person id, priority: timestamp, target: milestone id
var MilestoneIndex = vbolt.IndexExt(&Info, "milestones_by", vpack.FInt, vpack.UnixTimeKey, vpack.FInt)

func updateMilestoneIndex(tx *vbolt.Tx, entry Milestone) {
	vbolt.SetTargetSingleTermExt(
		tx,
		MilestoneIndex,
		entry.Id,
		entry.Date,
		entry.PersonId,
	)
}

func QueryMilestones(personId int) (milestones []Milestone) {
	vbolt.WithReadTx(db, func(tx *bolt.Tx) {
		var window vbolt.Window
		var milestoneIds []int
		vbolt.ReadTermTargets(tx, MilestoneIndex, personId, &milestoneIds, window)
		vbolt.ReadSlice(tx, MilestoneBucket, milestoneIds, &milestones)
	})
	return
}

func RegisterMilestonesPages(mux *http.ServeMux) {
	mux.Handle("GET /milestones", PublicHandler(http.HandlerFunc(milestonesPage)))
	mux.Handle("GET /milestones/add", AuthHandler(http.HandlerFunc(addMilestonesPage)))
	mux.Handle("GET /milestones/edit/{id}", AuthHandler(http.HandlerFunc(editMilestonesPage)))
	mux.Handle("GET /milestones/delete/{id}", AuthHandler(http.HandlerFunc(deleteMilestone)))
	mux.Handle("POST /milestones/add", AuthHandler(http.HandlerFunc(saveMilestone)))
}

func milestonesPage(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, "milestones")
}

func addMilestonesPage(w http.ResponseWriter, r *http.Request) {
	vbolt.WithReadTx(db, func(tx *bolt.Tx) {
		RenderTemplateWithData(w, "milestones-add", map[string]interface{}{
			"People": getAllPeople(tx),
		})
	})
}

func editMilestonesPage(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, "milestones-add")
}

func deleteMilestone(w http.ResponseWriter, r *http.Request) {
}

func saveMilestone(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	measureDate := r.FormValue("measureDate")
	numericValue, _ := strconv.ParseFloat(r.FormValue("numericValue"), 64)
	personId, _ := strconv.Atoi(r.FormValue("personId"))
	id, _ := strconv.Atoi(r.FormValue("id"))
	//milestoneType := r.FormValue("milestoneType")
	milestoneType := MilestoneWeight
	unit := r.FormValue("unit")
	textValue := r.FormValue("textValue")
	notes := r.FormValue("notes")
	age, _ := strconv.ParseFloat(r.FormValue("age"), 64)

	measureDateTime, _ := time.Parse("2006-01-02", measureDate)

	entry := Milestone{
		Id:           id,
		Date:         measureDateTime,
		PersonId:     personId,
		Type:         milestoneType,
		NumericValue: numericValue,
		Unit:         unit,
		TextValue:    textValue,
		Notes:        notes,
		Age:          age,
	}
	vbolt.WithWriteTx(db, func(tx *bolt.Tx) {
		if entry.Id == 0 {
			entry.Id = vbolt.NextIntId(tx, MilestoneBucket)
		}
		vbolt.Write(tx, MilestoneBucket, entry.Id, &entry)
		updateMilestoneIndex(tx, entry)
		vbolt.TxCommit(tx)
	})

	http.Redirect(w, r, "/milestones", http.StatusFound)
}
