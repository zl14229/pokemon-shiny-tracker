package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"github.com/zl14229/pokemon-shiny-tracker/internal/testutil"
	"gorm.io/gorm"
)

func TestIndividualIdentityHistoryAndUnknowns(t *testing.T) {
	db := testutil.DB(t)
	r := testutil.Seed(t, db)
	s := Shinies{db}
	ctx := context.Background()
	row, err := s.Create(ctx, ShinyInput{CurrentSpeciesID: r.Species.ID})
	if err != nil {
		t.Fatal(err)
	}
	if row.HuntID != nil || row.OriginalSpeciesID != nil || row.OriginalGameID != nil || row.ObtainedDate != nil || row.Source != "unknown" || row.HoldingStatus != "unknown" {
		t.Fatal("fabricated original facts", row)
	}
	otherForm := model.Form{Labels: model.Labels{Slug: "other-form", Name: "Other Form"}, SpeciesID: r.Other.ID}
	if err = db.Create(&otherForm).Error; err != nil {
		t.Fatal(err)
	}
	home := model.Game{Labels: model.Labels{Slug: "home", Name: "HOME"}, Kind: "storage"}
	if err = db.Create(&home).Error; err != nil {
		t.Fatal(err)
	}
	in := ShinyInput{CurrentSpeciesID: r.Species.ID, CurrentFormID: &r.Form.ID, OriginalSpeciesID: &r.Species.ID, OriginalGameID: &r.Game.ID, Source: "trade", HoldingStatus: "held", ObtainedDate: ptr("2020"), Nickname: ptr(" 宝 ")}
	updated, err := s.Update(ctx, row.ID, in)
	if err != nil || updated.ID != row.ID || updated.Nickname == nil || *updated.Nickname != "宝" {
		t.Fatal(updated, err)
	}
	events, _, total, err := s.History(ctx, row.ID, 100, 0)
	if err != nil || total < 2 {
		t.Fatal(events, err)
	}
	in.ChangeType = "evolution"
	in.CurrentSpeciesID = r.Other.ID
	in.CurrentFormID = &otherForm.ID
	in.CurrentGameID = &home.ID
	in.CurrentLocation = ptr("Box 2")
	in.HoldingStatus = "traded"
	in.EventDate = ptr("2024-02")
	in.HistoryNote = ptr("用户自述变化，未验证进化规则")
	updated, err = s.Update(ctx, row.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != row.ID || *updated.OriginalSpeciesID != r.Species.ID || *updated.OriginalGameID != r.Game.ID || updated.CurrentSpeciesID != r.Other.ID {
		t.Fatal("identity or original facts changed", updated)
	}
	var count int64
	db.Model(&model.ShinyRecord{}).Count(&count)
	if count != 1 {
		t.Fatal("evolution/transfer duplicated record")
	}
	events, deleted, total, err := s.History(ctx, row.ID, 100, 0)
	if err != nil || deleted {
		t.Fatal(err)
	}
	kinds := map[string]bool{}
	for _, ev := range events {
		if !json.Valid([]byte(ev.BeforeJSON)) || !json.Valid([]byte(ev.AfterJSON)) {
			t.Fatal("bad snapshot")
		}
		kinds[ev.Kind] = true
		if ev.Kind == "evolution" && (ev.EventDate == nil || *ev.EventDate != "2024-02") {
			t.Fatal("event date precision lost")
		}
	}
	for _, kind := range []string{"registered", "form_change", "evolution", "location_change", "holding_change"} {
		if !kinds[kind] {
			t.Fatal("missing history", kind, kinds)
		}
	}
	in.ChangeType = ""
	in.Nickname = ptr("宝")
	if _, err = s.Update(ctx, row.ID, in); err != nil {
		t.Fatal(err)
	}
	_, _, afterNoop, _ := s.History(ctx, row.ID, 100, 0)
	if afterNoop != total {
		t.Fatal("no-op created event")
	}
	in.CurrentFormID = nil
	in.CurrentGameID = nil
	in.CurrentLocation = nil
	in.Nickname = nil
	in.EventDate = nil
	cleared, err := s.Update(ctx, row.ID, in)
	if err != nil || cleared.CurrentFormID != nil || cleared.CurrentGameID != nil || cleared.Nickname != nil {
		t.Fatal("nullable fields not cleared", cleared, err)
	}
	if err = s.Delete(ctx, row.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Get(ctx, row.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal(err)
	}
	events, deleted, total, err = s.History(ctx, row.ID, 1, 0)
	if err != nil || !deleted || len(events) != 1 || total < 5 {
		t.Fatal("audit inaccessible", events, deleted, total, err)
	}
	var retained model.ShinyRecord
	if err = db.Unscoped().First(&retained, row.ID).Error; err != nil || retained.ID != row.ID {
		t.Fatal("permanent identity removed")
	}
	rows, n, err := s.List(ctx, "")
	if err != nil || n.Total != 0 || len(rows) != 0 {
		t.Fatal(rows, n, err)
	}
}
func TestCompletedHuntLastOutcomeProtection(t *testing.T) {
	db := testutil.DB(t)
	r := testutil.Seed(t, db)
	ctx := context.Background()
	hs := Hunts{db}
	ss := Shinies{db}
	h, err := hs.Create(ctx, HuntInput{SpeciesID: r.Species.ID})
	if err != nil {
		t.Fatal(err)
	}
	// Non-target outcomes are valid; association never infers source/original game.
	in := ShinyInput{HuntID: &h.ID, CurrentSpeciesID: r.Other.ID, Source: "unknown", HoldingStatus: "held"}
	one, err := ss.Create(ctx, in)
	if err != nil || one.OriginalGameID != nil || one.Source != "unknown" {
		t.Fatal(one, err)
	}
	if _, err = hs.Update(ctx, h.ID, HuntInput{SpeciesID: r.Species.ID, Status: "completed"}); err != nil {
		t.Fatal(err)
	}
	otherHunt, err := hs.Create(ctx, HuntInput{SpeciesID: r.Other.ID})
	if err != nil {
		t.Fatal(err)
	}
	for _, destination := range []*uint64{nil, &otherHunt.ID} {
		replacement := in
		replacement.HuntID = destination
		if _, err = ss.Update(ctx, one.ID, replacement); err == nil {
			t.Fatal("last result detached")
		}
	}
	if err = ss.Delete(ctx, one.ID); err == nil {
		t.Fatal("last result deleted")
	}
	two, err := ss.Create(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if err = ss.Delete(ctx, one.ID); err != nil {
		t.Fatal(err)
	}
	if err = ss.Delete(ctx, two.ID); err == nil {
		t.Fatal("deleted result counted as active")
	}
	if _, err = hs.Update(ctx, h.ID, HuntInput{SpeciesID: r.Species.ID, Status: "abandoned"}); err != nil {
		t.Fatal(err)
	}
	if err = ss.Delete(ctx, two.ID); err != nil {
		t.Fatal(err)
	}
	if err = hs.Delete(ctx, h.ID); err == nil {
		t.Fatal("historical association orphaned")
	}
}
func TestIndividualValidationAndHistoryRollback(t *testing.T) {
	db := testutil.DB(t)
	r := testutil.Seed(t, db)
	ctx := context.Background()
	s := Shinies{db}
	row, err := s.Create(ctx, ShinyInput{CurrentSpeciesID: r.Species.ID, Notes: ptr("original")})
	if err != nil {
		t.Fatal(err)
	}
	sleep := model.Game{Labels: model.Labels{Slug: "sleep", Name: "Sleep"}, Kind: "spin_off", CollectionSystem: "sleep"}
	if err = db.Create(&sleep).Error; err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*ShinyInput){"species": func(in *ShinyInput) { in.CurrentSpeciesID = 0 }, "hunt": func(in *ShinyInput) { in.HuntID = ptr(uint64(9999)) }, "formWithoutSpecies": func(in *ShinyInput) { in.OriginalFormID = &r.Form.ID }, "wrongForm": func(in *ShinyInput) { in.OriginalSpeciesID = &r.Other.ID; in.OriginalFormID = &r.Form.ID }, "level": func(in *ShinyInput) { in.Level = ptr(101) }, "date": func(in *ShinyInput) { in.ObtainedDate = ptr("2023-02-29") }, "eventDate": func(in *ShinyInput) { in.EventDate = ptr("") }, "source": func(in *ShinyInput) { in.Source = "owned" }, "holding": func(in *ShinyInput) { in.HoldingStatus = "lost" }, "speciesWithoutReason": func(in *ShinyInput) { in.CurrentSpeciesID = r.Other.ID }, "falseEvolution": func(in *ShinyInput) { in.ChangeType = "evolution" }, "collection": func(in *ShinyInput) { in.OriginalGameID = &r.Game.ID; in.CurrentGameID = &sleep.ID }} {
		t.Run(name, func(t *testing.T) {
			in := ShinyInput{CurrentSpeciesID: r.Species.ID, Source: "unknown", HoldingStatus: "unknown"}
			change(&in)
			if _, err = s.Update(ctx, row.ID, in); err == nil {
				t.Fatal("invalid input saved")
			}
			got, _ := s.Get(ctx, row.ID)
			if got.Notes == nil || *got.Notes != "original" {
				t.Fatal("validation modified existing record")
			}
		})
	}
	// Every write and history event are one transaction, including creation/deletion.
	if err = db.Exec("CREATE TRIGGER reject_history BEFORE INSERT ON shiny_histories BEGIN SELECT RAISE(ABORT, 'test history failure'); END").Error; err != nil {
		t.Fatal(err)
	}
	if _, err = s.Create(ctx, ShinyInput{CurrentSpeciesID: r.Other.ID}); err == nil {
		t.Fatal("creation ignored failed history")
	}
	if _, err = s.Update(ctx, row.ID, ShinyInput{CurrentSpeciesID: r.Species.ID, Source: "unknown", HoldingStatus: "released", Notes: ptr("changed")}); err == nil {
		t.Fatal("update ignored failed history")
	}
	if err = s.Delete(ctx, row.ID); err == nil {
		t.Fatal("delete ignored failed history")
	}
	got, err := s.Get(ctx, row.ID)
	if err != nil || got.HoldingStatus != "unknown" || *got.Notes != "original" {
		t.Fatal("failed audit not rolled back", got, err)
	}
	var n int64
	db.Model(&model.ShinyRecord{}).Count(&n)
	if n != 1 {
		t.Fatal("failed create persisted")
	}
	db.Model(&model.ShinyHistory{}).Count(&n)
	if n != 1 {
		t.Fatal("partial history persisted")
	}
}

func TestUnknownCollectionAndDeletedHunt(t *testing.T) {
	db := testutil.DB(t)
	r := testutil.Seed(t, db)
	ctx := context.Background()
	s := Shinies{db}
	unknown := model.Game{Labels: model.Labels{Slug: "unknown-system", Name: "Unknown"}, CollectionSystem: "unknown"}
	if err := db.Create(&unknown).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, ShinyInput{CurrentSpeciesID: r.Species.ID, OriginalGameID: &r.Game.ID, CurrentGameID: &unknown.ID}); err != nil {
		t.Fatal("unknown system treated as known incompatible", err)
	}
	hs := Hunts{db}
	h, err := hs.Create(ctx, HuntInput{SpeciesID: r.Species.ID})
	if err != nil {
		t.Fatal(err)
	}
	if err = hs.Delete(ctx, h.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Create(ctx, ShinyInput{CurrentSpeciesID: r.Species.ID, HuntID: &h.ID}); err == nil {
		t.Fatal("linked deleted Hunt")
	}
}
