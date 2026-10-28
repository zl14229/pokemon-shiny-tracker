package service

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/zl14229/pokemon-shiny-tracker/internal/backup"
	"github.com/zl14229/pokemon-shiny-tracker/internal/database"
	"github.com/zl14229/pokemon-shiny-tracker/internal/huntdata"
	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"github.com/zl14229/pokemon-shiny-tracker/internal/reference"
	"github.com/zl14229/pokemon-shiny-tracker/internal/testutil"
)

var plannerCtx = context.Background()

func plannerFixture(t *testing.T) (Planner, PlannerRules) {
	t.Helper()
	db := testutil.DB(t)
	if e := reference.EnsureSeed(db); e != nil {
		t.Fatal(e)
	}
	if e := reference.EnsureCatalog(db); e != nil {
		t.Fatal(e)
	}
	if e := huntdata.EnsureBundled(db); e != nil {
		t.Fatal(e)
	}
	var refs []model.ExternalReference
	if e := db.Where("resource = 'game' AND external_id IN ?", []int{40, 41}).Order("external_id").Find(&refs).Error; e != nil || len(refs) != 2 {
		t.Fatal(refs, e)
	}
	return Planner{DB: db, RandomIndex: func(int) (int, error) { return 0, nil }}, PlannerRules{GameIDs: []uint64{*refs[0].GameID, *refs[1].GameID}, Count: 1}
}
func TestPlannerUniformTargetsBeforeVersionsAndNoReplacement(t *testing.T) {
	// One target has ten versions; the other targets have one. Target weights
	// must stay equal, independently of the selected game's multiplicity.
	pool := PlannerPool{Coverage: PlannerCoverage{Actual: 1}, Candidates: []PlannerCandidate{{Key: "a"}, {Key: "b", Games: []PlannerChoice{{GameID: 1}}}, {Key: "c", Games: []PlannerChoice{{GameID: 1}}}}}
	for i := 1; i <= 10; i++ {
		pool.Candidates[0].Games = append(pool.Candidates[0].Games, PlannerChoice{GameID: uint64(i)})
	}
	rng := rand.New(rand.NewSource(20261029))
	s := Planner{RandomIndex: func(n int) (int, error) { return rng.Intn(n), nil }}
	counts := map[string]int{}
	versions := map[uint64]int{}
	for i := 0; i < 30000; i++ {
		p, e := s.pick(pool)
		if e != nil {
			t.Fatal(e)
		}
		counts[p[0].Key]++
		if p[0].Key == "a" {
			versions[p[0].Choice.GameID]++
		}
	}
	for _, key := range []string{"a", "b", "c"} {
		if counts[key] < 9600 || counts[key] > 10400 {
			t.Fatal("version-biased target sampling", counts)
		}
	}
	for _, count := range versions {
		if count < 850 || count > 1150 {
			t.Fatal("biased game choice", versions)
		}
	}
	pool.Coverage.Actual = 3
	for i := 0; i < 50; i++ {
		p, e := s.pick(pool)
		if e != nil {
			t.Fatal(e)
		}
		seen := map[string]bool{}
		for _, v := range p {
			if seen[v.Key] {
				t.Fatal("duplicate target")
			}
			seen[v.Key] = true
		}
	}
}
func TestPlannerRealCoverageHistoryOriginalGameAndActiveExclusion(t *testing.T) {
	s, r := plannerFixture(t)
	pool, e := s.Pool(plannerCtx, r)
	if e != nil || len(pool.Candidates) != 12 || pool.Coverage.Games[0].VerifiedDirect != 12 {
		t.Fatal(pool.Coverage, e)
	}
	c := pool.Candidates[0]
	origin := uint64(1)
	current := r.GameIDs[0]
	rec := model.ShinyRecord{OriginalSpeciesID: &c.SpeciesID, CurrentSpeciesID: c.SpeciesID, OriginalGameID: &origin, CurrentGameID: &current, Source: "trade", HoldingStatus: "traded"}
	if e = s.DB.Create(&rec).Error; e != nil {
		t.Fatal(e)
	}
	after, e := s.Pool(plannerCtx, r)
	if e != nil || after.Coverage.ExcludedObtained != 1 || len(after.Candidates) != 11 {
		t.Fatal(after.Coverage, e)
	}
	r.Pool = "selected_missing"
	after, e = s.Pool(plannerCtx, r)
	if e != nil || len(after.Candidates) != 12 {
		t.Fatal("transfer credited as obtained in selected game", after.Coverage, e)
	}
	r.Dimension = "form"
	after, e = s.Pool(plannerCtx, r)
	if e != nil || len(after.Candidates) != 12 {
		t.Fatal("unknown original form inferred", e)
	}
	h := model.Hunt{SpeciesID: c.SpeciesID, GameID: &origin, Status: "in_progress"}
	if e = s.DB.Create(&h).Error; e != nil {
		t.Fatal(e)
	}
	after, e = s.Pool(plannerCtx, r)
	if e != nil || after.Coverage.ExcludedActive != 1 {
		t.Fatal("active target in another game not excluded", after.Coverage, e)
	}
	no := false
	r.ExcludeActive = &no
	after, e = s.Pool(plannerCtx, r)
	if e != nil || len(after.Candidates) != 12 {
		t.Fatal(after.Coverage, e)
	}
	// A registered snapshot is not a historical evolution proof.
	ghost := model.ShinyRecord{Base: rec.Base, CurrentSpeciesID: pool.Candidates[1].SpeciesID}
	history := model.ShinyHistory{ShinyRecordID: rec.ID, Kind: "registered", BeforeJSON: "{}", AfterJSON: plannerJSON(ghost)}
	if e = s.DB.Create(&history).Error; e != nil {
		t.Fatal(e)
	}
	r.Dimension = "species"
	r.Pool = "all_missing"
	after, e = s.Pool(plannerCtx, r)
	if e != nil || after.Coverage.ExcludedObtained != 1 {
		t.Fatal(after.Coverage, e)
	}
	history.Kind = "evolution"
	if e = s.DB.Model(&history).Update("kind", "evolution").Error; e != nil {
		t.Fatal(e)
	}
	after, e = s.Pool(plannerCtx, r)
	if e != nil || after.Coverage.ExcludedObtained != 2 {
		t.Fatal("recorded evolution omitted", after.Coverage, e)
	}
}
func TestPlannerSpeciesFormAndVersionDedup(t *testing.T) {
	s, r := plannerFixture(t)
	d, _ := huntdata.Bundled()
	var target huntdata.Target
	for _, v := range d.Targets {
		if v.SpeciesID == 194 {
			target = v
			break
		}
	}
	target.FormID = 194
	target.Evidence.Source = "synthetic test only"
	d = huntdata.Dataset{Version: "synthetic-additional-form", Targets: []huntdata.Target{target}}
	if _, e := huntdata.Import(s.DB, d, true, huntdata.Digest(d)); e != nil {
		t.Fatal(e)
	}
	p, e := s.Pool(plannerCtx, r)
	if e != nil || len(p.Candidates) != 12 {
		t.Fatal(p.Coverage, e)
	}
	for _, c := range p.Candidates {
		if c.SpeciesID == func() uint64 { var f model.Form; s.DB.Where("slug = ?", "wooper").First(&f); return f.SpeciesID }() {
			if len(c.Games) != 2 {
				t.Fatal("game duplicated by forms", c)
			}
			if len(c.Games[0].Evidence) != 2 {
				t.Fatal("form evidence lost")
			}
		}
	}
	r.Dimension = "form"
	p, e = s.Pool(plannerCtx, r)
	if e != nil || len(p.Candidates) != 13 {
		t.Fatal(p.Coverage, e)
	}
}
func TestPlannerShortageEmptyIdempotencePersistenceAndHuntConversion(t *testing.T) {
	s, r := plannerFixture(t)
	r.Count = 100
	v, e := s.Draw(plannerCtx, PlannerDrawInput{RequestID: "draw-shortage-001", Rules: r})
	if e != nil || len(v.Targets) != 12 || !v.Coverage.Shortage || v.Status != "draft" {
		t.Fatal(v.Coverage, e)
	}
	list, n, e := s.List(plannerCtx, 100, 0)
	if e != nil || n != 0 || len(list) != 0 {
		t.Fatal("draft exposed as saved", e)
	}
	repeat, e := s.Draw(plannerCtx, PlannerDrawInput{RequestID: "draw-shortage-001", Rules: r})
	if e != nil || repeat.ID != v.ID {
		t.Fatal("draw retry duplicated", e)
	}
	saved, e := s.Save(plannerCtx, v.ID)
	if e != nil || saved.Status != "active" {
		t.Fatal(e)
	}
	saved, e = s.StartHunt(plannerCtx, v.ID, v.Targets[0].ID, nil)
	if e != nil || saved.Targets[0].HuntID == nil {
		t.Fatal(e)
	}
	again, e := s.StartHunt(plannerCtx, v.ID, v.Targets[0].ID, nil)
	if e != nil || *again.Targets[0].HuntID != *saved.Targets[0].HuntID {
		t.Fatal("conversion duplicated", e)
	}
	var count int64
	s.DB.Model(&model.Hunt{}).Count(&count)
	if count != 1 {
		t.Fatal(count)
	}
	restored, e := (Planner{DB: s.DB}).Get(plannerCtx, v.ID)
	if e != nil || plannerJSON(restored) != plannerJSON(saved) {
		t.Fatal("reload changed snapshot", e)
	}
	r.GameIDs = []uint64{1}
	empty, e := s.Draw(plannerCtx, PlannerDrawInput{RequestID: "empty-test-001", Rules: r})
	if e != nil || len(empty.Targets) != 0 || !empty.Coverage.Shortage {
		t.Fatal(empty, e)
	}
	if _, e = s.Save(plannerCtx, empty.ID); e == nil {
		t.Fatal("empty result saved")
	}
	var source string
	s.DB.Raw("PRAGMA database_list").Row().Scan(new(int), new(string), &source)
	zip := filepath.Join(t.TempDir(), "plan.zip")
	if _, e = backup.Backup(plannerCtx, source, zip); e != nil {
		t.Fatal(e)
	}
	dir := filepath.Join(t.TempDir(), "restore")
	if _, e = backup.Restore(plannerCtx, zip, dir); e != nil {
		t.Fatal(e)
	}
	db, e := database.Open(filepath.Join(dir, "database.sqlite3"))
	if e != nil {
		t.Fatal(e)
	}
	conn, _ := db.DB()
	defer conn.Close()
	copy, e := (Planner{DB: db}).Get(plannerCtx, v.ID)
	if e != nil || plannerJSON(copy) != plannerJSON(saved) {
		t.Fatal("restored plan differs", e)
	}
}
func TestPlannerFailsClosedAndDoesNotAlterCollection(t *testing.T) {
	s, r := plannerFixture(t)
	v, e := s.Draw(plannerCtx, PlannerDrawInput{RequestID: "fresh-evidence-001", Rules: r})
	if e != nil {
		t.Fatal(e)
	}
	s.Save(plannerCtx, v.ID)
	d, _ := huntdata.Bundled()
	if e = huntdata.Rollback(s.DB, d.Version, huntdata.Digest(d)); e != nil {
		t.Fatal(e)
	}
	if _, e = s.StartHunt(plannerCtx, v.ID, v.Targets[0].ID, nil); e == nil {
		t.Fatal("retracted target converted")
	}
	pool, e := s.Pool(plannerCtx, r)
	if e != nil || pool.Coverage.Available != 0 {
		t.Fatal(pool, e)
	}
	var n int64
	s.DB.Model(&model.Hunt{}).Count(&n)
	if n != 0 {
		t.Fatal("failed conversion created Hunt")
	}
	s.DB.Model(&model.ShinyRecord{}).Count(&n)
	if n != 0 {
		t.Fatal("planner created collection")
	}
}
func TestPlannerMissingSourceReturnsActionableFailure(t *testing.T) {
	s, r := plannerFixture(t)
	if e := s.DB.Model(&model.ShinyAvailability{}).Where("id > 0").Update("source_url", "broken").Error; e != nil {
		t.Fatal(e)
	}
	_, e := s.Pool(plannerCtx, r)
	var p *Error
	if !errors.As(e, &p) || p.Code != "reference_unavailable" {
		t.Fatal(e)
	}
}
func TestPlannerRejectsBadRulesAndRandomRollback(t *testing.T) {
	s, r := plannerFixture(t)
	for _, bad := range []PlannerRules{{GameIDs: r.GameIDs, Count: 0}, {GameIDs: []uint64{1, 1}, Count: 1}, {GameIDs: []uint64{999999}, Count: 1}, {GameIDs: r.GameIDs, Count: 1, Pool: "current"}} {
		if _, e := s.Pool(plannerCtx, bad); e == nil {
			t.Fatal(bad)
		}
	}
	s.RandomIndex = func(int) (int, error) { return 0, errors.New("rng unavailable") }
	if _, e := s.Draw(plannerCtx, PlannerDrawInput{RequestID: "failed-rng-001", Rules: r}); e == nil {
		t.Fatal("rng failure ignored")
	}
	var n int64
	s.DB.Model(&model.HuntPlan{}).Count(&n)
	if n != 0 {
		t.Fatal("partial plan")
	}
	// Explicit storage/system boundaries cannot be bypassed by arbitrary IDs.
	storage := model.Game{Labels: model.Labels{Slug: "storage-test"}, Kind: "storage", CollectionSystem: "pokemon"}
	s.DB.Create(&storage)
	if _, e := s.Pool(plannerCtx, PlannerRules{GameIDs: []uint64{storage.ID}, Count: 1}); e == nil {
		t.Fatal("storage selected")
	}
	sleep := model.Game{Labels: model.Labels{Slug: "sleep-test"}, CollectionSystem: "sleep"}
	s.DB.Create(&sleep)
	if _, e := s.Pool(plannerCtx, PlannerRules{GameIDs: []uint64{sleep.ID, r.GameIDs[0]}, Count: 1}); e == nil {
		t.Fatal("systems mixed")
	}
}

// Keep the typed snapshot contract checked independently of its SQL storage.
func TestPlannerSnapshotUsesStructuredFacts(t *testing.T) {
	s, r := plannerFixture(t)
	v, e := s.Draw(plannerCtx, PlannerDrawInput{RequestID: "snapshot-001", Rules: r})
	if e != nil {
		t.Fatal(e)
	}
	b, _ := json.Marshal(v)
	var raw map[string]any
	if e = json.Unmarshal(b, &raw); e != nil {
		t.Fatal(e)
	}
	if _, ok := raw["rules_json"]; ok {
		t.Fatal("raw storage field exposed")
	}
	if len(v.Targets[0].Snapshot.Choice.Evidence) == 0 {
		t.Fatal("source snapshot missing")
	}
}

func TestPlannerNeverSamplesLockedTransferUnknownOrConflict(t *testing.T) {
	s, r := plannerFixture(t)
	d, _ := huntdata.Bundled()
	if e := huntdata.Rollback(s.DB, d.Version, huntdata.Digest(d)); e != nil {
		t.Fatal(e)
	}
	d.Version = "synthetic-statuses-only"
	d.Targets[0].Availability = "confirmed_locked"
	d.Targets[0].DirectHunt = "no"
	d.Targets[1].Availability = "transfer_only"
	d.Targets[1].DirectHunt = "no"
	d.Targets[2].Availability = "unknown"
	d.Targets[2].Verification = "unknown"
	d.Targets[2].DirectHunt = "unknown"
	d.Targets[2].Evidence.Source = ""
	d.Targets[3].Verification = "conflict"
	d.Targets[3].DirectHunt = "unknown"
	d.Targets[4].GuideStatus = "unknown"
	d.Targets[4].Methods = nil
	d.Targets[5].Methods[0].ShinyRate = nil
	if _, e := huntdata.Import(s.DB, d, true, huntdata.Digest(d)); e != nil {
		t.Fatal(e)
	}
	r.GameIDs = r.GameIDs[:1]
	r.Count = 100
	v, e := s.Draw(plannerCtx, PlannerDrawInput{RequestID: "safety-statuses-001", Rules: r})
	if e != nil || len(v.Targets) != 8 {
		t.Fatal(v.Coverage, e)
	}
	missingGuide, unknownOdds := false, false
	for _, v := range v.Targets {
		for _, row := range v.Snapshot.Choice.Evidence {
			if !row.Fact.Huntable() {
				t.Fatal("illegal target sampled", row)
			}
			if len(row.Fact.Methods) == 0 {
				missingGuide = true
			}
			for _, m := range row.Fact.Methods {
				if m.ShinyRate == nil {
					unknownOdds = true
				}
			}
		}
	}
	if !missingGuide || !unknownOdds {
		t.Fatal("valid target was lost or unknown odds invented")
	}
}
func TestPlannerLinksExistingHuntAndRejectsStaleMissingPool(t *testing.T) {
	s, r := plannerFixture(t)
	v, e := s.Draw(plannerCtx, PlannerDrawInput{RequestID: "existing-link-001", Rules: r})
	if e != nil {
		t.Fatal(e)
	}
	s.Save(plannerCtx, v.ID)
	target := v.Targets[0]
	h := model.Hunt{SpeciesID: target.SpeciesID, GameID: &target.GameID, Status: "in_progress"}
	if e = s.DB.Create(&h).Error; e != nil {
		t.Fatal(e)
	}
	linked, e := s.StartHunt(plannerCtx, v.ID, target.ID, &h.ID)
	if e != nil || linked.Targets[0].HuntID == nil || *linked.Targets[0].HuntID != h.ID {
		t.Fatal(e)
	}
	var n int64
	s.DB.Model(&model.Hunt{}).Count(&n)
	if n != 1 {
		t.Fatal("link created duplicate")
	}
	second, e := s.Draw(plannerCtx, PlannerDrawInput{RequestID: "stale-missing-001", Rules: r})
	if e != nil {
		t.Fatal(e)
	}
	s.Save(plannerCtx, second.ID)
	target = second.Targets[0]
	rec := model.ShinyRecord{CurrentSpeciesID: target.SpeciesID, OriginalSpeciesID: &target.SpeciesID, Source: "trade", HoldingStatus: "held"}
	s.DB.Create(&rec)
	if _, e = s.StartHunt(plannerCtx, second.ID, target.ID, nil); e == nil {
		t.Fatal("newly obtained target ignored")
	}
	s.DB.Model(&model.Hunt{}).Count(&n)
	if n != 1 {
		t.Fatal("stale conversion wrote Hunt")
	}
}
