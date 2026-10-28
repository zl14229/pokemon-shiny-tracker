package service

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/zl14229/pokemon-shiny-tracker/internal/backup"
	"github.com/zl14229/pokemon-shiny-tracker/internal/database"
	"github.com/zl14229/pokemon-shiny-tracker/internal/huntdata"
	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
)

var bingoStart = time.Date(2030, 1, 1, 10, 0, 0, 0, time.UTC)
var bingoNow = time.Date(2030, 1, 3, 20, 0, 0, 0, time.UTC)

func bingoFixture(t *testing.T, size int) (Planner, PlannerView) {
	t.Helper()
	s, r := plannerFixture(t)
	s.Now = func() time.Time { return bingoNow }
	if size == 5 {
		// Test-only extra facts exercise a full 5x5. They are not real availability
		// claims and never enter the bundled production dataset or coverage report.
		d := huntdata.Dataset{Version: "synthetic-t028-only", Targets: []huntdata.Target{}}
		for i := 1; i <= 14; i++ {
			d.Targets = append(d.Targets, huntdata.Target{GameVersionID: 40, SpeciesID: i, FormID: i, Availability: "confirmed_available", DirectHunt: "yes", Verification: "verified", GuideStatus: "unknown", Evidence: huntdata.Evidence{Source: "synthetic T028 fixture; not real game fact", URL: "https://example.org/t028", Version: "test-only", Scope: "synthetic game/form mapping only", CheckedAt: bingoStart}})
		}
		if _, e := huntdata.Import(s.DB, d, true, huntdata.Digest(d)); e != nil {
			t.Fatal(e)
		}
	}
	s.Now = func() time.Time { return bingoStart }
	v, e := s.CreateBingo(plannerCtx, BingoCreateInput{RequestID: fmt.Sprintf("bingo-create-%d-001", size), Rules: r, Size: size, CalendarZone: "UTC"})
	if e != nil {
		t.Fatal(e)
	}
	s.Now = func() time.Time { return bingoNow }
	return s, v
}
func bingoRecord(t *testing.T, s Planner, target PlannerTargetView, day *string) model.ShinyRecord {
	t.Helper()
	f := target.Snapshot.Choice.Evidence[0].FormID
	r := model.ShinyRecord{Base: model.Base{CreatedAt: time.Date(2030, 1, 2, 15, 0, 0, 0, time.UTC)}, OriginalSpeciesID: &target.SpeciesID, CurrentSpeciesID: target.SpeciesID, OriginalFormID: &f, CurrentFormID: &f, OriginalGameID: &target.GameID, CurrentGameID: &target.GameID, ObtainedDate: day, Source: "self_hunted", HoldingStatus: "held"}
	if e := s.DB.Omit("Hunt", "OriginalSpecies", "CurrentSpecies", "OriginalForm", "CurrentForm", "OriginalGame", "CurrentGame").Create(&r).Error; e != nil {
		t.Fatal(e)
	}
	return r
}
func TestBingoLinesAllSizesAndBothDiagonals(t *testing.T) {
	for _, n := range []int{3, 5} {
		if len(BingoLines(n, map[int]bool{})) != 0 {
			t.Fatal("free cell")
		}
		all := map[int]bool{}
		for i := 1; i <= n*n; i++ {
			all[i] = true
		}
		if len(BingoLines(n, all)) != 2*n+2 {
			t.Fatal(n)
		}
		diag := map[int]bool{}
		for i := 0; i < n; i++ {
			diag[i*n+i+1] = true
		}
		if len(BingoLines(n, diag)) != 1 {
			t.Fatal("main diagonal")
		}
		diag = map[int]bool{}
		for i := 0; i < n; i++ {
			diag[i*n+n-i] = true
		}
		if len(BingoLines(n, diag)) != 1 {
			t.Fatal("other diagonal")
		}
	}
}
func TestBingoGridPersistenceNoDuplicatesRerollAndStartedBoundary(t *testing.T) {
	s, v := bingoFixture(t, 3)
	if len(v.Targets) != 9 || len(v.Bingo.Lines) != 0 || v.Status != "active" {
		t.Fatal(v)
	}
	seen := map[string]bool{}
	for _, c := range v.Targets {
		if seen[c.TargetKey] || c.Status != "pending" {
			t.Fatal("duplicate or free cell")
		}
		seen[c.TargetKey] = true
	}
	target := v.Targets[8]
	in := BingoRerollInput{RequestID: "bingo-reroll-001", ExpectedTargetKey: target.TargetKey}
	rerolled, e := s.RerollBingo(plannerCtx, v.ID, target.ID, in)
	if e != nil {
		t.Fatal(e)
	}
	if seen[rerolled.Targets[8].TargetKey] {
		t.Fatal("reroll duplicated grid target")
	}
	retry, e := s.RerollBingo(plannerCtx, v.ID, target.ID, in)
	if e != nil || plannerJSON(retry) != plannerJSON(rerolled) {
		t.Fatal("reroll retry changed card", e)
	}
	if _, e = s.StartHunt(plannerCtx, v.ID, target.ID, nil, target.TargetKey); e == nil {
		t.Fatal("stale target started")
	}
	started, e := s.StartHunt(plannerCtx, v.ID, v.Targets[0].ID, nil, v.Targets[0].TargetKey)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.RerollBingo(plannerCtx, v.ID, v.Targets[0].ID, BingoRerollInput{RequestID: "started-reroll-001", ExpectedTargetKey: v.Targets[0].TargetKey}); e == nil {
		t.Fatal("started cell rerolled")
	}
	restored, e := (Planner{DB: s.DB}).Get(plannerCtx, v.ID)
	if e != nil || plannerJSON(restored) != plannerJSON(started) {
		t.Fatal("reload differs", e)
	}
	if len(restored.Bingo.Events) != 3 {
		t.Fatal("creation/reroll/link audit missing", len(restored.Bingo.Events))
	}
}
func TestBingoRejectsInsufficientAndKeepsMultipleCards(t *testing.T) {
	s, r := plannerFixture(t)
	if _, e := s.CreateBingo(plannerCtx, BingoCreateInput{RequestID: "insufficient-001", Rules: r, Size: 5}); e == nil {
		t.Fatal("12 candidates filled 25 cells")
	}
	var n int64
	s.DB.Model(&model.HuntPlan{}).Count(&n)
	if n != 0 {
		t.Fatal("partial card on failure")
	}
	input := BingoCreateInput{RequestID: "many-cards-001", Rules: r, Size: 3}
	a, e := s.CreateBingo(plannerCtx, input)
	if e != nil {
		t.Fatal(e)
	}
	again, e := s.CreateBingo(plannerCtx, input)
	if e != nil || a.ID != again.ID {
		t.Fatal("create duplicated")
	}
	input.RequestID = "many-cards-002"
	b, e := s.CreateBingo(plannerCtx, input)
	if e != nil || b.ID == a.ID {
		t.Fatal(e)
	}
	input.Size = 5
	input.RequestID = "many-cards-001"
	if _, e = s.CreateBingo(plannerCtx, input); e == nil {
		t.Fatal("request reused with new rules")
	}
	input.RequestID = "bad-zone-001"
	input.Size = 3
	input.CalendarZone = "not/a/zone"
	if _, e = s.CreateBingo(plannerCtx, input); e == nil {
		t.Fatal("unknown calendar zone")
	}
}
func TestBingoAutomaticNewPersonalLegalRecordsAndHistoryBoundaries(t *testing.T) {
	s, v := bingoFixture(t, 3)
	day := "2030-01-02"
	first := bingoRecord(t, s, v.Targets[0], &day)
	second := bingoRecord(t, s, v.Targets[1], &day)
	third := bingoRecord(t, s, v.Targets[2], &day)
	// Transfer does not erase an original-game catch; current location is not the
	// origin filter. Losing ownership also does not erase a newly earned outcome.
	s.DB.Model(&first).Updates(map[string]any{"current_game_id": 1, "holding_status": "traded"})
	out, e := s.RefreshBingo(plannerCtx, v.ID)
	if e != nil || len(out.Bingo.Lines) != 1 {
		t.Fatal(out.Bingo, e)
	}
	for i := 0; i < 3; i++ {
		if out.Targets[i].Status != "completed" || out.Targets[i].ShinyRecordID == nil {
			t.Fatal("new record not credited")
		}
	}
	if *out.Targets[1].ShinyRecordID != second.ID || *out.Targets[2].ShinyRecordID != third.ID {
		t.Fatal("identity mismatch")
	}
	if _, e = s.RerollBingo(plannerCtx, v.ID, v.Targets[0].ID, BingoRerollInput{RequestID: "complete-reroll-001", ExpectedTargetKey: v.Targets[0].TargetKey}); e == nil {
		t.Fatal("completed cell rerolled")
	}
	closed, e := s.CloseBingo(plannerCtx, v.ID, "completed")
	if e != nil || closed.Status != "completed" {
		t.Fatal(e)
	}
	after, e := s.RefreshBingo(plannerCtx, v.ID)
	if e != nil || plannerJSON(after) != plannerJSON(closed) {
		t.Fatal("closed history mutated", e)
	}
	if _, e = s.CloseBingo(plannerCtx, v.ID, "abandoned"); e == nil {
		t.Fatal("completed challenge overwritten")
	}
}
func TestBingoOldTradeTransferUnknownDateAndLegalityCannotAutoComplete(t *testing.T) {
	cases := []struct {
		name   string
		change func(*model.ShinyRecord)
		needs  bool
	}{
		{"old identity", func(r *model.ShinyRecord) { r.CreatedAt = bingoStart.Add(-time.Hour) }, false},
		{"old collection", func(r *model.ShinyRecord) { d := "2029-12-31"; r.ObtainedDate = &d }, false},
		{"trade", func(r *model.ShinyRecord) { r.Source = "trade" }, false},
		{"event", func(r *model.ShinyRecord) { r.Source = "event" }, false},
		{"source unknown", func(r *model.ShinyRecord) { r.Source = "unknown" }, true},
		{"transfer only", func(r *model.ShinyRecord) { id := uint64(1); r.OriginalGameID = &id }, false},
		{"missing date", func(r *model.ShinyRecord) { r.ObtainedDate = nil }, true},
		{"month", func(r *model.ShinyRecord) { d := "2030-01"; r.ObtainedDate = &d }, true},
		{"same day", func(r *model.ShinyRecord) { d := "2030-01-01"; r.ObtainedDate = &d }, true},
		{"future", func(r *model.ShinyRecord) { d := "2031-01-01"; r.ObtainedDate = &d }, true},
		{"unknown form", func(r *model.ShinyRecord) { r.OriginalFormID = nil }, true},
		{"unknown original game", func(r *model.ShinyRecord) { r.OriginalGameID = nil }, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, v := bingoFixture(t, 3)
			day := "2030-01-02"
			r := bingoRecord(t, s, v.Targets[0], &day)
			tc.change(&r)
			if e := s.DB.Omit("Hunt", "OriginalSpecies", "CurrentSpecies", "OriginalForm", "CurrentForm", "OriginalGame", "CurrentGame").Save(&r).Error; e != nil {
				t.Fatal(e)
			}
			out, e := s.RefreshBingo(plannerCtx, v.ID)
			if e != nil || out.Targets[0].Status == "completed" {
				t.Fatal(out, e)
			}
			if tc.needs && out.Bingo.Cells[0].Kind != "needs_review" {
				t.Fatal("missing review feedback", out.Bingo.Cells[0])
			}
		})
	}
	s, v := bingoFixture(t, 3)
	day := "2030-01-02"
	bingoRecord(t, s, v.Targets[0], &day)
	d, _ := huntdata.Bundled()
	if e := huntdata.Rollback(s.DB, d.Version, huntdata.Digest(d)); e != nil {
		t.Fatal(e)
	}
	out, e := s.RefreshBingo(plannerCtx, v.ID)
	if e != nil || out.Targets[0].Status == "completed" || out.Bingo.Cells[0].Kind != "needs_review" {
		t.Fatal("retracted source credited", e)
	}
}
func TestBingoExplicitTimestampReviewCannotOverrideSourceOrDate(t *testing.T) {
	s, v := bingoFixture(t, 3)
	day := "2030-01-01"
	r := bingoRecord(t, s, v.Targets[0], &day)
	s.DB.Model(&r).Update("created_at", bingoStart.Add(time.Hour))
	pending, e := s.RefreshBingo(plannerCtx, v.ID)
	if e != nil || pending.Bingo.Cells[0].Kind != "needs_review" {
		t.Fatal(e)
	}
	input := BingoDateInput{RecordID: r.ID, ObtainedAt: bingoStart.Add(30 * time.Minute), EvidenceNote: "synthetic: reviewed capture timestamp", Confirmed: true}
	for _, bad := range []BingoDateInput{{RecordID: r.ID, ObtainedAt: bingoStart.Add(-time.Hour), EvidenceNote: "test", Confirmed: true}, {RecordID: r.ID, ObtainedAt: bingoNow.Add(time.Hour), EvidenceNote: "test", Confirmed: true}, {RecordID: r.ID, ObtainedAt: bingoStart.Add(30 * time.Minute), EvidenceNote: "test", Confirmed: false}} {
		if _, e = s.VerifyBingoDate(plannerCtx, v.ID, v.Targets[0].ID, bad); e == nil {
			t.Fatal("invalid date evidence accepted")
		}
	}
	s.DB.Model(&r).Update("source", "trade")
	if _, e = s.VerifyBingoDate(plannerCtx, v.ID, v.Targets[0].ID, input); e == nil {
		t.Fatal("manual trade override")
	}
	s.DB.Model(&r).Update("source", "self_hunted")
	out, e := s.VerifyBingoDate(plannerCtx, v.ID, v.Targets[0].ID, input)
	if e != nil || out.Targets[0].Status != "completed" || out.Bingo.Cells[0].Proof.Method != "manual_timestamp" {
		t.Fatal(e)
	}
	retry, e := s.VerifyBingoDate(plannerCtx, v.ID, v.Targets[0].ID, input)
	if e != nil || plannerJSON(retry) != plannerJSON(out) {
		t.Fatal("manual verification not idempotent", e)
	}
	input.EvidenceNote = "changed evidence"
	if _, e = s.VerifyBingoDate(plannerCtx, v.ID, v.Targets[0].ID, input); e == nil {
		t.Fatal("completed evidence changed")
	}
}
func TestBingoFiveByFiveAutoCompletionBackupAndAbandonedHistory(t *testing.T) {
	s, v := bingoFixture(t, 5)
	day := "2030-01-02"
	for _, target := range v.Targets {
		bingoRecord(t, s, target, &day)
	}
	complete, e := s.RefreshBingo(plannerCtx, v.ID)
	if e != nil || complete.Status != "completed" || len(complete.Bingo.Lines) != 12 {
		t.Fatal(complete.Status, e)
	}
	var source string
	s.DB.Raw("PRAGMA database_list").Row().Scan(new(int), new(string), &source)
	zip := filepath.Join(t.TempDir(), "bingo.zip")
	if _, e = backup.Backup(plannerCtx, source, zip); e != nil {
		t.Fatal(e)
	}
	dest := filepath.Join(t.TempDir(), "restored")
	if _, e = backup.Restore(plannerCtx, zip, dest); e != nil {
		t.Fatal(e)
	}
	db, e := database.Open(filepath.Join(dest, "database.sqlite3"))
	if e != nil {
		t.Fatal(e)
	}
	conn, _ := db.DB()
	defer conn.Close()
	restored, e := (Planner{DB: db}).Get(plannerCtx, v.ID)
	if e != nil || plannerJSON(restored) != plannerJSON(complete) {
		t.Fatal("backup lost card/proofs/history", e)
	}
	other, card := bingoFixture(t, 3)
	closed, e := other.CloseBingo(plannerCtx, card.ID, "abandoned")
	if e != nil || closed.Status != "abandoned" || len(closed.Targets) != 9 {
		t.Fatal(e)
	}
	if _, e = other.RerollBingo(plannerCtx, card.ID, card.Targets[0].ID, BingoRerollInput{RequestID: "abandoned-reroll-001", ExpectedTargetKey: card.Targets[0].TargetKey}); e == nil {
		t.Fatal("abandoned card rerolled")
	}
	if _, e = other.StartHunt(plannerCtx, card.ID, card.Targets[0].ID, nil, card.Targets[0].TargetKey); e == nil {
		t.Fatal("closed challenge started")
	}
}
func TestBingoRerollRefusesUnsyncedQualifiedOutcome(t *testing.T) {
	s, v := bingoFixture(t, 3)
	day := "2030-01-02"
	bingoRecord(t, s, v.Targets[0], &day)
	_, e := s.RerollBingo(plannerCtx, v.ID, v.Targets[0].ID, BingoRerollInput{RequestID: "fresh-outcome-reroll-001", ExpectedTargetKey: v.Targets[0].TargetKey})
	if e == nil {
		t.Fatal("unrefreshed earned completion silently replaced")
	}
	out, e := s.RefreshBingo(plannerCtx, v.ID)
	if e != nil || out.Targets[0].Status != "completed" {
		t.Fatal(e)
	}
}
func TestBingoSourceCorruptionFailsWithoutPartialCompletion(t *testing.T) {
	s, v := bingoFixture(t, 3)
	day := "2030-01-02"
	bingoRecord(t, s, v.Targets[0], &day)
	s.DB.Model(&model.ShinyAvailability{}).Where("id > 0").Update("source_url", "bad")
	_, e := s.RefreshBingo(plannerCtx, v.ID)
	var p *Error
	if !errors.As(e, &p) || p.Code != "reference_unavailable" {
		t.Fatal(e)
	}
	var target model.HuntPlanTarget
	s.DB.First(&target, v.Targets[0].ID)
	if target.Status != "pending" {
		t.Fatal("partial completion on invalid sources")
	}
}

func TestBingoCalendarDayUsesStoredZone(t *testing.T) {
	s, r := plannerFixture(t)
	start := time.Date(2030, 1, 1, 20, 0, 0, 0, time.UTC)
	s.Now = func() time.Time { return start }
	v, e := s.CreateBingo(plannerCtx, BingoCreateInput{RequestID: "calendar-zone-001", Rules: r, Size: 3, CalendarZone: "Asia/Shanghai"})
	if e != nil {
		t.Fatal(e)
	}
	s.Now = func() time.Time { return bingoNow }
	day := "2030-01-02"
	rec := bingoRecord(t, s, v.Targets[0], &day)
	out, e := s.RefreshBingo(plannerCtx, v.ID)
	if e != nil || out.Targets[0].Status == "completed" || out.Bingo.Cells[0].Reviews[0].Code != "same_day" {
		t.Fatal("UTC date substituted for challenge calendar", e)
	}
	day = "2030-01-03"
	s.DB.Model(&rec).Update("obtained_date", day)
	out, e = s.RefreshBingo(plannerCtx, v.ID)
	if e != nil || out.Targets[0].Status != "completed" {
		t.Fatal(e)
	}
}
func TestBingoRerollWithNoAlternativeIsAtomic(t *testing.T) {
	s, v := bingoFixture(t, 3)
	pool, e := s.Pool(plannerCtx, v.Rules)
	if e != nil {
		t.Fatal(e)
	}
	existing := map[string]bool{}
	for _, t := range v.Targets {
		existing[t.TargetKey] = true
	}
	for _, c := range pool.Candidates {
		if !existing[c.Key] {
			r := model.ShinyRecord{CurrentSpeciesID: c.SpeciesID, Source: "trade", HoldingStatus: "held"}
			if e = s.DB.Create(&r).Error; e != nil {
				t.Fatal(e)
			}
		}
	}
	_, e = s.RerollBingo(plannerCtx, v.ID, v.Targets[0].ID, BingoRerollInput{RequestID: "no-alternative-001", ExpectedTargetKey: v.Targets[0].TargetKey})
	var p *Error
	if !errors.As(e, &p) || p.Code != "insufficient_candidates" {
		t.Fatal(e)
	}
	after, e := s.Get(plannerCtx, v.ID)
	if e != nil || plannerJSON(after) != plannerJSON(v) {
		t.Fatal("failed reroll mutated card", e)
	}
}
