package api

import (
	"encoding/json"
	"testing"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"github.com/zl14229/pokemon-shiny-tracker/internal/reference"
	"github.com/zl14229/pokemon-shiny-tracker/internal/service"
	"github.com/zl14229/pokemon-shiny-tracker/internal/testutil"
)

func TestOfflineDexAPIUnknownDenominatorAndPagination(t *testing.T) {
	db := testutil.DB(t)
	if e := reference.EnsureCatalog(db); e != nil {
		t.Fatal(e)
	}
	router := NewRouter(db)
	var species model.Species
	db.Where("national_dex = ?", 25).First(&species)
	if w := jsonRequest(router, "POST", "/api/v1/shiny-records", `{"current_species_id":`+stringID(species.ID)+`,"holding_status":"held"}`); w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	w := request(router, "GET", "/api/v1/dex?acquired=obtained&page_size=1")
	var v service.DexResult
	if e := json.Unmarshal(w.Body.Bytes(), &v); e != nil {
		t.Fatal(e)
	}
	if w.Code != 200 || v.Meta.Total != 1 || v.Data[0].NationalDex != 25 || v.Summary.Percentage != nil || v.Summary.EligibleTargets != 0 || v.Coverage.Species != 1025 || v.Data[0].CrossCheckStatus != "unknown" {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, path := range []string{"/api/v1/dex?basis=current&dimension=form", "/api/v1/dex?display=legal", "/api/v1/reference/availability"} {
		if w := request(router, "GET", path); w.Code != 200 {
			t.Fatal(path, w.Code)
		}
	}
	for _, path := range []string{"/api/v1/dex?basis=bad", "/api/v1/dex?scope=game&game_id=0", "/api/v1/reference/availability?game_id=0", "/api/v1/dex?dimension=form&dimension=species"} {
		if w := request(router, "GET", path); w.Code != 400 {
			t.Fatal(path, w.Code)
		}
	}
}
func stringID(id uint64) string { b, _ := json.Marshal(id); return string(b) }
