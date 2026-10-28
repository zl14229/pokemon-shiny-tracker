package api

import (
	"bytes"
	"encoding/json"
	"github.com/zl14229/pokemon-shiny-tracker/internal/huntdata"
	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"github.com/zl14229/pokemon-shiny-tracker/internal/reference"
	"github.com/zl14229/pokemon-shiny-tracker/internal/service"
	"github.com/zl14229/pokemon-shiny-tracker/internal/testutil"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestBingoAPIInsufficiencyCreateReviewManualProofAndHistory(t *testing.T) {
	db := testutil.DB(t)
	reference.EnsureSeed(db)
	reference.EnsureCatalog(db)
	if e := huntdata.EnsureBundled(db); e != nil {
		t.Fatal(e)
	}
	router := NewRouter(db)
	post := func(path string, v any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(v)
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", path, bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, r)
		return w
	}
	in := service.BingoCreateInput{RequestID: "api-bingo-001", Rules: service.PlannerRules{GameIDs: []uint64{3, 4}, Count: 9}, Size: 5, CalendarZone: "UTC"}
	if w := post("/api/v1/planner/bingo", in); w.Code != 409 {
		t.Fatal(w.Code, w.Body.String())
	}
	in.Size = 3
	w := post("/api/v1/planner/bingo", in)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var out struct {
		Data service.PlannerView `json:"data"`
	}
	if e := json.Unmarshal(w.Body.Bytes(), &out); e != nil {
		t.Fatal(e)
	}
	v := out.Data
	target := v.Targets[0]
	form := target.Snapshot.Choice.Evidence[0].FormID
	day := v.Bingo.StartedAt.Format("2006-01-02")
	record := model.ShinyRecord{OriginalSpeciesID: &target.SpeciesID, OriginalFormID: &form, CurrentSpeciesID: target.SpeciesID, CurrentFormID: &form, OriginalGameID: &target.GameID, CurrentGameID: &target.GameID, ObtainedDate: &day, Source: "self_hunted", HoldingStatus: "held"}
	if e := db.Create(&record).Error; e != nil {
		t.Fatal(e)
	}
	base := "/api/v1/planner/plans/" + strconv.FormatUint(v.ID, 10)
	w = post(base+"/refresh", struct{}{})
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	json.Unmarshal(w.Body.Bytes(), &out)
	if out.Data.Targets[0].Status == "completed" {
		t.Fatal("same-day automatically completed")
	}
	timestamp := v.Bingo.StartedAt.Add(time.Nanosecond)
	w = post(base+"/targets/"+strconv.FormatUint(target.ID, 10)+"/verify-date", service.BingoDateInput{RecordID: record.ID, ObtainedAt: timestamp, EvidenceNote: "synthetic API timestamp review", Confirmed: true})
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	json.Unmarshal(w.Body.Bytes(), &out)
	if out.Data.Targets[0].Status != "completed" {
		t.Fatal("explicit timestamp not credited")
	}
	if w = post(base+"/close", map[string]string{"status": "abandoned"}); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	json.Unmarshal(w.Body.Bytes(), &out)
	if out.Data.Status != "abandoned" || out.Data.Targets[0].Status != "completed" {
		t.Fatal("abandon lost earned history")
	}
	if w = post(base+"/targets/"+strconv.FormatUint(target.ID, 10)+"/reroll", service.BingoRerollInput{RequestID: "api-reroll-001", ExpectedTargetKey: target.TargetKey}); w.Code != 409 {
		t.Fatal("closed/completed target rerolled", w.Code)
	}
}
