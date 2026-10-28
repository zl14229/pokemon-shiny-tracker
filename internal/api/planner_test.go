package api

import (
	"bytes"
	"encoding/json"
	"github.com/zl14229/pokemon-shiny-tracker/internal/huntdata"
	"github.com/zl14229/pokemon-shiny-tracker/internal/reference"
	"github.com/zl14229/pokemon-shiny-tracker/internal/service"
	"github.com/zl14229/pokemon-shiny-tracker/internal/testutil"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestPlannerAPIOfflineDraftSaveHuntAndOriginGuard(t *testing.T) {
	db := testutil.DB(t)
	reference.EnsureSeed(db)
	reference.EnsureCatalog(db)
	if e := huntdata.EnsureBundled(db); e != nil {
		t.Fatal(e)
	}
	r := NewRouter(db)
	post := func(path string, data string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", path, bytes.NewBufferString(data))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w
	}
	input := `{"request_id":"api-draw-001","rules":{"game_ids":[3,4],"count":2}}`
	w := post("/api/v1/planner/random", input)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var out struct {
		Data service.PlannerView `json:"data"`
	}
	if e := json.Unmarshal(w.Body.Bytes(), &out); e != nil || len(out.Data.Targets) != 2 {
		t.Fatal(e, w.Body.String())
	}
	base := "/api/v1/planner/plans/" + strconv.FormatUint(out.Data.ID, 10)
	if w = post(base+"/save", `{}`); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = post(base+"/targets/"+strconv.FormatUint(out.Data.Targets[0].ID, 10)+"/hunt", `{}`); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	bad := httptest.NewRequest("POST", "/api/v1/planner/random", bytes.NewBufferString(input))
	bad.Header.Set("Content-Type", "application/json")
	bad.Header.Set("Origin", "https://unrelated.example")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, bad)
	if w.Code != 403 {
		t.Fatal("cross-site write accepted")
	}
	for _, path := range []string{"/api/v1/planner/plans?limit=101", base + "/targets/0/hunt"} {
		if path[len(path)-4:] == "hunt" {
			w = post(path, `{}`)
		} else {
			w = httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		}
		if w.Code != 400 {
			t.Fatal(path, w.Code)
		}
	}
}
