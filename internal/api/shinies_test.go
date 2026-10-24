package api

import (
	"strings"
	"testing"

	"github.com/zl14229/pokemon-shiny-tracker/internal/testutil"
)

func TestShinyHTTPIndependentAndAssociatedCRUD(t *testing.T) {
	db := testutil.DB(t)
	testutil.Seed(t, db)
	router := NewRouter(db)
	w := jsonRequest(router, "POST", "/api/v1/shiny-records", `{"current_species_id":1,"obtained_date":"2018","source":"trade","holding_status":"held"}`)
	if w.Code != 201 || !strings.Contains(w.Body.String(), `"hunt_id":null`) {
		t.Fatal(w.Code, w.Body.String())
	}
	w = jsonRequest(router, "PUT", "/api/v1/shiny-records/1", `{"current_species_id":2,"source":"trade","holding_status":"traded","change_type":"evolution"}`)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, path := range []string{"/api/v1/shiny-records", "/api/v1/shiny-records/1", "/api/v1/shiny-records/1/history?limit=1&offset=1"} {
		if w = request(router, "GET", path); w.Code != 200 {
			t.Fatal(path, w.Code, w.Body.String())
		}
	}
	if w = request(router, "DELETE", "/api/v1/shiny-records/1"); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = request(router, "GET", "/api/v1/shiny-records/1"); w.Code != 404 {
		t.Fatal(w.Code)
	}
	w = request(router, "GET", "/api/v1/shiny-records/1/history")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"record_deleted":true`) || !strings.Contains(w.Body.String(), `"kind":"deleted"`) {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = jsonRequest(router, "POST", "/api/v1/hunts", `{"species_id":1}`); w.Code != 201 {
		t.Fatal(w.Body.String())
	}
	if w = jsonRequest(router, "POST", "/api/v1/shiny-records", `{"current_species_id":2,"hunt_id":1}`); w.Code != 201 {
		t.Fatal(w.Body.String())
	}
	if w = jsonRequest(router, "PUT", "/api/v1/hunts/1", `{"species_id":1,"status":"completed"}`); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w = request(router, "DELETE", "/api/v1/shiny-records/2"); w.Code != 409 {
		t.Fatal("last outcome deleted", w.Code)
	}
}
func TestShinyHTTPValidation(t *testing.T) {
	db := testutil.DB(t)
	testutil.Seed(t, db)
	router := NewRouter(db)
	for _, payload := range []string{`{}`, `null`, `{"current_species_id":1,"hunt_id":0}`, `{"current_species_id":1,"level":0}`, `{"current_species_id":1,"original_form_id":1}`, `{"current_species_id":2,"current_form_id":1}`, `{"current_species_id":1,"original_game_id":999}`, `{"current_species_id":1,"source":"caught"}`, `{"current_species_id":1,"holding_status":"lost"}`, `{"current_species_id":1,"id":42}`} {
		w := jsonRequest(router, "POST", "/api/v1/shiny-records", payload)
		if w.Code != 400 {
			t.Fatal(payload, w.Code, w.Body.String())
		}
	}
	for _, path := range []string{"/api/v1/shiny-records/0", "/api/v1/shiny-records/1/history?limit=501", "/api/v1/shiny-records/1/history?offset=-1"} {
		if w := request(router, "GET", path); w.Code != 400 {
			t.Fatal(path, w.Code)
		}
	}
	if w := request(router, "GET", "/api/v1/shiny-records/999/history"); w.Code != 404 {
		t.Fatal(w.Code)
	}
}
