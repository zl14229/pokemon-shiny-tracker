package service

import (
	"context"
	"errors"
	"testing"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"github.com/zl14229/pokemon-shiny-tracker/internal/testutil"
	"gorm.io/gorm"
)

func ptr[T any](v T) *T { return &v }
func TestHuntLifecycleAndOutcomes(t *testing.T) {
	db := testutil.DB(t)
	r := testutil.Seed(t, db)
	s := Hunts{db}
	ctx := context.Background()
	h, err := s.Create(ctx, HuntInput{SpeciesID: r.Species.ID})
	if err != nil {
		t.Fatal(err)
	}
	if h.Status != "in_progress" || h.EncounterCount != nil || h.DurationSeconds != nil || h.GameID != nil {
		t.Fatal("unknown values inferred", h)
	}
	in := HuntInput{SpeciesID: r.Species.ID, Status: "completed"}
	if _, err = s.Update(ctx, h.ID, in); err == nil {
		t.Fatal("empty hunt completed")
	}
	if _, err = s.Create(ctx, in); err == nil {
		t.Fatal("completed hunt created without outcomes")
	}
	outcomes := []model.ShinyRecord{{HuntID: &h.ID, CurrentSpeciesID: r.Species.ID}, {HuntID: &h.ID, CurrentSpeciesID: r.Other.ID}}
	for i := range outcomes {
		if err := db.Create(&outcomes[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	in.EncounterCount = ptr(int64(0))
	in.CountUnit = ptr("eggs")
	in.DurationSeconds = ptr(int64(0))
	in.StartDate = ptr("2024")
	in.EndDate = ptr("2024-01")
	in.FormID = &r.Form.ID
	in.GameID = &r.Game.ID
	for _, state := range []string{"completed", "abandoned", "in_progress", "completed"} {
		in.Status = state
		updated, err := s.Update(ctx, h.ID, in)
		if err != nil || updated.Status != state || updated.EncounterCount == nil || *updated.EncounterCount != 0 {
			t.Fatal(state, updated, err)
		}
	}
	var n int64
	db.Model(&model.ShinyRecord{}).Where("hunt_id = ?", h.ID).Count(&n)
	if n != 2 {
		t.Fatal("outcomes lost on transition")
	}
	if err = s.Delete(ctx, h.ID); err == nil {
		t.Fatal("linked hunt deleted")
	}
	db.Where("hunt_id = ?", h.ID).Delete(&model.ShinyRecord{})
	if _, err = s.Update(ctx, h.ID, in); err == nil {
		t.Fatal("deleted outcomes counted for completion")
	}
	if err = s.Delete(ctx, h.ID); err == nil {
		t.Fatal("historical links orphaned")
	}
	other, err := s.Create(ctx, HuntInput{SpeciesID: r.Other.ID, Status: "abandoned"})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Delete(ctx, other.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Get(ctx, other.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("deleted record visible", err)
	}
	if err = s.Delete(ctx, other.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("repeated delete", err)
	}
	rows, total, err := s.List(ctx, "")
	if err != nil || len(rows) != 1 || total.Total != 1 {
		t.Fatal(rows, total, err)
	}
}
func TestHuntValidationAndAtomicFailure(t *testing.T) {
	db := testutil.DB(t)
	r := testutil.Seed(t, db)
	s := Hunts{db}
	ctx := context.Background()
	h, err := s.Create(ctx, HuntInput{SpeciesID: r.Species.ID, Notes: ptr("original")})
	if err != nil {
		t.Fatal(err)
	}
	storage := model.Game{Labels: model.Labels{Slug: "home", Name: "HOME"}, Kind: "storage"}
	if err = db.Create(&storage).Error; err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*HuntInput){
		"species0": func(in *HuntInput) { in.SpeciesID = 0 }, "missingSpecies": func(in *HuntInput) { in.SpeciesID = 9999 },
		"wrongForm": func(in *HuntInput) { in.SpeciesID = r.Other.ID; in.FormID = &r.Form.ID }, "missingGame": func(in *HuntInput) { in.GameID = ptr(uint64(9999)) }, "storage": func(in *HuntInput) { in.GameID = &storage.ID },
		"negative": func(in *HuntInput) { in.EncounterCount = ptr(int64(-1)); in.CountUnit = ptr("eggs") }, "unitMissing": func(in *HuntInput) { in.EncounterCount = ptr(int64(0)) }, "unitBlank": func(in *HuntInput) { in.EncounterCount = ptr(int64(1)); in.CountUnit = ptr("  ") },
		"duration": func(in *HuntInput) { in.DurationSeconds = ptr(int64(-1)) }, "invalidDate": func(in *HuntInput) { in.StartDate = ptr("2025-02-29") }, "emptyDate": func(in *HuntInput) { in.StartDate = ptr("") }, "reverseDate": func(in *HuntInput) { in.StartDate = ptr("2025-03"); in.EndDate = ptr("2025-02") }, "status": func(in *HuntInput) { in.Status = "done" }, "missingUpdateStatus": func(in *HuntInput) { in.Status = "" },
	} {
		t.Run(name, func(t *testing.T) {
			in := HuntInput{SpeciesID: r.Species.ID, Status: "in_progress", Notes: ptr("changed")}
			mutate(&in)
			if _, err = s.Update(ctx, h.ID, in); err == nil {
				t.Fatal("invalid input accepted")
			}
			actual, err := s.Get(ctx, h.ID)
			if err != nil || *actual.Notes != "original" {
				t.Fatal("failed validation wrote data", actual, err)
			}
		})
	}
	in := HuntInput{SpeciesID: r.Species.ID, Status: "in_progress", CountUnit: ptr(" encounters ")}
	updated, err := s.Update(ctx, h.ID, in)
	if err != nil || updated.Notes != nil || updated.CountUnit == nil || *updated.CountUnit != "encounters" || updated.EncounterCount != nil {
		t.Fatal("PUT null/unknown semantics", updated, err)
	}
	if _, err = s.Update(ctx, 9999, in); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal(err)
	}
}
func TestPartialDateBounds(t *testing.T) {
	for _, v := range []string{"2024", "2024-02", "2024-02-29", "0001", "9999-12-31"} {
		lo, hi, err := DateBounds(v)
		if err != nil || lo.After(hi) {
			t.Fatal(v, lo, hi, err)
		}
	}
	for _, v := range []string{"0000", "2024-2", "2024-02-30", "2024-13", "", "2024-01-01Z"} {
		if _, _, err := DateBounds(v); err == nil {
			t.Fatal(v)
		}
	}
}
