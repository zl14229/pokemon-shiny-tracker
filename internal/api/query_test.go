package api

import (
	"encoding/json"
	"testing"

	"github.com/zl14229/pokemon-shiny-tracker/internal/testutil"
)

func TestListHTTPPagingAndBadQueries(t *testing.T) {
	db := testutil.DB(t)
	testutil.Seed(t, db)
	router := NewRouter(db)
	for _, path := range []string{"hunts", "shiny-records"} {
		w := request(router, "GET", "/api/v1/"+path+"?page=2&page_size=1&sort=id&order=asc")
		var response struct {
			Data []any `json:"data"`
			Meta struct {
				Total int `json:"total"`
				Page  int `json:"page"`
				Size  int `json:"page_size"`
			} `json:"meta"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if w.Code != 200 || response.Data == nil || len(response.Data) != 0 || response.Meta.Page != 2 || response.Meta.Size != 1 || response.Meta.Total != 0 {
			t.Fatal(w.Code, w.Body.String())
		}
		for _, query := range []string{"page=0", "page_size=101", "sort=bad", "page=1&page=2", "q=%ZZ", "limit=100", "unknown=1"} {
			w = request(router, "GET", "/api/v1/"+path+"?"+query)
			if w.Code != 400 {
				t.Fatal(path, query, w.Code, w.Body.String())
			}
		}
	}
}
