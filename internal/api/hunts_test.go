package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"github.com/zl14229/pokemon-shiny-tracker/internal/testutil"
)

func jsonRequest(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestHuntHTTPCRUD(t *testing.T) {
	db := testutil.DB(t)
	testutil.Seed(t, db)
	router := NewRouter(db)
	response := jsonRequest(router, "POST", "/api/v1/hunts", `{"species_id":1,"encounter_count":0,"count_unit":"eggs"}`)
	if response.Code != 201 {
		t.Fatal(response.Code, response.Body.String())
	}
	var result struct {
		Data model.Hunt `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Data.ID != 1 || result.Data.Status != "in_progress" || result.Data.EncounterCount == nil {
		t.Fatal(response.Body.String())
	}
	for _, path := range []string{"/api/v1/hunts", "/api/v1/hunts/1"} {
		if w := request(router, "GET", path); w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	if w := jsonRequest(router, "PUT", "/api/v1/hunts/1", `{"species_id":1,"status":"completed"}`); w.Code != 409 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := jsonRequest(router, "PUT", "/api/v1/hunts/1", `{"species_id":2,"status":"abandoned"}`); w.Code != 200 || !strings.Contains(w.Body.String(), `"encounter_count":null`) {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := request(router, "DELETE", "/api/v1/hunts/1"); w.Code != 204 || w.Body.Len() != 0 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := request(router, "GET", "/api/v1/hunts/1"); w.Code != 404 {
		t.Fatal(w.Code)
	}
	if w := request(router, "GET", "/api/v1/hunts"); !strings.Contains(w.Body.String(), `"data":[]`) {
		t.Fatal(w.Body.String())
	}
}
func TestHuntHTTPRejectsInvalidInputs(t *testing.T) {
	db := testutil.DB(t)
	testutil.Seed(t, db)
	router := NewRouter(db)
	for _, payload := range []string{`null`, `[]`, `{}`, `{"species_id":1,"id":99}`, `{"species_id":1} {}`, `{"species_id":1,"encounter_count":1.5}`, `{"species_id":1,"encounter_count":9223372036854775808}`, `{"species_id":1,"form_id":-1}`, `{"species_id":1,"duration_seconds":-1}`, `{"species_id":1,"status":"done"}`, `{"species_id":1,"start_date":"2023-02-29"}`} {
		if w := jsonRequest(router, "POST", "/api/v1/hunts", payload); w.Code != 400 {
			t.Fatal(payload, w.Code, w.Body.String())
		}
	}
	if w := jsonRequest(router, "POST", "/api/v1/hunts", `{"species_id":1,"notes":"`+strings.Repeat("x", 65536)+`"}`); w.Code != 413 {
		t.Fatal(w.Code)
	}
	if w := jsonRequest(router, "POST", "/api/v1/hunts", `{"species_id":1,"notes":"`+strings.Repeat("宝", 5001)+`"}`); w.Code != 400 {
		t.Fatal(w.Code)
	}
	r := httptest.NewRequest("POST", "/api/v1/hunts", strings.NewReader(`{"species_id":1}`))
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != 415 {
		t.Fatal(w.Code)
	}
	for _, path := range []string{"/api/v1/hunts/0", "/api/v1/hunts/-1", "/api/v1/hunts/9223372036854775808"} {
		if w := request(router, "GET", path); w.Code != 400 {
			t.Fatal(path, w.Code)
		}
	}
	if w := request(router, "DELETE", "/api/v1/hunts/999"); w.Code != 404 {
		t.Fatal(w.Code)
	}
	// A database failure uses the established redacted error contract.
	conn, _ := db.DB()
	conn.Close()
	w = jsonRequest(router, "POST", "/api/v1/hunts", `{"species_id":1}`)
	if w.Code != 500 || strings.Contains(w.Body.String(), "database") {
		t.Fatal(w.Code, w.Body.String())
	}
}
