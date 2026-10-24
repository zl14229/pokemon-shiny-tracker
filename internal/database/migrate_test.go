package database

import (
	"errors"
	"path/filepath"
	"testing"

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
	err = migrate(db, []migration{{2, func(tx *gorm.DB) error {
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
	if version != 1 {
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
