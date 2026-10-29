package service

import (
	"context"
	"math"
	"reflect"
	"testing"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"gorm.io/gorm/clause"
)

func TestStatisticsAdvancedSamplesDistinguishWeightedMeanAndMedian(t *testing.T) {
	f := dexSeed(t)
	for i, values := range [][3]int64{{100, 600, 2}, {300, 1200, 1}} {
		h := statsHunt(t, f, &f.game.ID, "wild", "encounters", &values[0], &values[1], "completed")
		for n := int64(0); n < values[2]; n++ {
			statsRecord(t, f, &h.ID, "self_hunted", ptr("2024-02-02"), &f.game.ID)
		}
		v := getStats(t, f, "")
		g := v.Effort[0]
		if i == 0 {
			if g.CountSampleStatus != "single_sample" || g.MedianCountPerShiny != nil || g.MedianSecondsPerShiny != nil {
				t.Fatal(g)
			}
		} else {
			if g.CountSampleStatus != "descriptive_subset" || g.TimeSampleStatus != "descriptive_subset" || g.CountSamples != 2 || g.CountOutcomes != 3 || math.Abs(*g.CountPerShiny-400.0/3) > 1e-9 || *g.MedianCountPerShiny != 175 || *g.SecondsPerShiny != 600 || *g.MedianSecondsPerShiny != 750 {
				t.Fatal(g)
			}
		}
	}
	v := getStats(t, f, "from=2024-01-01&to=2024-12-31")
	g := v.Effort[0]
	if g.CountPerShiny != nil || g.MedianCountPerShiny != nil || g.CountSampleStatus != "no_eligible_samples" || g.Exclusions["date_filter"] != 2 {
		t.Fatal(g)
	}
}

func TestStatisticsExclusionEvidenceAndMissingMetrics(t *testing.T) {
	for _, kind := range []string{"unfinished_hunt", "unknown_game", "unknown_method", "source_filter", "no_outcomes", "outcome_evidence", "missing_count", "missing_unit", "missing_duration", "zero_duration", "unsafe_count", "unsafe_duration"} {
		t.Run(kind, func(t *testing.T) {
			f := dexSeed(t)
			n, seconds := int64(10), int64(60)
			h := model.Hunt{SpeciesID: f.a.ID, GameID: &f.game.ID, Method: ptr("wild"), CountUnit: ptr("encounters"), EncounterCount: &n, DurationSeconds: &seconds, Status: "completed"}
			query := ""
			switch kind {
			case "unfinished_hunt":
				h.Status = "abandoned"
			case "unknown_game":
				h.GameID = nil
			case "unknown_method":
				h.Method = nil
			case "source_filter":
				query = "source=trade"
			case "missing_count":
				h.EncounterCount = nil
			case "missing_unit":
				h.CountUnit = nil
				h.EncounterCount = nil
			case "missing_duration":
				h.DurationSeconds = nil
			case "zero_duration":
				seconds = 0
			case "unsafe_count":
				n = maxChartInteger + 1
			case "unsafe_duration":
				seconds = maxChartInteger + 1
			}
			if e := f.db.Omit(clause.Associations).Create(&h).Error; e != nil {
				t.Fatal(e)
			}
			if kind != "no_outcomes" {
				source := "self_hunted"
				if kind == "outcome_evidence" {
					source = "trade"
				}
				statsRecord(t, f, &h.ID, source, nil, &f.game.ID)
			}
			g := getStats(t, f, query).Effort[0]
			if g.Exclusions[kind] != 1 {
				t.Fatal(g)
			}
			if kind == "missing_count" || kind == "missing_unit" || kind == "unsafe_count" {
				if g.CountPerShiny != nil || g.SecondsPerShiny == nil {
					t.Fatal(g)
				}
			} else if kind == "missing_duration" || kind == "zero_duration" || kind == "unsafe_duration" {
				if g.SecondsPerShiny != nil || g.CountPerShiny == nil {
					t.Fatal(g)
				}
			} else if g.CountPerShiny != nil || g.SecondsPerShiny != nil {
				t.Fatal(g)
			}
		})
	}
}

func TestStatisticsSafeIntegerAndZeroBoundaries(t *testing.T) {
	f := dexSeed(t)
	zero := int64(0)
	n := maxChartInteger
	h := statsHunt(t, f, &f.game.ID, "wild", "encounters", &n, &zero, "completed")
	statsRecord(t, f, &h.ID, "self_hunted", nil, &f.game.ID)
	g := getStats(t, f, "").Effort[0]
	if g.Count == nil || *g.Count != maxChartInteger || g.Seconds == nil || *g.Seconds != 0 || g.SecondsPerShiny != nil {
		t.Fatal(g)
	}
	h = statsHunt(t, f, &f.game.ID, "wild", "encounters", &n, &zero, "completed")
	statsRecord(t, f, &h.ID, "self_hunted", nil, &f.game.ID)
	g = getStats(t, f, "").Effort[0]
	if !g.Overflow || g.Count != nil || g.Seconds != nil || g.CountPerShiny != nil || g.MedianCountPerShiny == nil || *g.MedianCountPerShiny != float64(n) {
		t.Fatal(g)
	}
	f2 := dexSeed(t)
	h = statsHunt(t, f2, &f2.game.ID, "wild", "encounters", &zero, &zero, "completed")
	statsRecord(t, f2, &h.ID, "self_hunted", nil, &f2.game.ID)
	g = getStats(t, f2, "").Effort[0]
	if g.CountPerShiny == nil || *g.CountPerShiny != 0 || g.SecondsPerShiny != nil {
		t.Fatal(g)
	}
}

func TestStatisticsCandidatesNeverDeduplicateOrWritePersonalHistory(t *testing.T) {
	f := dexSeed(t)
	r1 := statsRecord(t, f, nil, "self_hunted", ptr("2024-02-02"), &f.game.ID)
	r2 := statsRecord(t, f, nil, "self_hunted", ptr("2024-02-02"), &f.game.ID)
	v := getStats(t, f, "")
	if v.Individuals != 2 || v.Quality.DistinctRecords != 2 || v.Quality.CandidateGroups != 1 || v.Quality.CandidateRecords != 2 || !reflect.DeepEqual(v.Quality.Candidates[0].RecordIDs, []uint64{r1.ID, r2.ID}) {
		t.Fatal(v)
	}
	var historyBefore, historyAfter int64
	f.db.Model(&model.ShinyHistory{}).Count(&historyBefore)
	w := getStats(t, f, "")
	f.db.Model(&model.ShinyHistory{}).Count(&historyAfter)
	if historyBefore != historyAfter || !reflect.DeepEqual(v, w) {
		t.Fatal("statistics mutated personal data")
	}
	r1 = updateDexRecord(t, f, r1, func(in *ShinyInput) {
		in.CurrentSpeciesID = f.b.ID
		in.CurrentFormID = &f.fb.ID
		in.CurrentGameID = &f.other.ID
		in.ChangeType = "evolution"
	})
	updateDexRecord(t, f, r1, func(in *ShinyInput) { in.HoldingStatus = "released" })
	v = getStats(t, f, "game_id="+fmtUint(f.game.ID))
	if v.Individuals != 2 || v.Quality.DistinctRecords != 2 || v.Quality.CandidateGroups != 0 {
		t.Fatal(v)
	}
	if e := (Shinies{DB: f.db}).Delete(context.Background(), r2.ID); e != nil {
		t.Fatal(e)
	}
	v = getStats(t, f, "")
	if v.Individuals != 1 || v.Quality.CandidateGroups != 0 {
		t.Fatal(v)
	}
}

func TestStatisticsDateFilterRegressionUsesExistingOverlap(t *testing.T) {
	f := dexSeed(t)
	for _, d := range []*string{ptr("2024-02-29"), ptr("2024-02"), ptr("2024"), ptr("2023-02-28"), nil} {
		statsRecord(t, f, nil, "self_hunted", d, &f.game.ID)
	}
	for _, tc := range []struct {
		q           string
		n, unplaced int
	}{{"from=2024-02-29&to=2024-02-29", 3, 2}, {"from=2024-02-01&to=2024-02-29", 3, 1}, {"from=0001-01-01&to=9999-12-31", 4, 1}, {"from=2025-01-01&to=2025-12-31", 0, 0}} {
		v := getStats(t, f, tc.q)
		if v.Individuals != tc.n || v.Quality.DistinctRecords != tc.n || v.Coverage.UnplacedRecords != tc.unplaced {
			t.Fatal(tc, v)
		}
		rows, meta, e := (Shinies{DB: f.db}).List(context.Background(), "obtained_from="+tc.q[5:15]+"&obtained_to="+tc.q[len(tc.q)-10:])
		if e != nil || meta.Total != int64(tc.n) || len(rows) != tc.n {
			t.Fatal("existing query diverged", tc, meta, e)
		}
	}
}

func TestStatisticsCandidateCapsAndRepeatedIdentity(t *testing.T) {
	records := []model.ShinyRecord{}
	for i := uint64(1); i <= 110; i++ {
		records = append(records, model.ShinyRecord{Base: model.Base{ID: i}, CurrentSpeciesID: 1})
	}
	records = append(records, records[0])
	v := statisticsQuality(records, 2)
	if v.DistinctRecords != 110 || v.CandidateRecords != 110 || v.CandidateGroups != 1 || !v.CandidatesTruncated || len(v.Candidates[0].RecordIDs) != 100 || v.CountComparison != "separate_contexts" {
		t.Fatal(v)
	}
	records = nil
	for i := uint64(1); i <= 44; i++ {
		records = append(records, model.ShinyRecord{Base: model.Base{ID: i}, CurrentSpeciesID: (i + 1) / 2})
	}
	v = statisticsQuality(records, 1)
	if v.CandidateGroups != 22 || len(v.Candidates) != 20 || !v.CandidatesTruncated || v.CandidateRecords != 44 {
		t.Fatal(v)
	}
}
