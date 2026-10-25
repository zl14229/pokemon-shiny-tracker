package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"github.com/zl14229/pokemon-shiny-tracker/internal/workbook"
	"gorm.io/gorm"
)

type ImportMapping struct {
	Sheet     string                       `json:"sheet"`
	HeaderRow int                          `json:"header_row"`
	Columns   map[string]string            `json:"columns"`
	Values    map[string]map[string]string `json:"values"`
}
type ImportPreviewRow struct {
	Number     int                      `json:"number"`
	Input      ShinyInput               `json:"input"`
	Errors     []string                 `json:"errors"`
	Warnings   []string                 `json:"warnings"`
	Candidates []uint64                 `json:"candidates"`
	Raw        map[string]workbook.Cell `json:"raw"`
}
type ImportPreview struct {
	Token      string             `json:"token"`
	ExpiresAt  time.Time          `json:"expires_at"`
	FileSHA256 string             `json:"file_sha256"`
	Mapping    ImportMapping      `json:"mapping"`
	Rows       []ImportPreviewRow `json:"rows"`
}
type ImportConfirm struct {
	Token               string `json:"token"`
	Rows                []int  `json:"rows"`
	AcknowledgeWarnings bool   `json:"acknowledge_warnings"`
}
type ImportResult struct {
	Batch model.ImportBatch `json:"batch"`
	Rows  []model.ImportRow `json:"rows"`
}
type Imports struct {
	DB       *gorm.DB
	mu       sync.Mutex
	previews map[string]ImportPreview
}

func NewImports(db *gorm.DB) *Imports { return &Imports{DB: db, previews: map[string]ImportPreview{}} }
func importFields() map[string]bool {
	m := map[string]bool{}
	for _, s := range strings.Fields("current_species_id current_national_dex original_species_id original_national_dex current_form_id original_form_id original_game_id current_game_id current_location obtained_date source holding_status nickname gender nature ability ball level notes") {
		m[s] = true
	}
	return m
}
func (s *Imports) Preview(ctx context.Context, data []byte, m ImportMapping) (ImportPreview, error) {
	b, e := workbook.Read(data)
	if e != nil {
		return ImportPreview{}, invalid(e.Error())
	}
	if m.HeaderRow < 1 || len(m.Columns) == 0 {
		return ImportPreview{}, invalid("请选择表头行和字段映射")
	}
	fields := importFields()
	seen := map[string]bool{}
	for col, f := range m.Columns {
		if len(col) < 1 || len(col) > 2 || !fields[f] || seen[f] && f != "notes" {
			return ImportPreview{}, invalid("列映射无效或重复")
		}
		seen[f] = true
	}
	if seen["current_species_id"] == seen["current_national_dex"] {
		return ImportPreview{}, invalid("必须且只能映射当前物种ID或全国编号之一")
	}
	if seen["original_species_id"] && seen["original_national_dex"] {
		return ImportPreview{}, invalid("原始物种映射重复")
	}
	var sheet *workbook.Sheet
	for i := range b.Sheets {
		if b.Sheets[i].Name == m.Sheet {
			sheet = &b.Sheets[i]
		}
	}
	if sheet == nil {
		return ImportPreview{}, invalid("工作表不存在")
	}
	var records []model.ShinyRecord
	if e = s.DB.WithContext(ctx).Unscoped().Find(&records).Error; e != nil {
		return ImportPreview{}, e
	}
	sum := sha256.Sum256(data)
	p := ImportPreview{FileSHA256: hex.EncodeToString(sum[:]), Mapping: m, Rows: []ImportPreviewRow{}, ExpiresAt: time.Now().UTC().Add(30 * time.Minute)}
	previous := map[string]int{}
	for _, r := range sheet.Rows {
		if r.Number <= m.HeaderRow {
			continue
		}
		nonempty := false
		for _, c := range r.Cells {
			if strings.TrimSpace(c.Value) != "" {
				nonempty = true
			}
		}
		if !nonempty {
			continue
		}
		row := ImportPreviewRow{Number: r.Number, Raw: r.Cells, Errors: []string{}, Warnings: []string{}, Candidates: []uint64{}}
		values := map[string]any{}
		notes := []string{}
		cols := []string{}
		for c := range r.Cells {
			cols = append(cols, c)
		}
		sort.Strings(cols)
		for _, col := range cols {
			c := r.Cells[col]
			v := strings.TrimSpace(c.Value)
			if v == "" {
				continue
			}
			f := m.Columns[col]
			if f == "" {
				row.Warnings = append(row.Warnings, col+":未映射，原值仅保存于批次溯源")
				continue
			}
			if c.Issue != "" {
				row.Errors = append(row.Errors, col+":"+c.Issue)
				continue
			}
			if c.Formula {
				row.Warnings = append(row.Warnings, col+":仅使用公式缓存值，未计算，须核对新鲜度")
			}
			if mapped, ok := m.Values[f][v]; ok {
				v = mapped
			}
			if f == "notes" {
				notes = append(notes, col+": "+v)
				continue
			}
			if strings.HasSuffix(f, "_id") || strings.HasSuffix(f, "_dex") || f == "level" {
				n, err := strconv.ParseUint(strings.TrimPrefix(v, "#"), 10, 53)
				if err != nil || n == 0 {
					row.Errors = append(row.Errors, col+":需要正整数ID/全国编号或明确的值映射")
					continue
				}
				if strings.HasSuffix(f, "_dex") {
					var sp model.Species
					if e := s.DB.WithContext(ctx).Where("national_dex = ?", n).First(&sp).Error; e != nil {
						if e != gorm.ErrRecordNotFound {
							return ImportPreview{}, e
						}
						row.Errors = append(row.Errors, col+":全国编号不在本地资料库，请先补充经校验资料")
						continue
					}
					f = strings.Replace(f, "national_dex", "species_id", 1)
					n = sp.ID
				}
				values[f] = n
			} else {
				values[f] = v
			}
		}
		if len(notes) > 0 {
			values["notes"] = strings.Join(notes, "\n")
		}
		raw, _ := json.Marshal(values)
		if json.Unmarshal(raw, &row.Input) != nil {
			row.Errors = append(row.Errors, "映射后的字段类型不受支持")
		}
		if e := row.Input.validate(s.DB.WithContext(ctx), true); e != nil {
			if p, ok := e.(*Error); ok {
				row.Errors = append(row.Errors, p.Message)
			} else {
				return ImportPreview{}, e
			}
		}
		for _, old := range records {
			if duplicateImport(row.Input, old) {
				row.Candidates = append(row.Candidates, old.ID)
			}
		}
		if len(row.Candidates) > 0 {
			row.Warnings = append(row.Warnings, "已有个体疑似重复（包含已删除身份），不能据此断言同一只")
		}
		k := fmt.Sprintf("%d/%v/%v", row.Input.CurrentSpeciesID, ptrValue(row.Input.OriginalGameID), ptrValue(row.Input.ObtainedDate))
		if n, ok := previous[k]; ok {
			row.Warnings = append(row.Warnings, fmt.Sprintf("与本表第%d行疑似重复，仍可为不同个体", n))
		}
		previous[k] = r.Number
		p.Rows = append(p.Rows, row)
	}
	if len(p.Rows) == 0 {
		return ImportPreview{}, invalid("表头后没有可预览的非空行")
	}
	token := make([]byte, 24)
	if _, e = rand.Read(token); e != nil {
		return ImportPreview{}, e
	}
	p.Token = hex.EncodeToString(token)
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range s.previews {
		if time.Now().After(v.ExpiresAt) {
			delete(s.previews, k)
		}
	}
	if len(s.previews) >= 3 {
		return ImportPreview{}, conflict("预览缓存已满，请清除旧预览或等待30分钟")
	}
	s.previews[p.Token] = p
	return p, nil
}
func ptrValue[T comparable](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}
func duplicateImport(in ShinyInput, r model.ShinyRecord) bool {
	return in.CurrentSpeciesID == r.CurrentSpeciesID && same(in.OriginalGameID, r.OriginalGameID) && same(in.ObtainedDate, r.ObtainedDate)
}
func (s *Imports) Discard(token string) { s.mu.Lock(); delete(s.previews, token); s.mu.Unlock() }
func (s *Imports) Confirm(ctx context.Context, in ImportConfirm) (ImportResult, error) {
	result := ImportResult{Rows: []model.ImportRow{}}
	if len(in.Token) != 48 || len(in.Rows) == 0 || len(in.Rows) > 500 {
		return result, invalid("每批明确选择1至500行")
	}
	sort.Ints(in.Rows)
	for i, n := range in.Rows {
		if n < 1 || i > 0 && n == in.Rows[i-1] {
			return result, invalid("行号重复或无效")
		}
	}
	selected, _ := json.Marshal(in.Rows)
	s.mu.Lock()
	p, ok := s.previews[in.Token]
	s.mu.Unlock()
	e := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Where("token = ?", in.Token).First(&result.Batch).Error
		if err == nil {
			if result.Batch.SelectedJSON != string(selected) {
				return conflict("此确认令牌已用于其他行选择")
			}
			return tx.Where("import_batch_id = ?", result.Batch.ID).Order("row_number").Find(&result.Rows).Error
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}
		if !ok || time.Now().After(p.ExpiresAt) {
			return conflict("预览已失效，请重新预览后确认")
		}
		rows := map[int]ImportPreviewRow{}
		for _, r := range p.Rows {
			rows[r.Number] = r
		}
		var existing []model.ShinyRecord
		if err = tx.Unscoped().Find(&existing).Error; err != nil {
			return err
		}
		for _, n := range in.Rows {
			r, exists := rows[n]
			if !exists || len(r.Errors) > 0 {
				return invalid("所选行包含错误或不在预览中")
			}
			warn := len(r.Warnings) > 0
			for _, old := range existing {
				if duplicateImport(r.Input, old) {
					warn = true
				}
			}
			if warn && !in.AcknowledgeWarnings {
				return conflict("请核对疑似重复、公式缓存或未映射列并显式确认警告")
			}
		}
		mapping, _ := json.Marshal(p.Mapping)
		result.Batch = model.ImportBatch{Token: in.Token, FileSHA256: p.FileSHA256, Sheet: p.Mapping.Sheet, MappingJSON: string(mapping), SelectedJSON: string(selected)}
		if err = tx.Create(&result.Batch).Error; err != nil {
			return err
		}
		for _, n := range in.Rows {
			r := rows[n]
			row, err := (Shinies{DB: tx}).Create(ctx, r.Input)
			if err != nil {
				return err
			}
			raw, _ := json.Marshal(r.Raw)
			trace := model.ImportRow{ImportBatchID: result.Batch.ID, RowNumber: n, ShinyRecordID: row.ID, RawJSON: string(raw)}
			if err = tx.Create(&trace).Error; err != nil {
				return err
			}
			result.Rows = append(result.Rows, trace)
		}
		return nil
	})
	if e == nil {
		s.Discard(in.Token)
	}
	return result, e
}
