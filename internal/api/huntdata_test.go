package api

import (
	"encoding/json"
	"github.com/zl14229/pokemon-shiny-tracker/internal/huntdata"
	"github.com/zl14229/pokemon-shiny-tracker/internal/reference"
	"github.com/zl14229/pokemon-shiny-tracker/internal/testutil"
	"net/http/httptest"
	"testing"
)

func TestOfflineReviewedHuntDataAPI(t *testing.T) {
	db := testutil.DB(t)
	if e := reference.EnsureSeed(db); e != nil {
		t.Fatal(e)
	}
	if e := reference.EnsureCatalog(db); e != nil {
		t.Fatal(e)
	}
	if e := huntdata.EnsureBundled(db); e != nil {
		t.Fatal(e)
	}
	r := NewRouter(db)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/reference/hunt-data?limit=5", nil))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var out struct {
		Data []huntdata.Row `json:"data"`
		Meta struct {
			Total    int    `json:"total"`
			Coverage string `json:"coverage"`
		} `json:"meta"`
	}
	if e := json.Unmarshal(w.Body.Bytes(), &out); e != nil {
		t.Fatal(e)
	}
	if len(out.Data) != 5 || out.Meta.Total != 24 || out.Meta.Coverage != "partial" || !out.Data[0].Fact.Huntable() {
		t.Fatal(out)
	}
	for _, path := range []string{"/api/v1/reference/hunt-data?game_id=0", "/api/v1/reference/hunt-data?limit=0"} {
		w = httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 400 {
			t.Fatal(path, w.Code)
		}
	}
}
