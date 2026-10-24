package database

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMigrationRollbackAndReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db.sqlite3")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	defer conn.Close()
	err = migrate(db, []migration{{CurrentVersion + 1, func(tx *gorm.DB) error {
		if err := tx.Exec("CREATE TABLE rollback_probe(id INTEGER)").Error; err != nil {
			return err
		}
		return errors.New("injected failure")
	}}})
	if err == nil {
		t.Fatal("migration should fail")
	}
	if db.Migrator().HasTable("rollback_probe") {
		t.Fatal("failed migration persisted")
	}
	var version int
	db.Raw("PRAGMA user_version").Scan(&version)
	if version != CurrentVersion {
		t.Fatal(version)
	}
	conn.Close()
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	reopenConn, _ := reopened.DB()
	defer reopenConn.Close()
	if !reopened.Migrator().HasTable("shiny_records") {
		t.Fatal("reopen lost schema")
	}
}

func TestVersionOneUpgradePreservesPersonalData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.sqlite3")
	db, err := gorm.Open(sqlite.Open(dsn(path, "_foreign_keys=on")), NewConfig())
	if err != nil {
		t.Fatal(err)
	}
	if err := migrate(db, migrations[:1]); err != nil {
		t.Fatal(err)
	}
	species := model.Species{Labels: model.Labels{Slug: "private-reference", Name: "Private Reference"}, NationalDex: 25}
	if err := db.Create(&species).Error; err != nil {
		t.Fatal(err)
	}
	record := model.ShinyRecord{CurrentSpeciesID: species.ID}
	if err := db.Create(&record).Error; err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	conn.Close()
	upgraded, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	current, _ := upgraded.DB()
	defer current.Close()
	var got model.ShinyRecord
	if err := upgraded.First(&got, record.ID).Error; err != nil || got.CurrentSpeciesID != species.ID {
		t.Fatal("personal identity changed", err)
	}
	if !upgraded.Migrator().HasTable("external_references") {
		t.Fatal("reference migration missing")
	}
}
