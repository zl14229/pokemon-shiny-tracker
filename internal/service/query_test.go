package service

import (
	"context"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"github.com/zl14229/pokemon-shiny-tracker/internal/testutil"
)

func TestHuntCombinedFiltersLiteralSearchAndDates(t *testing.T) {
	db := testutil.DB(t)
	r := testutil.Seed(t, db)
	s := Hunts{db}
	ctx := context.Background()
	inputs := []HuntInput{
		{SpeciesID: r.Species.ID, FormID: &r.Form.ID, GameID: &r.Game.ID, Method: ptr("eggs"), Status: "in_progress", EncounterCount: ptr(int64(0)), CountUnit: ptr("eggs"), DurationSeconds: ptr(int64(60)), StartDate: ptr("2024"), Notes: ptr(`100% _ O'Brien \ 宝可梦`)},
		{SpeciesID: r.Species.ID, GameID: &r.Game.ID, Method: ptr("eggs"), Status: "abandoned", EncounterCount: ptr(int64(10)), CountUnit: ptr("eggs"), StartDate: ptr("2024-02")},
		{SpeciesID: r.Other.ID, Method: ptr("encounter"), EncounterCount: ptr(int64(1)), CountUnit: ptr("encounters"), StartDate: ptr("2024-03-01")},
		{SpeciesID: r.Species.ID, StartDate: ptr("9999-12")},
		{SpeciesID: r.Species.ID},
	}
	for _, in := range inputs {
		if _, err := s.Create(ctx, in); err != nil {
			t.Fatal(err)
		}
	}
	result := model.ShinyRecord{HuntID: ptr(uint64(1)), CurrentSpeciesID: r.Other.ID}
	if err := db.Create(&result).Error; err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		query string
		ids   []uint64
	}{
		{"species_id=1&form_id=1&game_id=1&status=in_progress&method=eggs&count_unit=eggs&min_count=0&max_count=0&min_duration=60&max_duration=60&has_results=true", []uint64{1}},
		{"count_unit=eggs&min_count=1&sort=encounter_count&order=asc", []uint64{2}},
		{"has_results=true", []uint64{1}},
		{"start_from=2024-02-29&start_to=2024-02-29&sort=id&order=asc", []uint64{1, 2}},
		{"start_from=2024-03&start_to=2024-03&sort=id&order=asc", []uint64{1, 3}},
		{"start_from=9999-12-31&start_to=9999-12-31", []uint64{4}},
		{"q=" + url.QueryEscape("%"), []uint64{1}}, {"q=" + url.QueryEscape("_"), []uint64{1}}, {"q=" + url.QueryEscape(`\`), []uint64{1}}, {"q=" + url.QueryEscape("O'Brien"), []uint64{1}}, {"q=" + url.QueryEscape("宝可梦"), []uint64{1}},
		{"q=" + url.QueryEscape("' OR 1=1 --"), []uint64{}},
		{"q=Other+Species", []uint64{3}},
		{"species_id=99999", []uint64{}},
	} {
		t.Run(tc.query, func(t *testing.T) {
			rows, meta, err := s.List(ctx, tc.query)
			ids := []uint64{}
			for _, row := range rows {
				ids = append(ids, row.ID)
			}
			if err != nil || !reflect.DeepEqual(ids, tc.ids) || meta.Total != int64(len(tc.ids)) {
				t.Fatal(ids, meta, err)
			}
		})
	}
	db.Delete(&result)
	rows, _, err := s.List(ctx, "has_results=true")
	if err != nil || len(rows) != 0 {
		t.Fatal("soft-deleted outcomes included", rows, err)
	}
	if err := s.Delete(ctx, 5); err != nil {
		t.Fatal(err)
	}
	rows, meta, err := s.List(ctx, "")
	if err != nil || len(rows) != 4 || meta.Total != 4 {
		t.Fatal("deleted hunt visible", rows, meta, err)
	}
}
func TestShinyOriginalCurrentFiltersAndStablePages(t *testing.T) {
	db := testutil.DB(t)
	r := testutil.Seed(t, db)
	ctx := context.Background()
	s := Shinies{db}
	hs := Hunts{db}
	h, err := hs.Create(ctx, HuntInput{SpeciesID: r.Species.ID})
	if err != nil {
		t.Fatal(err)
	}
	otherGame := model.Game{Labels: model.Labels{Slug: "other-game", Name: "Other Game"}}
	if err = db.Create(&otherGame).Error; err != nil {
		t.Fatal(err)
	}
	for _, in := range []ShinyInput{
		{CurrentSpeciesID: r.Other.ID, OriginalSpeciesID: &r.Species.ID, OriginalFormID: &r.Form.ID, OriginalGameID: &r.Game.ID, CurrentGameID: &otherGame.ID, Source: "trade", HoldingStatus: "traded", Nickname: ptr("100%_ 宝"), ObtainedDate: ptr("2024"), Level: ptr(10)},
		{CurrentSpeciesID: r.Species.ID, CurrentFormID: &r.Form.ID, OriginalSpeciesID: &r.Other.ID, OriginalGameID: &otherGame.ID, CurrentGameID: &r.Game.ID, Source: "self_hunted", HoldingStatus: "held", HuntID: &h.ID, ObtainedDate: ptr("2024-02"), Level: ptr(10)},
		{CurrentSpeciesID: r.Species.ID, HoldingStatus: "released", ObtainedDate: ptr("2024-03-01")},
		{CurrentSpeciesID: r.Other.ID},
	} {
		if _, err := s.Create(ctx, in); err != nil {
			t.Fatal(err)
		}
	}
	// Identical primary sort values must use a stable ID tie break.
	stamp := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err = db.Model(&model.ShinyRecord{}).Where("1=1").UpdateColumn("created_at", stamp).Error; err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		q   string
		ids []uint64
	}{
		{"original_species_id=1&current_species_id=2&original_form_id=1&original_game_id=1&current_game_id=2&source=trade&holding_status=traded&has_hunt=false", []uint64{1}},
		{"current_form_id=1&hunt_id=1&has_hunt=true", []uint64{2}},
		{"original_species_id=2&current_species_id=2", []uint64{}},
		{"has_hunt=false&hunt_id=1", []uint64{}},
		{"q=" + url.QueryEscape("%_"), []uint64{1}},
		{"obtained_from=2024-02-29&obtained_to=2024-02-29&sort=id&order=asc", []uint64{1, 2}},
		{"sort=level&order=asc", []uint64{1, 2, 3, 4}}, {"sort=level&order=desc", []uint64{2, 1, 4, 3}},
		{"sort=created_at&order=asc&page_size=2&page=1", []uint64{1, 2}}, {"sort=created_at&order=asc&page_size=2&page=2", []uint64{3, 4}},
		{"sort=created_at&order=desc&page_size=2&page=1", []uint64{4, 3}}, {"sort=created_at&order=desc&page_size=2&page=2", []uint64{2, 1}},
		{"page=3&page_size=2", []uint64{}},
	} {
		t.Run(tc.q, func(t *testing.T) {
			rows, meta, err := s.List(ctx, tc.q)
			ids := []uint64{}
			for _, row := range rows {
				ids = append(ids, row.ID)
			}
			if err != nil || !reflect.DeepEqual(ids, tc.ids) {
				t.Fatal(ids, meta, err)
			}
			if meta.PageSize == 2 && meta.Total != 4 {
				t.Fatal("paging total counted page only", meta)
			}
		})
	}
	if err = s.Delete(ctx, 1); err != nil {
		t.Fatal(err)
	}
	rows, meta, err := s.List(ctx, "q="+url.QueryEscape("%_"))
	if err != nil || len(rows) != 0 || meta.Total != 0 {
		t.Fatal("deleted record searched", rows, meta, err)
	}
}
func TestListQueryBoundaries(t *testing.T) {
	for _, kind := range []string{"hunt", "shiny"} {
		for _, raw := range []string{"page=0", "page=-1", "page=1.2", "page=2147483647&page_size=100", "page_size=0", "page_size=101", "sort=notes;DROP+TABLE+hunts", "order=DESC", "wat=1", "page=1&page=2", "q=", "q=%ZZ", "q=%FF"} {
			if _, err := parseList(raw, kind); err == nil {
				t.Fatal(kind, raw, "accepted")
			}
		}
	}
	for _, raw := range []string{"status=done", "species_id=0", "species_id=9223372036854775808", "has_results=1", "min_count=-1", "min_count=0", "sort=encounter_count", "count_unit=eggs&min_count=2&max_count=1", "start_from=2023-02-29", "start_from=2025&start_to=2024", "min_duration=2&max_duration=1"} {
		if _, err := parseList(raw, "hunt"); err == nil {
			t.Fatal(raw, "accepted")
		}
	}
	for _, raw := range []string{"holding_status=lost", "source=caught", "has_hunt=yes", "hunt_id=0", "obtained_from=2025&obtained_to=2024", "species_id=1"} {
		if _, err := parseList(raw, "shiny"); err == nil {
			t.Fatal(raw, "accepted")
		}
	}
}
func TestQueryFailureIsNotAnEmptySuccess(t *testing.T) {
	db := testutil.DB(t)
	conn, _ := db.DB()
	conn.Close()
	_, _, err := (Hunts{db}).List(context.Background(), "")
	if err == nil {
		t.Fatal("query failure reported success")
	}
}
