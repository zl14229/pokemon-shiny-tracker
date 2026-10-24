package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/zl14229/pokemon-shiny-tracker/internal/reference"
	"github.com/zl14229/pokemon-shiny-tracker/internal/testutil"
)

func TestOfflineReferenceQueriesAndEvidence(t *testing.T) {
	db := testutil.DB(t)
	if err := reference.EnsureSeed(db); err != nil {
		t.Fatal(err)
	}
	router := NewRouter(db)
	for _, path := range []string{"/api/v1/reference/species", "/api/v1/reference/forms?species_id=1", "/api/v1/reference/games", "/api/v1/reference/species/1", "/api/v1/reference/species/1/sources", "/api/v1/reference/coverage"} {
		response := request(router, http.MethodGet, path)
		if response.Code != 200 {
			t.Fatalf("%s: %d %s", path, response.Code, response.Body.String())
		}
	}
	response := request(router, http.MethodGet, "/api/v1/reference/coverage")
	var result struct {
		Data struct {
			Coverage     string         `json:"coverage"`
			Availability string         `json:"shiny_availability"`
			Counts       map[string]int `json:"counts"`
		} `json:"data"`
	}
	json.Unmarshal(response.Body.Bytes(), &result)
	if result.Data.Coverage != "partial" || result.Data.Availability != "not_verified" || result.Data.Counts["species"] != 3 || result.Data.Counts["forms"] != 3 || result.Data.Counts["games"] != 4 {
		t.Fatal(response.Body.String())
	}
	for _, path := range []string{"/api/v1/reference/species/0", "/api/v1/reference/species/no", "/api/v1/reference/species?limit=0", "/api/v1/reference/forms?species_id=-1", "/api/v1/reference/games?offset=-1"} {
		if response := request(router, http.MethodGet, path); response.Code != 400 {
			t.Fatalf("bad query accepted %s %d", path, response.Code)
		}
	}
	if response := request(router, http.MethodGet, "/api/v1/reference/species/99999"); response.Code != 404 {
		t.Fatal("missing reference not 404")
	}
	response = request(router, http.MethodGet, "/api/v1/reference/species?limit=1&offset=1")
	var list struct {
		Data []struct {
			ID uint64 `json:"id"`
		} `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	json.Unmarshal(response.Body.Bytes(), &list)
	if len(list.Data) != 1 || list.Data[0].ID != 2 || list.Meta.Total != 3 {
		t.Fatal(response.Body.String())
	}
}
