package api

import (
	"encoding/json"
	"github.com/zl14229/pokemon-shiny-tracker/internal/service"
	"github.com/zl14229/pokemon-shiny-tracker/internal/testutil"
	"testing"
)

func TestDashboardOfflineAPI(t *testing.T) {
	db := testutil.DB(t)
	r := NewRouter(db)
	w := request(r, "GET", "/api/v1/dashboard")
	var v struct {
		Data service.DashboardResult `json:"data"`
	}
	if e := json.Unmarshal(w.Body.Bytes(), &v); e != nil {
		t.Fatal(e)
	}
	if w.Code != 200 || len(v.Data.Dex) != 4 || v.Data.Dex[0].Summary.Percentage != nil {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, q := range []string{"?game_id=1", "?collection_system=", "?collection_system=a&collection_system=b"} {
		if w := request(r, "GET", "/api/v1/dashboard"+q); w.Code != 400 {
			t.Fatal(q, w.Code)
		}
	}
	conn, _ := db.DB()
	conn.Close()
	if w := request(r, "GET", "/api/v1/dashboard"); w.Code != 500 {
		t.Fatal(w.Code)
	}
}
