package database

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const CurrentVersion = 1

// ValidatePath rejects relative paths, file symlinks and files inside Git checkouts.
func ValidatePath(path string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return errors.New("database path must be a clean absolute path outside a Git checkout")
	}
	if st, err := os.Lstat(path); err == nil && !st.Mode().IsRegular() {
		return errors.New("database path must be a regular file, not a symlink")
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	parent := filepath.Dir(path)
	var tail []string
	for {
		_, err := os.Stat(parent)
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return err
		}
		tail = append([]string{filepath.Base(parent)}, tail...)
		parent = filepath.Dir(parent)
	}
	resolved, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return err
	}
	parent = filepath.Join(append([]string{resolved}, tail...)...)
	for {
		if _, err := os.Lstat(filepath.Join(parent, ".git")); err == nil {
			return errors.New("database must be stored outside Git checkouts")
		}
		next := filepath.Dir(parent)
		if next == parent {
			break
		}
		parent = next
	}
	return nil
}

func dsn(path string, query string) string {
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(path), RawQuery: query}
	return u.String()
}

func preflight(path string) error {
	db, err := sql.Open("sqlite3", dsn(path, "mode=ro"))
	if err != nil {
		return err
	}
	defer db.Close()
	var version, tables int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version > CurrentVersion {
		return errors.New("database schema is newer than this application")
	}
	if err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&tables); err != nil {
		return err
	}
	if version == 0 && tables != 0 {
		return errors.New("refusing to migrate an unrecognized existing database")
	}
	if version == 1 {
		var known int
		if err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name IN ('games','species','forms','hunts','shiny_records','shiny_histories')").Scan(&known); err != nil {
			return err
		}
		if known != 6 {
			return errors.New("recognized database schema is incomplete")
		}
	}
	return nil
}

// Open initializes only empty/new databases; recognized versions migrate atomically.
func Open(path string) (*gorm.DB, error) {
	if err := ValidatePath(path); err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); err == nil {
		if err := preflight(path); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	db, err := gorm.Open(sqlite.Open(dsn(path, "_foreign_keys=on&_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_txlock=immediate")), NewConfig())
	if err != nil {
		return nil, err
	}
	conn, err := db.DB()
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)
	if err := Migrate(db); err != nil {
		_ = conn.Close()
		return nil, err
	}
	var violations []struct{ Table string }
	if err := db.Raw("PRAGMA foreign_key_check").Scan(&violations).Error; err != nil || len(violations) > 0 {
		_ = conn.Close()
		return nil, fmt.Errorf("database foreign key validation failed")
	}
	return db, nil
}
