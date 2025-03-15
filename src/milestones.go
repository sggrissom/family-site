package main

import (
	"fmt"
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

func parseMilestoneTypeLabel(t MilestoneType) string {
	switch t {
	case MilestoneHeight:
		return "height"
	case MilestoneWeight:
		return "weight"
	case MilestoneCrawling:
		return "crawling"
	case MilestoneWalking:
		return "walking"
	case MilestoneFirstWord:
		return "first_word"
	default:
		return ""
	}
}

func parseMilestoneType(s string) (MilestoneType, error) {
	switch s {
	case "height":
		return MilestoneHeight, nil
	case "weight":
		return MilestoneWeight, nil
	case "crawling":
		return MilestoneCrawling, nil
	case "walking":
		return MilestoneWalking, nil
	case "first_word":
		return MilestoneFirstWord, nil
	default:
		return 0, fmt.Errorf("unknown milestone type: %s", s)
	}
}

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
	mux.Handle("GET /milestones/{id}", PublicHandler(ContextFunc(milestonesPage)))
	mux.Handle("GET /milestones/add", AuthHandler(ContextFunc(addMilestonesPage)))
	mux.Handle("GET /milestones/edit/{id}", AuthHandler(ContextFunc(editMilestonesPage)))
	mux.Handle("GET /milestones/delete/{id}", AuthHandler(ContextFunc(deleteMilestone)))
	mux.Handle("POST /milestones/add", AuthHandler(ContextFunc(saveMilestone)))
}

func milestonesPage(context ResponseContext) {
	id := context.r.PathValue("id")
	idVal, err := strconv.Atoi(id)
	if err != nil {
		idVal = 1
	}
	RenderTemplateWithData(context, "milestones", map[string]interface{}{
		"Milestones": QueryMilestones(idVal),
	})
}

func addMilestonesPage(context ResponseContext) {
	vbolt.WithReadTx(db, func(tx *bolt.Tx) {
		RenderTemplateWithData(context, "milestones-add", map[string]interface{}{
			"People": getAllPeople(tx),
		})
	})
}

func editMilestonesPage(context ResponseContext) {
	RenderTemplate(context, "milestones-add")
}

func deleteMilestone(context ResponseContext) {
}

func saveMilestone(context ResponseContext) {
	context.r.ParseForm()
	measureDate := context.r.FormValue("measureDate")
	numericValue, _ := strconv.ParseFloat(context.r.FormValue("numericValue"), 64)
	personId, _ := strconv.Atoi(context.r.FormValue("personId"))
	id, _ := strconv.Atoi(context.r.FormValue("id"))
	milestoneType, _ := parseMilestoneType(context.r.FormValue("milestoneType"))
	unit := context.r.FormValue("unit")
	textValue := context.r.FormValue("textValue")
	notes := context.r.FormValue("notes")

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
		Age:          0,
	}
	vbolt.WithWriteTx(db, func(tx *bolt.Tx) {
		if entry.Id == 0 {
			entry.Id = vbolt.NextIntId(tx, MilestoneBucket)
		}
		vbolt.Write(tx, MilestoneBucket, entry.Id, &entry)
		updateMilestoneIndex(tx, entry)
		vbolt.TxCommit(tx)
	})

	http.Redirect(context.w, context.r, "/milestones", http.StatusFound)
}
