package service

import (
	"context"
	"testing"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"gorm.io/gorm/clause"
)

func statsHunt(t *testing.T, f dexFixture, game *uint64, method, unit string, count, seconds *int64, status string) model.Hunt {
	t.Helper()
	h := model.Hunt{SpeciesID: f.a.ID, GameID: game, Method: &method, CountUnit: &unit, EncounterCount: count, DurationSeconds: seconds, Status: status, StartDate: ptr("2024-02-01")}
	if e := f.db.Omit(clause.Associations).Create(&h).Error; e != nil {
		t.Fatal(e)
	}
	return h
}
func statsRecord(t *testing.T, f dexFixture, h *uint64, source string, date *string, game *uint64) model.ShinyRecord {
	t.Helper()
	r, e := (Shinies{DB: f.db}).Create(context.Background(), ShinyInput{HuntID: h, OriginalSpeciesID: &f.a.ID, OriginalFormID: &f.fa.ID, CurrentSpeciesID: f.a.ID, CurrentFormID: &f.fa.ID, OriginalGameID: game, CurrentGameID: game, Source: source, HoldingStatus: "held", ObtainedDate: date})
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func getStats(t *testing.T, f dexFixture, q string) StatisticsResult {
	t.Helper()
	v, e := (Analytics{DB: f.db}).Statistics(context.Background(), q)
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func TestStatisticsGroupsAndEfficiencyUseSeparateCohorts(t *testing.T) {
	f := dexSeed(t)
	n, seconds := int64(100), int64(3600)
	h := statsHunt(t, f, &f.game.ID, "wild", "encounters", &n, &seconds, "completed")
	statsRecord(t, f, &h.ID, "self_hunted", ptr("2024-02-02"), &f.game.ID)
	statsRecord(t, f, &h.ID, "self_hunted", ptr("2024-02"), &f.game.ID)
	h2 := statsHunt(t, f, &f.game.ID, "wild", "eggs", &n, nil, "in_progress")
	statsRecord(t, f, &h2.ID, "self_hunted", nil, &f.game.ID)
	statsRecord(t, f, nil, "trade", ptr("2024"), &f.game.ID)
	statsRecord(t, f, nil, "self_hunted", ptr("2024-02-03"), &f.sleep.ID)
	v := getStats(t, f, "")
	if v.Hunts != 2 || v.Individuals != 4 || len(v.Effort) != 2 || v.Coverage.UnplacedRecords != 2 || len(v.Trend) != 1 || v.Trend[0].Individuals != 2 {
		t.Fatalf("%+v", v)
	}
	for _, e := range v.Effort {
		if *e.Unit == "encounters" {
			if e.CountPerShiny == nil || *e.CountPerShiny != 50 || e.SecondsPerShiny == nil || *e.SecondsPerShiny != 1800 || e.CountSamples != 1 || e.CountOutcomes != 2 {
				t.Fatal(e)
			}
		} else if e.CountPerShiny != nil || e.Seconds != nil || e.KnownDurationHunts != 0 {
			t.Fatal(e)
		}
	}
	w := getStats(t, f, "count_unit=encounters&min_count=100&max_count=100&min_duration=3600&max_duration=3600")
	if w.Hunts != 1 || w.Individuals != 2 {
		t.Fatal(w)
	}
	w = getStats(t, f, "source=trade")
	if w.Hunts != 2 || w.Individuals != 1 {
		t.Fatal(w)
	}
	for _, e := range w.Effort {
		if e.CountPerShiny != nil || e.SecondsPerShiny != nil {
			t.Fatal(e)
		}
	}
}
func TestStatisticsDatePrecisionAndOriginalGameFilters(t *testing.T) {
	f := dexSeed(t)
	n := int64(10)
	h := statsHunt(t, f, &f.game.ID, "wild", "encounters", &n, &n, "completed")
	r := statsRecord(t, f, &h.ID, "self_hunted", ptr("2024-02-29"), &f.game.ID)
	updateDexRecord(t, f, r, func(in *ShinyInput) {
		in.CurrentGameID = &f.other.ID
		in.CurrentSpeciesID = f.b.ID
		in.CurrentFormID = &f.fb.ID
		in.ChangeType = "evolution"
	})
	statsRecord(t, f, nil, "self_hunted", ptr("2024-02"), &f.game.ID)
	statsRecord(t, f, nil, "trade", nil, nil)
	v := getStats(t, f, "game_id="+fmtUint(f.game.ID)+"&from=2024-02-29&to=2024-02-29")
	if v.Individuals != 2 || v.Hunts != 0 || v.Coverage.UnplacedRecords != 1 || v.Trend[0].Individuals != 1 {
		t.Fatal(v)
	}
	if w := getStats(t, f, "game_id="+fmtUint(f.other.ID)); w.Individuals != 0 {
		t.Fatal("transfer became acquisition", w)
	}
	v = getStats(t, f, "from=2024-01-01&to=2024-12-31")
	if v.Effort[0].CountPerShiny != nil || v.Effort[0].SecondsPerShiny != nil {
		t.Fatal("cumulative effort became subperiod efficiency")
	}
}
func TestStatisticsRejectsInvalidAndKeepsUnknown(t *testing.T) {
	f := dexSeed(t)
	for _, q := range []string{"min_count=1", "from=2024", "from=2024-02-30", "from=2025-01-01&to=2024-01-01", "source=bad", "game_id=999999", "game_id=" + fmtUint(f.sleep.ID), "method=a&method=b", "unknown=x", "min_duration=-1", "count_unit=eggs&min_count=2&max_count=1"} {
		if _, e := (Analytics{DB: f.db}).Statistics(context.Background(), q); e == nil {
			t.Fatal(q)
		}
	}
	h := statsHunt(t, f, nil, "wild", "encounters", nil, nil, "in_progress")
	statsRecord(t, f, &h.ID, "unknown", nil, nil)
	v := getStats(t, f, "")
	if v.Effort[0].Count != nil || v.Effort[0].Seconds != nil || v.Effort[0].CountPerShiny != nil || v.Coverage.UnknownObtainedDate != 1 {
		t.Fatal(v)
	}
}

func TestStatisticsLinkedFiltersCannotCrossCollectionSystems(t *testing.T) {
	f := dexSeed(t)
	n := int64(5)
	h := statsHunt(t, f, &f.sleep.ID, "sleep-only", "encounters", &n, &n, "in_progress")
	statsRecord(t, f, &h.ID, "unknown", nil, &f.game.ID)
	v := getStats(t, f, "method=sleep-only")
	if v.Hunts != 0 || v.Individuals != 0 {
		t.Fatal("linked filters crossed systems", v)
	}
}
