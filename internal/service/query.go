package service

import (
	"context"
	"math"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"gorm.io/gorm"
)

type Page struct {
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}
type listQuery struct {
	page                       Page
	filters                    map[string]string
	search, sort, order, table string
}

var huntIDs = map[string]string{"species_id": "species_id", "form_id": "form_id", "game_id": "game_id"}
var shinyIDs = map[string]string{"hunt_id": "hunt_id", "original_species_id": "original_species_id", "original_form_id": "original_form_id", "current_species_id": "current_species_id", "current_form_id": "current_form_id", "original_game_id": "original_game_id", "current_game_id": "current_game_id"}
var huntDates = map[string]string{"start_from": "start_date", "start_to": "start_date", "end_from": "end_date", "end_to": "end_date"}
var shinyDates = map[string]string{"obtained_from": "obtained_date", "obtained_to": "obtained_date"}

func queryMaps(kind string) (map[string]string, map[string]string) {
	if kind == "hunt" {
		return huntIDs, huntDates
	}
	return shinyIDs, shinyDates
}
func parseList(raw, kind string) (listQuery, error) {
	q := listQuery{page: Page{Page: 1, PageSize: 50}, filters: map[string]string{}, sort: "created_at", order: "desc", table: "shiny_records"}
	if kind == "hunt" {
		q.table = "hunts"
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return q, invalid("查询编码无效")
	}
	ids, dates := queryMaps(kind)
	allowed := map[string]bool{"page": true, "page_size": true, "q": true, "sort": true, "order": true}
	for k := range ids {
		allowed[k] = true
	}
	for k := range dates {
		allowed[k] = true
	}
	if kind == "hunt" {
		for _, k := range []string{"status", "method", "count_unit", "has_results", "min_count", "max_count", "min_duration", "max_duration"} {
			allowed[k] = true
		}
	} else {
		for _, k := range []string{"source", "holding_status", "has_hunt", "current_location"} {
			allowed[k] = true
		}
	}
	for key, vs := range values {
		if !allowed[key] || len(vs) != 1 {
			return q, invalid("查询字段未知或重复")
		}
		if !utf8.ValidString(vs[0]) {
			return q, invalid("查询编码无效")
		}
		value := strings.TrimSpace(vs[0])
		if value == "" {
			return q, invalid("查询字段不能为空")
		}
		switch key {
		case "page", "page_size":
			n, e := strconv.ParseUint(value, 10, 31)
			if e != nil || n == 0 {
				return q, invalid("分页参数须为正整数")
			}
			if key == "page" {
				q.page.Page = int(n)
			} else {
				q.page.PageSize = int(n)
			}
		case "q":
			if utf8.RuneCountInString(value) > 200 {
				return q, invalid("搜索词最多 200 字符")
			}
			q.search = value
		case "sort":
			q.sort = value
		case "order":
			if !member(value, "asc", "desc") {
				return q, invalid("order 须为 asc 或 desc")
			}
			q.order = value
		default:
			q.filters[key] = value
		}
	}
	if q.page.PageSize > 100 || int64(q.page.Page-1)*int64(q.page.PageSize) > math.MaxInt32 {
		return q, invalid("分页大小或偏移量超出范围")
	}
	sorts := []string{"id", "created_at", "updated_at"}
	if kind == "hunt" {
		sorts = append(sorts, "start_date", "end_date", "encounter_count", "duration_seconds")
	} else {
		sorts = append(sorts, "obtained_date", "level")
	}
	if !member(q.sort, sorts...) {
		return q, invalid("sort 字段不受支持")
	}
	for key, value := range q.filters {
		if _, ok := ids[key]; ok {
			n, e := strconv.ParseUint(value, 10, 63)
			if e != nil || n == 0 {
				return q, invalid("筛选 ID 须为正整数")
			}
			continue
		}
		if _, ok := dates[key]; ok {
			if _, _, e := DateBounds(value); e != nil {
				return q, e
			}
			continue
		}
		switch key {
		case "status":
			if !member(value, "in_progress", "completed", "abandoned") {
				return q, invalid("status 无效")
			}
		case "source":
			if !member(value, "self_hunted", "trade", "event", "other", "unknown") {
				return q, invalid("source 无效")
			}
		case "holding_status":
			if !member(value, "held", "traded", "released", "unknown") {
				return q, invalid("holding_status 无效")
			}
		case "has_hunt", "has_results":
			if !member(value, "true", "false") {
				return q, invalid("关联筛选须为 true 或 false")
			}
		case "min_count", "max_count", "min_duration", "max_duration":
			if _, e := strconv.ParseUint(value, 10, 63); e != nil {
				return q, invalid("累计值筛选须为非负整数")
			}
		default:
			max := 128
			if key == "count_unit" {
				max = 32
			}
			if key == "current_location" {
				max = 256
			}
			if utf8.RuneCountInString(value) > max {
				return q, invalid("筛选文本过长")
			}
		}
	}
	for _, prefix := range []string{"start", "end", "obtained"} {
		a, aok := q.filters[prefix+"_from"]
		b, bok := q.filters[prefix+"_to"]
		if aok && bok {
			lo, _, _ := DateBounds(a)
			_, hi, _ := DateBounds(b)
			if lo.After(hi) {
				return q, invalid("日期筛选范围倒置")
			}
		}
	}
	for _, suffix := range []string{"count", "duration"} {
		a, aok := q.filters["min_"+suffix]
		b, bok := q.filters["max_"+suffix]
		if aok && bok {
			lo, _ := strconv.ParseUint(a, 10, 63)
			hi, _ := strconv.ParseUint(b, 10, 63)
			if lo > hi {
				return q, invalid("累计值筛选范围倒置")
			}
		}
	}
	if kind == "hunt" && (q.sort == "encounter_count" || q.filters["min_count"] != "" || q.filters["max_count"] != "") && q.filters["count_unit"] == "" {
		return q, invalid("次数比较和排序须指定同一 count_unit")
	}
	return q, nil
}
func literalLike(value string) string {
	return "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(value) + "%"
}
func nameSearch(table, foreign string, value string) (string, []any) {
	terms := []string{}
	args := []any{}
	for _, col := range []string{"slug", "name", "name_zh"} {
		terms = append(terms, table+"."+col+` LIKE ? ESCAPE '\'`)
		args = append(args, value)
	}
	return "EXISTS (SELECT 1 FROM " + table + " WHERE " + table + ".id = " + foreign + " AND (" + strings.Join(terms, " OR ") + "))", args
}
func (q listQuery) apply(db *gorm.DB, kind string) *gorm.DB {
	ids, dates := queryMaps(kind)
	table := q.table
	for key, value := range q.filters {
		if col, ok := ids[key]; ok {
			db = db.Where(table+"."+col+" = ?", value)
			continue
		}
		if col, ok := dates[key]; ok {
			column := table + "." + col
			if strings.HasSuffix(key, "_from") {
				lo, _, _ := DateBounds(value)
				// Padding a partial month with day 31 compares identically to its real
				// last day against valid dates, without overflowing year 9999 in SQLite.
				upper := "CASE length(" + column + ") WHEN 4 THEN " + column + " || '-12-31' WHEN 7 THEN " + column + " || '-31' ELSE " + column + " END"
				db = db.Where("("+upper+") >= ?", lo.Format("2006-01-02"))
			} else {
				_, hi, _ := DateBounds(value)
				lower := "CASE length(" + column + ") WHEN 4 THEN " + column + " || '-01-01' WHEN 7 THEN " + column + " || '-01' ELSE " + column + " END"
				db = db.Where("("+lower+") <= ?", hi.Format("2006-01-02"))
			}
			continue
		}
		switch key {
		case "has_hunt":
			if value == "true" {
				db = db.Where("shiny_records.hunt_id IS NOT NULL")
			} else {
				db = db.Where("shiny_records.hunt_id IS NULL")
			}
		case "has_results":
			expr := "EXISTS (SELECT 1 FROM shiny_records WHERE shiny_records.hunt_id = hunts.id AND shiny_records.deleted_at IS NULL)"
			if value == "false" {
				expr = "NOT " + expr
			}
			db = db.Where(expr)
		case "min_count", "max_count", "min_duration", "max_duration":
			col := "encounter_count"
			if strings.HasSuffix(key, "duration") {
				col = "duration_seconds"
			}
			op := ">="
			if strings.HasPrefix(key, "max_") {
				op = "<="
			}
			n, _ := strconv.ParseUint(value, 10, 63)
			db = db.Where(table+"."+col+" "+op+" ?", n)
		default:
			db = db.Where(table+"."+key+" = ?", value)
		}
	}
	if q.search != "" {
		value := literalLike(q.search)
		terms := []string{}
		args := []any{}
		columns := []string{"nickname", "notes", "current_location"}
		foreigns := map[string][]string{"species": {"original_species_id", "current_species_id"}, "forms": {"original_form_id", "current_form_id"}, "games": {"original_game_id", "current_game_id"}}
		if kind == "hunt" {
			columns = []string{"method", "notes", "count_unit"}
			foreigns = map[string][]string{"species": {"species_id"}, "forms": {"form_id"}, "games": {"game_id"}}
		}
		for _, col := range columns {
			terms = append(terms, table+"."+col+` LIKE ? ESCAPE '\'`)
			args = append(args, value)
		}
		for ref, cols := range foreigns {
			for _, col := range cols {
				term, bind := nameSearch(ref, table+"."+col, value)
				terms = append(terms, term)
				args = append(args, bind...)
			}
		}
		db = db.Where("("+strings.Join(terms, " OR ")+")", args...)
	}
	return db
}
func list[T any](ctx context.Context, db *gorm.DB, raw, kind string) ([]T, Page, error) {
	rows := []T{}
	q, err := parseList(raw, kind)
	if err != nil {
		return rows, q.page, err
	}
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := q.apply(tx.Model(new(T)), kind)
		if err := query.Count(&q.page.Total).Error; err != nil {
			return err
		}
		column := q.table + "." + q.sort
		query = query.Order(column + " IS NULL ASC").Order(column + " " + q.order)
		if q.sort != "id" {
			query = query.Order(q.table + ".id " + q.order)
		}
		return query.Limit(q.page.PageSize).Offset((q.page.Page - 1) * q.page.PageSize).Find(&rows).Error
	})
	return rows, q.page, err
}
func (s Hunts) List(ctx context.Context, raw string) ([]model.Hunt, Page, error) {
	return list[model.Hunt](ctx, s.DB, raw, "hunt")
}
func (s Shinies) List(ctx context.Context, raw string) ([]model.ShinyRecord, Page, error) {
	return list[model.ShinyRecord](ctx, s.DB, raw, "shiny")
}
