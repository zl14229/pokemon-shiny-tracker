package service

import (
	"context"
	"reflect"
	"testing"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"gorm.io/gorm/clause"
)

func TestDashboardReusesDexAndPermanentIdentities(t *testing.T) {
	f := dexSeed(t)
	r := createDexRecord(t, f, f.a, &f.fa.ID, &f.game.ID, "held")
	r = updateDexRecord(t, f, r, func(in *ShinyInput) {
		in.CurrentSpeciesID = f.b.ID
		in.CurrentFormID = &f.fb.ID
		in.ChangeType = "evolution"
	})
	updateDexRecord(t, f, r, func(in *ShinyInput) { in.CurrentGameID = &f.other.ID; in.Source = "self_hunted" })
	createDexRecord(t, f, f.a, &f.regional.ID, &f.game.ID, "released")
	createDexRecord(t, f, f.a, nil, nil, "unknown")
	createDexRecord(t, f, f.c, &f.fc.ID, &f.sleep.ID, "held")
	del := createDexRecord(t, f, f.d, &f.fd.ID, &f.game.ID, "held")
	if err := (Shinies{DB: f.db}).Delete(context.Background(), del.ID); err != nil {
		t.Fatal(err)
	}
	zero := int64(0)
	for i, status := range []string{"in_progress", "completed", "abandoned"} {
		h := model.Hunt{SpeciesID: f.a.ID, GameID: &f.game.ID, Status: status}
		if i == 1 {
			h.EncounterCount = &zero
			h.DurationSeconds = &zero
			unit := "encounters"
			h.CountUnit = &unit
		}
		if e := f.db.Omit(clause.Associations).Create(&h).Error; e != nil {
			t.Fatal(e)
		}
	}
	v, e := (Analytics{DB: f.db}).Dashboard(context.Background(), "")
	if e != nil {
		t.Fatal(e)
	}
	if v.Collection.Historical != 3 || v.Collection.Held != 1 || v.Collection.UnknownHolding != 1 || v.Collection.SelfHunted != 1 || v.Collection.UnknownOriginalGame != 1 || v.Hunts.Total != 3 || v.Hunts.UnknownCount != 2 || v.Hunts.UnknownDuration != 2 {
		t.Fatalf("%+v", v)
	}
	for _, d := range v.Dex {
		want := getDex(t, f, "dimension="+d.Dimension+"&basis="+d.Basis)
		if !reflect.DeepEqual(d.Summary, want.Summary) || !reflect.DeepEqual(d.Coverage, want.Coverage) {
			t.Fatal("dashboard diverged from M4", d, want.Summary)
		}
	}
	sleep, e := (Analytics{DB: f.db}).Dashboard(context.Background(), "collection_system=sleep")
	if e != nil || sleep.Collection.Historical != 1 || sleep.Collection.Held != 1 || sleep.Hunts.Total != 0 {
		t.Fatal(sleep, e)
	}
}

func TestDashboardUnknownDenominatorAndStrictQuery(t *testing.T) {
	f := dexSeed(t)
	if e := f.db.Where("1 = 1").Delete(&model.ShinyAvailability{}).Error; e != nil {
		t.Fatal(e)
	}
	v, e := (Analytics{DB: f.db}).Dashboard(context.Background(), "")
	if e != nil || v.Collection.Historical != 0 || v.Hunts.Total != 0 {
		t.Fatal(v, e)
	}
	for _, d := range v.Dex {
		if d.Summary.Percentage != nil {
			t.Fatal("unknown denominator became zero")
		}
	}
	for _, q := range []string{"basis=current", "collection_system=", "collection_system=a&collection_system=b", "collection_system=%FF", "%ZZ"} {
		if _, e := (Analytics{DB: f.db}).Dashboard(context.Background(), q); e == nil {
			t.Fatal(q)
		}
	}
}
