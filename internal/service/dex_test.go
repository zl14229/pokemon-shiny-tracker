package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"github.com/zl14229/pokemon-shiny-tracker/internal/testutil"
	"gorm.io/gorm"
)

type dexFixture struct {
	db                             *gorm.DB
	a, b, c, d                     model.Species
	fa, regional, fb, fc, fd, mega model.Form
	game, other, sleep, home       model.Game
}

func dexSeed(t *testing.T) dexFixture {
	t.Helper()
	f := dexFixture{db: testutil.DB(t)}
	for i, p := range []*model.Species{&f.a, &f.b, &f.c, &f.d} {
		*p = model.Species{Labels: model.Labels{Slug: []string{"a", "b", "c", "d"}[i], Name: []string{"A", "B", "C", "D"}[i]}, NationalDex: i + 1}
		if e := f.db.Create(p).Error; e != nil {
			t.Fatal(e)
		}
	}
	for i, p := range []*model.Form{&f.fa, &f.regional, &f.fb, &f.fc, &f.fd, &f.mega} {
		*p = model.Form{Labels: model.Labels{Slug: []string{"a-default", "a-regional", "b-default", "c-default", "d-default", "d-mega"}[i], Name: "Form"}, SpeciesID: []uint64{f.a.ID, f.a.ID, f.b.ID, f.c.ID, f.d.ID, f.d.ID}[i]}
		if e := f.db.Create(p).Error; e != nil {
			t.Fatal(e)
		}
		inclusion := "included"
		if i == 5 {
			inclusion = "excluded"
		}
		entry := model.CatalogEntry{Kind: "form", ExternalID: i + 1, SpeciesID: p.SpeciesID, FormID: &p.ID, NamesJSON: `{"en":"Fixture Form"}`, NameStatus: "unknown", Inclusion: inclusion, Reason: "synthetic-test", SourceURL: "https://example.test/form", CrossCheckStatus: "unknown", SourceRevision: "test-v1", SourceSHA256: "test", FetchedAt: time.Now(), CheckedAt: time.Now()}
		if e := f.db.Create(&entry).Error; e != nil {
			t.Fatal(e)
		}
	}
	for i, p := range []*model.Game{&f.game, &f.other, &f.sleep, &f.home} {
		*p = model.Game{Labels: model.Labels{Slug: []string{"game-a", "game-b", "sleep", "home"}[i], Name: "Fixture Game"}, Kind: []string{"mainline", "mainline", "spin_off", "storage"}[i], CollectionSystem: []string{"pokemon", "pokemon", "sleep", "pokemon"}[i]}
		if e := f.db.Create(p).Error; e != nil {
			t.Fatal(e)
		}
	}
	addAvailability(t, f.db, f.game, f.fa, "confirmed_available", "verified")
	addAvailability(t, f.db, f.game, f.regional, "confirmed_available", "verified")
	addAvailability(t, f.db, f.game, f.fb, "transfer_only", "verified")
	addAvailability(t, f.db, f.game, f.fc, "confirmed_available", "conflict")
	addAvailability(t, f.db, f.game, f.fd, "confirmed_locked", "verified")
	addAvailability(t, f.db, f.game, f.mega, "confirmed_available", "verified")
	addAvailability(t, f.db, f.other, f.fc, "confirmed_available", "verified")
	addAvailability(t, f.db, f.other, f.fb, "confirmed_available", "unknown")
	addAvailability(t, f.db, f.home, f.fd, "confirmed_available", "verified")
	return f
}
func addAvailability(t *testing.T, db *gorm.DB, g model.Game, f model.Form, status, verification string) {
	t.Helper()
	now := time.Now().UTC()
	a := model.ShinyAvailability{GameID: g.ID, SpeciesID: f.SpeciesID, FormID: f.ID, Status: status, VerificationStatus: verification, Source: "synthetic-fixture-only", SourceURL: "https://example.test/availability", CheckedAt: &now, DataVersion: "test-v1"}
	if e := db.Create(&a).Error; e != nil {
		t.Fatal(e)
	}
}
func createDexRecord(t *testing.T, f dexFixture, s model.Species, form *uint64, game *uint64, holding string) model.ShinyRecord {
	t.Helper()
	r, e := (Shinies{DB: f.db}).Create(context.Background(), ShinyInput{OriginalSpeciesID: &s.ID, OriginalFormID: form, CurrentSpeciesID: s.ID, CurrentFormID: form, OriginalGameID: game, CurrentGameID: game, HoldingStatus: holding})
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func updateDexRecord(t *testing.T, f dexFixture, r model.ShinyRecord, change func(*ShinyInput)) model.ShinyRecord {
	t.Helper()
	raw, _ := json.Marshal(r)
	var in ShinyInput
	json.Unmarshal(raw, &in)
	change(&in)
	got, e := (Shinies{DB: f.db}).Update(context.Background(), r.ID, in)
	if e != nil {
		t.Fatal(e)
	}
	return got
}
func getDex(t *testing.T, f dexFixture, q string) DexResult {
	t.Helper()
	v, e := (Dex{DB: f.db}).List(context.Background(), q)
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func target(t *testing.T, v DexResult, id uint64) DexTarget {
	t.Helper()
	for _, row := range v.Data {
		if row.ID == id {
			return row
		}
	}
	t.Fatalf("target %d absent", id)
	return DexTarget{}
}
func idQuery(v uint64) string { return fmtUint(v) }
func fmtUint(v uint64) string { raw, _ := json.Marshal(v); return string(raw) }

func TestDexEvolutionTransferAndDoubleDedup(t *testing.T) {
	f := dexSeed(t)
	r := createDexRecord(t, f, f.a, &f.fa.ID, &f.game.ID, "held")
	r = updateDexRecord(t, f, r, func(in *ShinyInput) {
		in.CurrentSpeciesID = f.b.ID
		in.CurrentFormID = &f.fb.ID
		in.ChangeType = "evolution"
	})
	r = updateDexRecord(t, f, r, func(in *ShinyInput) { in.CurrentGameID = &f.other.ID })
	createDexRecord(t, f, f.a, &f.regional.ID, &f.game.ID, "held")
	createDexRecord(t, f, f.a, &f.fa.ID, &f.game.ID, "released")
	v := getDex(t, f, "")
	if v.Summary.EligibleTargets != 2 || v.Summary.ObservedTargets != 2 || v.Summary.ObtainedEligible != 1 || v.Summary.Percentage == nil || *v.Summary.Percentage != 50 {
		t.Fatal(v.Summary)
	}
	if target(t, getDex(t, f, "scope=game&game_id="+idQuery(f.game.ID)), f.d.ID).Availability != "confirmed_locked" {
		t.Fatal("excluded mega changed species availability")
	}
	a, b := target(t, v, f.a.ID), target(t, v, f.b.ID)
	if a.HistoricalCount != 3 || a.HeldCount != 1 || b.HistoricalCount != 1 || b.HeldCount != 1 {
		t.Fatal(a, b)
	}
	forms := getDex(t, f, "dimension=form")
	if forms.Summary.EligibleTargets != 3 || forms.Summary.ObtainedEligible != 2 || target(t, forms, f.fa.ID).HeldCount != 0 || target(t, forms, f.regional.ID).HeldCount != 1 {
		t.Fatal(forms.Summary)
	}
	orig := getDex(t, f, "scope=game&game_id="+idQuery(f.game.ID))
	if target(t, orig, f.b.ID).HistoricalCount != 1 || orig.Summary.EligibleTargets != 1 {
		t.Fatal(orig.Summary)
	}
	now := getDex(t, f, "basis=current&scope=game&game_id="+idQuery(f.game.ID))
	if target(t, now, f.b.ID).HeldCount != 0 || target(t, now, f.a.ID).HeldCount != 1 {
		t.Fatal(now.Data)
	}
	other := getDex(t, f, "basis=current&scope=game&game_id="+idQuery(f.other.ID))
	if target(t, other, f.b.ID).HeldCount != 1 || other.Summary.ObtainedEligible != 0 {
		t.Fatal(other.Summary)
	}
	var count int64
	f.db.Model(&model.ShinyRecord{}).Count(&count)
	if count != 3 {
		t.Fatal("evolution/transfer created an individual", count)
	}
	r = updateDexRecord(t, f, r, func(in *ShinyInput) { in.HoldingStatus = "traded" })
	v = getDex(t, f, "basis=current")
	if target(t, v, f.b.ID).Obtained {
		t.Fatal("traded individual counted held")
	}
	v = getDex(t, f, "")
	if !target(t, v, f.b.ID).Obtained {
		t.Fatal("traded history lost")
	}
}
func TestDexNoInferredFormCorrectionDeletedOrSleepCredit(t *testing.T) {
	f := dexSeed(t)
	r, e := (Shinies{DB: f.db}).Create(context.Background(), ShinyInput{CurrentSpeciesID: f.c.ID, CurrentGameID: &f.other.ID, HoldingStatus: "unknown"})
	if e != nil {
		t.Fatal(e)
	}
	r = updateDexRecord(t, f, r, func(in *ShinyInput) { in.CurrentSpeciesID = f.d.ID; in.ChangeType = "correction" })
	createDexRecord(t, f, f.a, nil, &f.game.ID, "held")
	sleepRecord := createDexRecord(t, f, f.b, &f.fb.ID, &f.sleep.ID, "held")
	updateDexRecord(t, f, sleepRecord, func(in *ShinyInput) { in.CurrentGameID = nil })
	deleted := createDexRecord(t, f, f.c, &f.fc.ID, &f.game.ID, "held")
	if e := (Shinies{DB: f.db}).Delete(context.Background(), deleted.ID); e != nil {
		t.Fatal(e)
	}
	v := getDex(t, f, "")
	if target(t, v, f.c.ID).HistoricalCount != 0 || target(t, v, f.b.ID).HistoricalCount != 0 || target(t, v, f.d.ID).HistoricalCount != 1 || target(t, v, f.d.ID).HeldCount != 0 {
		t.Fatal(v.Data)
	}
	forms := getDex(t, f, "dimension=form")
	if forms.Summary.ObservedTargets != 0 {
		t.Fatal("unknown form inferred", forms.Summary)
	}
	originalGame := getDex(t, f, "scope=game&game_id="+idQuery(f.other.ID))
	if target(t, originalGame, f.d.ID).HistoricalCount != 0 {
		t.Fatal("current game treated as original")
	}
	sleep := getDex(t, f, "collection_system=sleep")
	if target(t, sleep, f.b.ID).HeldCount != 1 || target(t, sleep, f.a.ID).HeldCount != 0 {
		t.Fatal("collection systems mixed")
	}
	sleepGame := getDex(t, f, "scope=game&game_id="+idQuery(f.sleep.ID))
	if sleepGame.Filters.System != "sleep" || target(t, sleepGame, f.b.ID).HistoricalCount != 1 {
		t.Fatal(sleepGame.Filters)
	}
	unknown := getDex(t, f, "holding=unknown")
	if len(unknown.Data) != 1 || unknown.Data[0].SpeciesID != f.d.ID {
		t.Fatal(unknown.Data)
	}
}
func TestDexRecordedIntermediateEvolutionAndFormChange(t *testing.T) {
	f := dexSeed(t)
	r := createDexRecord(t, f, f.a, &f.fa.ID, &f.game.ID, "held")
	r = updateDexRecord(t, f, r, func(in *ShinyInput) { in.CurrentFormID = &f.regional.ID })
	r = updateDexRecord(t, f, r, func(in *ShinyInput) {
		in.CurrentSpeciesID = f.b.ID
		in.CurrentFormID = &f.fb.ID
		in.ChangeType = "evolution"
	})
	updateDexRecord(t, f, r, func(in *ShinyInput) {
		in.CurrentSpeciesID = f.d.ID
		in.CurrentFormID = &f.fd.ID
		in.ChangeType = "evolution"
	})
	v := getDex(t, f, "dimension=form")
	for _, id := range []uint64{f.fa.ID, f.regional.ID, f.fb.ID, f.fd.ID} {
		if target(t, v, id).HistoricalCount != 1 {
			t.Fatal("intermediate proof absent/duplicated", id)
		}
	}
	v = getDex(t, f, "basis=current&dimension=form&acquired=obtained")
	if len(v.Data) != 1 || v.Data[0].ID != f.fd.ID {
		t.Fatal(v.Data)
	}
}
func TestDexUnknownAvailabilityNullDenominatorFiltersAndEvidence(t *testing.T) {
	f := dexSeed(t)
	f.db.Where("1 = 1").Delete(&model.ShinyAvailability{})
	createDexRecord(t, f, f.a, &f.fa.ID, &f.game.ID, "held")
	v := getDex(t, f, "")
	if v.Summary.EligibleTargets != 0 || v.Summary.Percentage != nil || v.Summary.ObservedOutsideDenominator != 1 {
		t.Fatal(v.Summary)
	}
	v = getDex(t, f, "display=legal")
	if len(v.Data) != 0 || v.Meta.Total != 0 {
		t.Fatal(v.Data)
	}
	addAvailability(t, f.db, f.game, f.fa, "confirmed_available", "unknown")
	if getDex(t, f, "").Summary.EligibleTargets != 0 {
		t.Fatal("unverified availability counted")
	}
	f.db.Model(&model.ShinyAvailability{}).Where("form_id = ?", f.fa.ID).Update("verification_status", "verified")
	v = getDex(t, f, "scope=game&game_id="+idQuery(f.game.ID)+"&acquired=missing&page_size=1")
	if v.Summary.EligibleTargets != 1 || v.Summary.ObtainedEligible != 1 || len(v.Data) != 1 || v.Data[0].Obtained {
		t.Fatal(v.Summary, v.Data)
	}
	v = getDex(t, f, "q=zzz")
	if v.Meta.Total != 0 || v.Summary.ObtainedEligible != 1 {
		t.Fatal("search changed completion scope")
	}
	for _, raw := range []string{"dimension=bad", "basis=bad", "scope=game", "game_id=1", "page=0", "page_size=101", "page=9999999999", "q=a&q=b", "alien=1", "scope=game&game_id=999", "scope=game&game_id=" + idQuery(f.sleep.ID) + "&collection_system=pokemon"} {
		if _, e := (Dex{DB: f.db}).List(context.Background(), raw); e == nil {
			t.Fatal("bad filter accepted", raw)
		}
	}
	f.db.Model(&model.ShinyAvailability{}).Where("form_id = ?", f.fa.ID).Update("source_url", "not-a-url")
	if getDex(t, f, "").Summary.EligibleTargets != 0 {
		t.Fatal("malformed evidence URL counted")
	}
	bad := model.ShinyAvailability{GameID: f.game.ID, SpeciesID: f.b.ID, FormID: f.fb.ID, Status: "confirmed_available", VerificationStatus: "verified"}
	if e := f.db.Create(&bad).Error; e == nil {
		t.Fatal("verified fact without evidence accepted")
	}
}
