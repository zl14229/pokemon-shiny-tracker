package service

import (
	"context"
	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"github.com/zl14229/pokemon-shiny-tracker/internal/testutil"
	"os"
	"testing"
)

func TestAttachmentsFileLinkAndRetention(t *testing.T) {
	db := testutil.DB(t)
	r := testutil.Seed(t, db)
	ctx := context.Background()
	owner, e := (Shinies{DB: db}).Create(ctx, ShinyInput{CurrentSpeciesID: r.Species.ID})
	if e != nil {
		t.Fatal(e)
	}
	s := Attachments{DB: db}
	a, e := s.Upload(ctx, owner.ID, "匿名.txt", []byte("anonymous"))
	if e != nil {
		t.Fatal(e)
	}
	_, data, e := s.Content(ctx, a.ID)
	if e != nil || string(data) != "anonymous" {
		t.Fatal(e)
	}
	b, e := s.Upload(ctx, owner.ID, "匿名.txt", []byte("other"))
	if e != nil || *a.StorageKey == *b.StorageKey {
		t.Fatal("overwrote blob", e)
	}
	link, e := s.Link(ctx, owner.ID, LinkInput{Name: "proof", URL: "https://example.com/proof"})
	if e != nil {
		t.Fatal(e)
	}
	if _, _, e = s.Content(ctx, link.ID); e == nil {
		t.Fatal("link served as file")
	}
	if e = s.Delete(ctx, a.ID); e != nil {
		t.Fatal(e)
	}
	if _, _, e = s.Content(ctx, a.ID); e == nil {
		t.Fatal("deleted association readable")
	}
	store, _ := s.store(ctx)
	if _, e = store.Read(*a.StorageKey, *a.SHA256, *a.SizeBytes); e != nil {
		t.Fatal("deletion destroyed snapshot blob", e)
	}
	rows, e := s.List(ctx, owner.ID)
	if e != nil || len(rows) != 2 {
		t.Fatal(e, len(rows))
	}
	for _, url := range []string{"javascript:alert(1)", "file:///private", "data:text/html,x", "https://user:password@example.com", "https:///missing"} {
		if _, e = s.Link(ctx, owner.ID, LinkInput{Name: "bad", URL: url}); e == nil {
			t.Fatal("unsafe URL", url)
		}
	}
}
func TestAttachmentFailureCleansOnlyNewBlob(t *testing.T) {
	db := testutil.DB(t)
	r := testutil.Seed(t, db)
	ctx := context.Background()
	owner, e := (Shinies{DB: db}).Create(ctx, ShinyInput{CurrentSpeciesID: r.Species.ID})
	if e != nil {
		t.Fatal(e)
	}
	s := Attachments{DB: db}
	a, e := s.Upload(ctx, owner.ID, "old.txt", []byte("preserved"))
	if e != nil {
		t.Fatal(e)
	}
	store, _ := s.store(ctx)
	before, _ := os.ReadDir(store.Root)
	db.Exec(`CREATE TRIGGER reject_attachment BEFORE INSERT ON attachments BEGIN SELECT RAISE(ABORT,'injected'); END`)
	if _, e = s.Upload(ctx, owner.ID, "new.txt", []byte("new")); e == nil {
		t.Fatal("failure expected")
	}
	after, _ := os.ReadDir(store.Root)
	if len(before) != len(after) {
		t.Fatal("orphan not cleaned")
	}
	if _, e = store.Read(*a.StorageKey, *a.SHA256, *a.SizeBytes); e != nil {
		t.Fatal("existing data touched")
	}
	db.Exec("DROP TRIGGER reject_attachment")
	db.Delete(&owner)
	if _, e = s.Upload(ctx, owner.ID, "bad.txt", []byte("data")); e == nil {
		t.Fatal("deleted owner accepted")
	}
	var count int64
	db.Model(&model.Attachment{}).Count(&count)
	if count != 1 {
		t.Fatal(count)
	}
}

func TestAttachmentQuotaAndNullableKeys(t *testing.T) {
	db := testutil.DB(t)
	r := testutil.Seed(t, db)
	ctx := context.Background()
	owner, e := (Shinies{DB: db}).Create(ctx, ShinyInput{CurrentSpeciesID: r.Species.ID})
	if e != nil {
		t.Fatal(e)
	}
	s := Attachments{DB: db}
	for i := 0; i < 100; i++ {
		if _, e = s.Link(ctx, owner.ID, LinkInput{Name: "anonymous", URL: "https://example.com"}); e != nil {
			t.Fatal(i, e)
		}
	}
	if _, e = s.Link(ctx, owner.ID, LinkInput{Name: "over", URL: "https://example.com"}); e == nil {
		t.Fatal("quota exceeded")
	}
}

func TestAttachmentSQLRejectsNullFileSizeAndEmptyLink(t *testing.T) {
	db := testutil.DB(t)
	r := testutil.Seed(t, db)
	owner, e := (Shinies{DB: db}).Create(context.Background(), ShinyInput{CurrentSpeciesID: r.Species.ID})
	if e != nil {
		t.Fatal(e)
	}
	key, hash, empty := "00000000000000000000000000000000.blob", "hash", ""
	for _, a := range []model.Attachment{{ShinyRecordID: owner.ID, Kind: "file", Name: "x.txt", MediaType: "text/plain; charset=utf-8", StorageKey: &key, SHA256: &hash}, {ShinyRecordID: owner.ID, Kind: "link", Name: "x", URL: &empty}} {
		if e := db.Create(&a).Error; e == nil {
			t.Fatal("incomplete attachment metadata accepted")
		}
	}
}
