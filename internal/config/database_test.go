package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDatabaseConfigIsReadOnlyAndOutsideCheckout(t *testing.T) {
	t.Setenv("SHINY_HTTP_PORT", "8080")
	t.Setenv("SHINY_DB_PATH", "")
	if err := os.Unsetenv("SHINY_DB_PATH"); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DBPath != filepath.Join(dir, "pokemon-shiny-tracker", "tracker.sqlite3") {
		t.Fatal(cfg.DBPath)
	}
	if _, err := os.Stat(cfg.DBPath); !os.IsNotExist(err) {
		t.Fatal("config created a database")
	}
	for _, value := range []string{"", "relative.sqlite3"} {
		t.Setenv("SHINY_DB_PATH", value)
		if _, err := Load(); err == nil {
			t.Fatal("invalid DB path accepted")
		}
	}
}
