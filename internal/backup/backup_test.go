package backup_test

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"github.com/zl14229/pokemon-shiny-tracker/internal/backup"
	"github.com/zl14229/pokemon-shiny-tracker/internal/database"
	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"github.com/zl14229/pokemon-shiny-tracker/internal/service"
	"github.com/zl14229/pokemon-shiny-tracker/internal/testutil"
	"gorm.io/gorm"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func source(t *testing.T) (string, *gorm.DB) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "source.sqlite3")
	db, e := database.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	sql, _ := db.DB()
	t.Cleanup(func() { _ = sql.Close() })
	testutil.Seed(t, db)
	return path, db
}
func populated(t *testing.T) (string, *gorm.DB, model.Attachment) {
	t.Helper()
	p, db := source(t)
	ctx := context.Background()
	h, e := (service.Hunts{DB: db}).Create(ctx, service.HuntInput{SpeciesID: 1})
	if e != nil {
		t.Fatal(e)
	}
	r, e := (service.Shinies{DB: db}).Create(ctx, service.ShinyInput{CurrentSpeciesID: 1, HuntID: &h.ID})
	if e != nil {
		t.Fatal(e)
	}
	s := service.Attachments{DB: db}
	a, e := s.Upload(ctx, r.ID, "anonymous.txt", []byte("retained anonymous evidence"))
	if e != nil {
		t.Fatal(e)
	}
	old, e := s.Upload(ctx, r.ID, "old.txt", []byte("soft-deleted evidence"))
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Delete(ctx, old.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Link(ctx, r.ID, service.LinkInput{Name: "proof", URL: "https://example.com/proof"}); e != nil {
		t.Fatal(e)
	}
	imp := service.NewImports(db)
	f := testutil.XLSX(t, [][]string{{"id"}, {"26"}})
	preview, e := imp.Preview(ctx, f, service.ImportMapping{Sheet: "匿名样本", HeaderRow: 1, Columns: map[string]string{"A": "current_national_dex"}})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = imp.Confirm(ctx, service.ImportConfirm{Token: preview.Token, Rows: []int{2}, AcknowledgeWarnings: true}); e != nil {
		t.Fatal(e)
	}
	return p, db, a
}
func TestRoundTripAndExportPreserveIDsHistoryImportsAndFiles(t *testing.T) {
	p, db, a := populated(t)
	ctx := context.Background()
	archive := filepath.Join(t.TempDir(), "backup.zip")
	before, _ := os.ReadFile(p)
	m, e := backup.Backup(ctx, p, archive)
	if e != nil {
		t.Fatal(e)
	}
	after, _ := os.ReadFile(p)
	if sha256.Sum256(before) != sha256.Sum256(after) {
		t.Fatal("backup modified source database")
	}
	if len(m.Files) != 2 {
		t.Fatal("soft-deleted file omitted")
	}
	if _, e = backup.Verify(ctx, archive); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 2; i++ {
		dest := filepath.Join(t.TempDir(), "restored")
		if _, e = backup.Restore(ctx, archive, dest); e != nil {
			t.Fatal(e)
		}
		restored, e := database.Open(filepath.Join(dest, "database.sqlite3"))
		if e != nil {
			t.Fatal(e)
		}
		sql, _ := restored.DB()
		var n int64
		for _, entry := range []struct {
			value any
			count int64
		}{{&model.ShinyRecord{}, 2}, {&model.ShinyHistory{}, 2}, {&model.ImportBatch{}, 1}, {&model.ImportRow{}, 1}, {&model.Hunt{}, 1}} {
			restored.Model(entry.value).Count(&n)
			if n != entry.count {
				t.Fatal("identity/history mismatch", n, entry.count)
			}
		}
		restored.Unscoped().Model(&model.Attachment{}).Count(&n)
		if n != 3 {
			t.Fatal("attachment history lost")
		}
		_, data, e := (service.Attachments{DB: restored}).Content(ctx, a.ID)
		if e != nil || string(data) != "retained anonymous evidence" {
			t.Fatal(e)
		}
		r, e := (service.Shinies{DB: restored}).Get(ctx, 2)
		if e != nil || r.HuntID != nil || r.OriginalSpeciesID != nil || r.ObtainedDate != nil || r.Source != "unknown" {
			t.Fatal("inferred data after restore", e)
		}
		_ = sql.Close()
	}
	output := filepath.Join(t.TempDir(), "export.json")
	if e = backup.Export(ctx, p, output); e != nil {
		t.Fatal(e)
	}
	data, _ := os.ReadFile(output)
	var doc struct {
		Format string
		Tables map[string][]map[string]any
	}
	if e = json.Unmarshal(data, &doc); e != nil {
		t.Fatal(e, string(data))
	}
	if len(doc.Tables) != 12 || len(doc.Tables["import_rows"]) != 1 || len(doc.Tables["attachments"]) != 3 {
		t.Fatal("export lost provenance")
	}
	found := false
	for _, row := range doc.Tables["attachments"] {
		if row["deleted_at"] != nil {
			found = true
		}
	}
	if !found {
		t.Fatal("softdelete state omitted")
	}
	var count int64
	db.Model(&model.ShinyRecord{}).Count(&count)
	if count != 2 {
		t.Fatal("source changed")
	}
}
func TestEmptyFilesAndNoOverwrite(t *testing.T) {
	p, _ := source(t)
	ctx := context.Background()
	out := filepath.Join(t.TempDir(), "empty.zip")
	if _, e := backup.Backup(ctx, p, out); e != nil {
		t.Fatal(e)
	}
	data, _ := os.ReadFile(out)
	if _, e := backup.Backup(ctx, p, out); e == nil {
		t.Fatal("overwrote backup")
	}
	again, _ := os.ReadFile(out)
	if sha256.Sum256(data) != sha256.Sum256(again) {
		t.Fatal("existing output changed")
	}
	dest := t.TempDir()
	marker := filepath.Join(dest, "personal-marker")
	_ = os.WriteFile(marker, []byte("keep"), 0600)
	if _, e := backup.Restore(ctx, out, dest); e == nil {
		t.Fatal("overwrote existing restore location")
	}
	if b, _ := os.ReadFile(marker); string(b) != "keep" {
		t.Fatal("personal marker damaged")
	}
	c, cancel := context.WithCancel(ctx)
	cancel()
	canceled := filepath.Join(t.TempDir(), "canceled.zip")
	if _, e := backup.Backup(c, p, canceled); e == nil {
		t.Fatal("canceled backup succeeded")
	}
	if _, e := os.Stat(canceled); !os.IsNotExist(e) {
		t.Fatal("partial output published")
	}
}
func rewrite(t *testing.T, archive, out string, transform func(string, []byte) (string, []byte), extra bool) {
	t.Helper()
	in, e := zip.OpenReader(archive)
	if e != nil {
		t.Fatal(e)
	}
	defer in.Close()
	f, e := os.Create(out)
	if e != nil {
		t.Fatal(e)
	}
	z := zip.NewWriter(f)
	for _, part := range in.File {
		r, e := part.Open()
		if e != nil {
			t.Fatal(e)
		}
		b, e := io.ReadAll(r)
		_ = r.Close()
		if e != nil {
			t.Fatal(e)
		}
		name, b := transform(part.Name, b)
		if name == "" {
			continue
		}
		w, e := z.Create(name)
		if e != nil {
			t.Fatal(e)
		}
		_, _ = w.Write(b)
	}
	if extra {
		w, _ := z.Create("../escape")
		_, _ = w.Write([]byte("bad"))
	}
	_ = z.Close()
	_ = f.Close()
}
func TestCorruptionMissingFilesAndArchiveTraversalAreRejected(t *testing.T) {
	p, _, a := populated(t)
	ctx := context.Background()
	good := filepath.Join(t.TempDir(), "good.zip")
	if _, e := backup.Backup(ctx, p, good); e != nil {
		t.Fatal(e)
	}
	for _, mode := range []string{"corrupt", "missing", "traversal"} {
		bad := filepath.Join(t.TempDir(), "bad.zip")
		rewrite(t, good, bad, func(name string, b []byte) (string, []byte) {
			if name == "files/"+*a.StorageKey {
				if mode == "missing" {
					return "", nil
				}
				if mode == "corrupt" {
					b = []byte("damaged")
				}
			}
			return name, b
		}, mode == "traversal")
		dest := filepath.Join(t.TempDir(), "restored")
		if _, e := backup.Restore(ctx, bad, dest); e == nil {
			t.Fatal("unsafe archive accepted", mode)
		}
		if _, e := os.Stat(dest); !os.IsNotExist(e) {
			t.Fatal("failed restore published directory")
		}
	}
	_ = os.Remove(filepath.Join(p+".files", *a.StorageKey))
	failed := filepath.Join(t.TempDir(), "missing.zip")
	if _, e := backup.Backup(ctx, p, failed); e == nil {
		t.Fatal("missing source file silently omitted")
	}
	if _, e := os.Stat(failed); !os.IsNotExist(e) {
		t.Fatal("invalid backup published")
	}
}
func TestBackupWithConcurrentWritesAndSoftDelete(t *testing.T) {
	p, db, a := populated(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		for i := 0; i < 10; i++ {
			if _, e := (service.Attachments{DB: db}).Upload(ctx, a.ShinyRecordID, "concurrent.txt", []byte("anonymous concurrent upload")); e != nil {
				done <- e
				return
			}
			if _, e := (service.Shinies{DB: db}).Create(ctx, service.ShinyInput{CurrentSpeciesID: 1}); e != nil {
				done <- e
				return
			}
		}
		done <- (service.Attachments{DB: db}).Delete(ctx, a.ID)
	}()
	out := filepath.Join(t.TempDir(), "concurrent.zip")
	_, e := backup.Backup(ctx, p, out)
	writerErr := <-done
	if e != nil || writerErr != nil {
		t.Fatal(e, writerErr)
	}
	if _, e = backup.Verify(ctx, out); e != nil {
		t.Fatal(e)
	}
}
