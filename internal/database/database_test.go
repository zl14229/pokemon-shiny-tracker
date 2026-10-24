package database_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/zl14229/pokemon-shiny-tracker/internal/database"
	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"github.com/zl14229/pokemon-shiny-tracker/internal/testutil"
)

func ptr[T any](v T) *T { return &v }

func TestInitializeReopenAndRelationships(t *testing.T) {
	db := testutil.DB(t)
	r := testutil.Seed(t, db)
	var version, fk int
	db.Raw("PRAGMA user_version").Scan(&version)
	db.Raw("PRAGMA foreign_keys").Scan(&fk)
	if version != database.CurrentVersion || fk != 1 {
		t.Fatalf("version=%d foreign_keys=%d", version, fk)
	}
	hunt := model.Hunt{SpeciesID: r.Species.ID, FormID: &r.Form.ID, GameID: &r.Game.ID}
	if err := db.Create(&hunt).Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		rec := model.ShinyRecord{HuntID: &hunt.ID, CurrentSpeciesID: r.Species.ID}
		if err := db.Create(&rec).Error; err != nil {
			t.Fatal(err)
		}
	}
	unknown := model.ShinyRecord{CurrentSpeciesID: r.Other.ID}
	if err := db.Create(&unknown).Error; err != nil {
		t.Fatal(err)
	}
	var got model.ShinyRecord
	db.First(&got, unknown.ID)
	if got.HuntID != nil || got.OriginalSpeciesID != nil || got.OriginalGameID != nil || got.ObtainedDate != nil {
		t.Fatal("unknowns were invented")
	}
	history := model.ShinyHistory{ShinyRecordID: unknown.ID, Kind: "registered", BeforeJSON: "{}", AfterJSON: "{}"}
	if err := db.Create(&history).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Unscoped().Delete(&unknown).Error; err == nil {
		t.Fatal("history identity was deleted")
	}
	if err := db.Unscoped().Delete(&hunt).Error; err == nil {
		t.Fatal("hunt with outcomes was deleted")
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	var count int64
	db.Model(&model.ShinyRecord{}).Where("hunt_id = ?", hunt.ID).Count(&count)
	if count != 2 {
		t.Fatal(count)
	}
}

func TestForeignKeysChecksAndFormOwnership(t *testing.T) {
	db := testutil.DB(t)
	r := testutil.Seed(t, db)
	cases := []any{
		&model.Hunt{SpeciesID: 99999},
		&model.Hunt{SpeciesID: r.Other.ID, FormID: &r.Form.ID},
		&model.Hunt{SpeciesID: r.Species.ID, Status: "bogus"},
		&model.Hunt{SpeciesID: r.Species.ID, EncounterCount: ptr(int64(-1)), CountUnit: ptr("eggs")},
		&model.Hunt{SpeciesID: r.Species.ID, EncounterCount: ptr(int64(0))},
		&model.Hunt{SpeciesID: r.Species.ID, DurationSeconds: ptr(int64(-1))},
		&model.ShinyRecord{CurrentSpeciesID: r.Other.ID, CurrentFormID: &r.Form.ID},
		&model.ShinyRecord{CurrentSpeciesID: r.Species.ID, OriginalFormID: &r.Form.ID},
		&model.ShinyRecord{CurrentSpeciesID: r.Species.ID, OriginalSpeciesID: &r.Other.ID, OriginalFormID: &r.Form.ID},
		&model.ShinyRecord{CurrentSpeciesID: r.Species.ID, HuntID: ptr(uint64(999))},
		&model.ShinyRecord{CurrentSpeciesID: r.Species.ID, Level: ptr(101)},
	}
	for i, v := range cases {
		if err := db.Create(v).Error; err == nil {
			t.Fatalf("invalid case %d accepted", i)
		}
	}
	hunt := model.Hunt{SpeciesID: r.Species.ID, EncounterCount: ptr(int64(0)), CountUnit: ptr("eggs")}
	if err := db.Create(&hunt).Error; err != nil {
		t.Fatal(err)
	}
}

func TestPathSafetyAndUnrecognizedDatabase(t *testing.T) {
	if _, err := database.Open("relative.sqlite3"); err == nil {
		t.Fatal("relative accepted")
	}
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Open(filepath.Join(dir, "private.sqlite3")); err == nil {
		t.Fatal("checkout accepted")
	}
	link := filepath.Join(t.TempDir(), "checkout-link")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Open(filepath.Join(link, "new", "private.sqlite3")); err == nil {
		t.Fatal("symlinked checkout accepted")
	}
	path := filepath.Join(t.TempDir(), "existing.sqlite3")
	raw, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec("CREATE TABLE personal(value TEXT); INSERT INTO personal VALUES('preserved')"); err != nil {
		t.Fatal(err)
	}
	raw.Close()
	before, _ := os.ReadFile(path)
	if _, err := database.Open(path); err == nil {
		t.Fatal("unknown existing database accepted")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("unknown database modified")
	}
	if err := os.Symlink(path, path+"-link"); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Open(path + "-link"); err == nil {
		t.Fatal("symlink accepted")
	}
	raw, _ = sql.Open("sqlite3", path)
	raw.Exec("PRAGMA user_version = 999")
	raw.Close()
	before, _ = os.ReadFile(path)
	if _, err := database.Open(path); err == nil {
		t.Fatal("future version accepted")
	}
	after, _ = os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("future database modified")
	}
	raw, _ = sql.Open("sqlite3", path)
	raw.Exec("PRAGMA user_version = 1")
	raw.Close()
	before, _ = os.ReadFile(path)
	if _, err := database.Open(path); err == nil {
		t.Fatal("incomplete known schema accepted")
	}
	after, _ = os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("incomplete schema modified")
	}
}
