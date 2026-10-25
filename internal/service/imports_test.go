package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"github.com/zl14229/pokemon-shiny-tracker/internal/reference"
	"github.com/zl14229/pokemon-shiny-tracker/internal/testutil"
	"github.com/zl14229/pokemon-shiny-tracker/internal/workbook"
	"os"
	"testing"
	"time"
)

func TestImportPreviewConfirmIdempotenceAndDuplicates(t *testing.T) {
	db := testutil.DB(t)
	testutil.Seed(t, db)
	s := NewImports(db)
	ctx := context.Background()
	data := testutil.XLSX(t, [][]string{{"编号", "日期", "球种", "次数"}, {"25", "2019-02", "匿名球", "100"}, {"25", "bad", "", ""}, {"25", "2019-02", "匿名球", ""}, {"999999", "", "", ""}})
	mapping := ImportMapping{Sheet: "匿名样本", HeaderRow: 1, Columns: map[string]string{"A": "current_national_dex", "B": "obtained_date", "C": "ball"}}
	p, e := s.Preview(ctx, data, mapping)
	if e != nil {
		t.Fatal(e)
	}
	if len(p.Rows) != 4 || len(p.Rows[0].Errors) > 0 || len(p.Rows[0].Warnings) == 0 || len(p.Rows[1].Errors) == 0 || len(p.Rows[2].Warnings) == 0 || len(p.Rows[3].Errors) == 0 {
		t.Fatal("incorrect per-row validation", p)
	}
	var n int64
	db.Model(&model.ShinyRecord{}).Count(&n)
	if n != 0 {
		t.Fatal("preview wrote records")
	}
	if _, e = s.Confirm(ctx, ImportConfirm{Token: p.Token, Rows: []int{2}, AcknowledgeWarnings: false}); e == nil {
		t.Fatal("warnings silently ignored")
	}
	if _, e = s.Confirm(ctx, ImportConfirm{Token: p.Token, Rows: []int{2, 3}, AcknowledgeWarnings: true}); e == nil {
		t.Fatal("invalid row written")
	}
	r, e := s.Confirm(ctx, ImportConfirm{Token: p.Token, Rows: []int{2, 4}, AcknowledgeWarnings: true})
	if e != nil {
		t.Fatal(e)
	}
	again, e := s.Confirm(ctx, ImportConfirm{Token: p.Token, Rows: []int{4, 2}, AcknowledgeWarnings: true})
	if e != nil || again.Batch.ID != r.Batch.ID {
		t.Fatal("retry not idempotent", e)
	}
	if _, e = s.Confirm(ctx, ImportConfirm{Token: p.Token, Rows: []int{2}, AcknowledgeWarnings: true}); e == nil {
		t.Fatal("reused token for other selection")
	}
	var records []model.ShinyRecord
	db.Find(&records)
	if len(records) != 2 || records[0].HuntID != nil || records[0].OriginalSpeciesID != nil || records[0].Source != "unknown" || records[0].HoldingStatus != "unknown" {
		t.Fatal("inferred unknown fields")
	}
	db.Model(&model.Hunt{}).Count(&n)
	if n != 0 {
		t.Fatal("fabricated hunt")
	}
	db.Model(&model.ShinyHistory{}).Count(&n)
	if n != 2 {
		t.Fatal("missing histories")
	}
	p, e = s.Preview(ctx, data, mapping)
	if e != nil || len(p.Rows[0].Candidates) != 2 {
		t.Fatal("existing duplicate candidates missing", e)
	}
}
func TestImportAtomicFailureExpiryAndConcurrentChanges(t *testing.T) {
	db := testutil.DB(t)
	testutil.Seed(t, db)
	s := NewImports(db)
	ctx := context.Background()
	data := testutil.XLSX(t, [][]string{{"id"}, {"25"}, {"26"}})
	m := ImportMapping{Sheet: "匿名样本", HeaderRow: 1, Columns: map[string]string{"A": "current_national_dex"}}
	p, e := s.Preview(ctx, data, m)
	if e != nil {
		t.Fatal(e)
	}
	db.Exec(`CREATE TRIGGER fail_import BEFORE INSERT ON import_rows WHEN NEW.row_number=3 BEGIN SELECT RAISE(ABORT,'injected'); END`)
	if _, e = s.Confirm(ctx, ImportConfirm{Token: p.Token, Rows: []int{2, 3}, AcknowledgeWarnings: true}); e == nil {
		t.Fatal("expected failure")
	}
	for _, v := range []any{&model.ShinyRecord{}, &model.ShinyHistory{}, &model.ImportBatch{}, &model.ImportRow{}} {
		var n int64
		db.Model(v).Count(&n)
		if n != 0 {
			t.Fatal("partial transaction persisted")
		}
	}
	db.Exec("DROP TRIGGER fail_import")
	db.Create(&model.ShinyRecord{CurrentSpeciesID: 1})
	if _, e = s.Confirm(ctx, ImportConfirm{Token: p.Token, Rows: []int{2}}); e == nil {
		t.Fatal("did not recheck live duplicates")
	}
	p.ExpiresAt = time.Now().Add(-time.Second)
	s.previews[p.Token] = p
	if _, e = s.Confirm(ctx, ImportConfirm{Token: p.Token, Rows: []int{2}, AcknowledgeWarnings: true}); e == nil {
		t.Fatal("expired preview accepted")
	}
}

// Opt-in local read-only validation. Neither sample values nor private filenames
// are logged or saved; CI has no personal sample and never claims this check.
func TestRealWorkbookMappingReadOnly(t *testing.T) {
	path := os.Getenv("PST_REAL_XLSX")
	if path == "" {
		t.Skip("real sample only when explicitly provided")
	}
	before, e := os.ReadFile(path)
	if e != nil {
		t.Fatal("sample unavailable")
	}
	h := sha256.Sum256(before)
	b, e := workbook.Read(before)
	if e != nil {
		t.Fatal(e)
	}
	db := testutil.DB(t)
	if e := reference.EnsureSeed(db); e != nil {
		t.Fatal(e)
	}
	s := NewImports(db)
	valid, invalid, warnings, previews := 0, 0, 0, 0
	for _, sheet := range b.Sheets {
		if len(sheet.Rows) == 0 {
			continue
		}
		headers := sheet.Rows[0].Cells
		cols := map[string]string{}
		for col, c := range headers {
			switch c.Value {
			case "编号":
				cols[col] = "current_national_dex"
			case "日期":
				cols[col] = "obtained_date"
			case "球种":
				cols[col] = "ball"
			case "形态", "获得方式", "次数", "头目", "游戏版本":
				cols[col] = "notes"
			}
		}
		if len(cols) == 0 {
			continue
		}
		p, e := s.Preview(context.Background(), before, ImportMapping{Sheet: sheet.Name, HeaderRow: sheet.Rows[0].Number, Columns: cols})
		if e != nil {
			t.Fatal("real mapping preview rejected", e)
		}
		previews++
		for _, r := range p.Rows {
			if len(r.Errors) == 0 {
				valid++
			} else {
				invalid++
			}
			warnings += len(r.Warnings)
		}
		s.Discard(p.Token)
	}
	var n int64
	db.Model(&model.ShinyRecord{}).Count(&n)
	after, e := os.ReadFile(path)
	if e != nil || sha256.Sum256(after) != h || n != 0 || valid == 0 || previews == 0 {
		t.Fatal("read-only mapping validation failed")
	}
	t.Logf("read-only sample: SHA256=%s sheets=%d mapped=%d valid_rows=%d error_rows=%d warnings=%d personal_writes=0 hash_unchanged=true", hex.EncodeToString(h[:]), len(b.Sheets), previews, valid, invalid, warnings)
}

func TestExplicitValueMappingAndCacheLimit(t *testing.T) {
	db := testutil.DB(t)
	r := testutil.Seed(t, db)
	s := NewImports(db)
	data := testutil.XLSX(t, [][]string{{"编号", "来源", "游戏"}, {"25", "交换", "已核对版本"}})
	m := ImportMapping{Sheet: "匿名样本", HeaderRow: 1, Columns: map[string]string{"A": "current_national_dex", "B": "source", "C": "original_game_id"}, Values: map[string]map[string]string{"source": {"交换": "trade"}, "original_game_id": {"已核对版本": "1"}}}
	ctx := context.Background()
	p, e := s.Preview(ctx, data, m)
	if e != nil || len(p.Rows[0].Errors) > 0 || p.Rows[0].Input.Source != "trade" || !same(p.Rows[0].Input.OriginalGameID, &r.Game.ID) {
		t.Fatal("explicit mappings failed", e)
	}
	for i := 0; i < 2; i++ {
		if _, e = s.Preview(ctx, data, m); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = s.Preview(ctx, data, m); e == nil {
		t.Fatal("unbounded preview cache")
	}
	s.Discard(p.Token)
	if _, e = s.Preview(ctx, data, m); e != nil {
		t.Fatal(e)
	}
}
