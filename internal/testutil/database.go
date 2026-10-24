// Package testutil creates isolated disposable databases, never user data.
package testutil

import (
	"path/filepath"
	"testing"

	"github.com/zl14229/pokemon-shiny-tracker/internal/database"
	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"gorm.io/gorm"
)

func DB(t testing.TB) *gorm.DB {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "test.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	conn, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return db
}

type References struct {
	Game    model.Game
	Species model.Species
	Other   model.Species
	Form    model.Form
}

func Seed(t testing.TB, db *gorm.DB) References {
	t.Helper()
	r := References{
		Game:    model.Game{Labels: model.Labels{Slug: "test-game", Name: "Test Game"}, Kind: "mainline"},
		Species: model.Species{Labels: model.Labels{Slug: "test-species", Name: "Test Species"}, NationalDex: 25},
		Other:   model.Species{Labels: model.Labels{Slug: "other-species", Name: "Other Species"}, NationalDex: 26},
	}
	for _, v := range []any{&r.Game, &r.Species, &r.Other} {
		if err := db.Create(v).Error; err != nil {
			t.Fatal(err)
		}
	}
	r.Form = model.Form{Labels: model.Labels{Slug: "test-form", Name: "Test Form"}, SpeciesID: r.Species.ID}
	if err := db.Create(&r.Form).Error; err != nil {
		t.Fatal(err)
	}
	return r
}
