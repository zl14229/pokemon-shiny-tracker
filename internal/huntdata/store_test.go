package huntdata

import (
	"context"
	"github.com/zl14229/pokemon-shiny-tracker/internal/backup"
	"github.com/zl14229/pokemon-shiny-tracker/internal/database"
	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"github.com/zl14229/pokemon-shiny-tracker/internal/reference"
	"github.com/zl14229/pokemon-shiny-tracker/internal/testutil"
	"path/filepath"
	"testing"
)

func TestBundledPreviewApplyRollbackAndPersonalIsolation(t *testing.T) {
	db := testutil.DB(t)
	if e := reference.EnsureSeed(db); e != nil {
		t.Fatal(e)
	}
	if e := reference.EnsureCatalog(db); e != nil {
		t.Fatal(e)
	}
	d, e := Bundled()
	if e != nil {
		t.Fatal(e)
	}
	if len(d.Targets) != 24 {
		t.Fatal("unexpected coverage")
	}
	a, e := mapTarget(db, d.Targets[0])
	if e != nil {
		t.Fatal(e)
	}
	r := model.ShinyRecord{CurrentSpeciesID: a.SpeciesID, CurrentFormID: &a.FormID, OriginalGameID: &a.GameID, Source: "trade", HoldingStatus: "held"}
	if e = db.Create(&r).Error; e != nil {
		t.Fatal(e)
	}
	before := encode(r)
	report, e := Import(db, d, false, "")
	if e != nil || len(report.Differences) != 24 {
		t.Fatal(report, e)
	}
	rows, e := Read(db)
	if e != nil || len(rows) != 0 {
		t.Fatal(rows, e)
	}
	if _, e = Import(db, d, true, "wrong"); e == nil {
		t.Fatal("unreviewed apply")
	}
	if _, e = Import(db, d, true, report.Checksum); e != nil {
		t.Fatal(e)
	}
	rows, e = Read(db)
	if e != nil || len(rows) != 24 {
		t.Fatal(len(rows), e)
	}
	if rows[0].Fact.Methods[0].EncounterRate != nil || rows[0].Fact.Methods[0].ShinyRate.Denominator != 4096 {
		t.Fatal("rate confusion")
	}
	second, e := Import(db, d, true, report.Checksum)
	if e != nil || len(second.Differences) != 0 {
		t.Fatal(second, e)
	}
	d2 := d
	d2.Version = "other"
	if _, e = Import(db, d2, true, Digest(d2)); e == nil {
		t.Fatal("overlap accepted")
	}
	if e = Rollback(db, d.Version, report.Checksum); e != nil {
		t.Fatal(e)
	}
	rows, e = Read(db)
	if e != nil || len(rows) != 0 {
		t.Fatal(rows, e)
	}
	var after model.ShinyRecord
	if e = db.First(&after, r.ID).Error; e != nil || encode(after) != before {
		t.Fatal("personal record changed", e)
	}
	var n int64
	db.Model(&model.Species{}).Count(&n)
	if n != 1025 {
		t.Fatal("identities removed")
	}
	if _, e = Import(db, d, true, report.Checksum); e == nil {
		t.Fatal("rolled back batch silently reinstalled")
	}
}
func TestFailedMappingIsAtomic(t *testing.T) {
	db := testutil.DB(t)
	reference.EnsureSeed(db)
	reference.EnsureCatalog(db)
	d, _ := Bundled()
	d.Targets[len(d.Targets)-1].SpeciesID = 25
	if _, e := Import(db, d, true, Digest(d)); e == nil {
		t.Fatal("mismatch allowed")
	}
	var n int64
	db.Model(&model.ShinyAvailability{}).Count(&n)
	if n != 0 {
		t.Fatal("partial apply")
	}
}
func TestModifiedEvidenceFailsClosed(t *testing.T) {
	db := testutil.DB(t)
	reference.EnsureSeed(db)
	reference.EnsureCatalog(db)
	if e := EnsureBundled(db); e != nil {
		t.Fatal(e)
	}
	db.Model(&model.ShinyAvailability{}).Where("id > 0").Update("status", "transfer_only")
	if _, e := Read(db); e == nil {
		t.Fatal("changed source accepted")
	}
}

func TestReferenceBackupRestoresOffline(t *testing.T) {
	db := testutil.DB(t)
	reference.EnsureSeed(db)
	reference.EnsureCatalog(db)
	if e := EnsureBundled(db); e != nil {
		t.Fatal(e)
	}
	var source string
	db.Raw("PRAGMA database_list").Row().Scan(new(int), new(string), &source)
	zip := filepath.Join(t.TempDir(), "snapshot.zip")
	if _, e := backup.Backup(context.Background(), source, zip); e != nil {
		t.Fatal(e)
	}
	dest := filepath.Join(t.TempDir(), "restored")
	if _, e := backup.Restore(context.Background(), zip, dest); e != nil {
		t.Fatal(e)
	}
	restored, e := database.Open(filepath.Join(dest, "database.sqlite3"))
	if e != nil {
		t.Fatal(e)
	}
	conn, _ := restored.DB()
	defer conn.Close()
	rows, e := Read(restored)
	if e != nil || len(rows) != 24 {
		t.Fatal(len(rows), e)
	}
}
