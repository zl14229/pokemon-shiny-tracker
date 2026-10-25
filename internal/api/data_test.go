package api

import (
	"context"
	"encoding/json"
	"github.com/zl14229/pokemon-shiny-tracker/internal/backup"
	"github.com/zl14229/pokemon-shiny-tracker/internal/testutil"
	"os"
	"path/filepath"
	"testing"
)

func TestDataDownloadAndNoRestoreAPI(t *testing.T) {
	db := testutil.DB(t)
	testutil.Seed(t, db)
	r := NewRouter(db)
	w := request(r, "GET", "/api/v1/data/export")
	if w.Code != 200 || w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal(w.Code, w.Body.String())
	}
	var v map[string]any
	if json.Unmarshal(w.Body.Bytes(), &v) != nil || v["format"] != "pokemon-shiny-tracker-export-v1" {
		t.Fatal("invalid export")
	}
	w = request(r, "GET", "/api/v1/data/backup")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	path := filepath.Join(t.TempDir(), "backup.zip")
	if e := os.WriteFile(path, w.Body.Bytes(), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := backup.Verify(context.Background(), path); e != nil {
		t.Fatal(e)
	}
	if w = jsonRequest(r, "POST", "/api/v1/data/restore", `{}`); w.Code != 404 {
		t.Fatal("unsafe restore endpoint exists")
	}
}
