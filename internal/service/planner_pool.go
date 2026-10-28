package service

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"sort"
	"strconv"

	"github.com/zl14229/pokemon-shiny-tracker/internal/huntdata"
	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"github.com/zl14229/pokemon-shiny-tracker/internal/reference"
	"gorm.io/gorm"
)

type PlannerRules struct {
	GameIDs       []uint64 `json:"game_ids"`
	Dimension     string   `json:"dimension"`
	Pool          string   `json:"pool"`
	Count         int      `json:"count"`
	ExcludeActive *bool    `json:"exclude_active"`
}

func (r *PlannerRules) normalize() error {
	if len(r.GameIDs) < 1 || len(r.GameIDs) > 20 || r.Count < 1 || r.Count > 100 {
		return invalid("选择1至20个版本，数量须为1至100")
	}
	if r.Dimension == "" {
		r.Dimension = "species"
	}
	if r.Pool == "" {
		r.Pool = "all_missing"
	}
	if r.ExcludeActive == nil {
		v := true
		r.ExcludeActive = &v
	}
	if !member(r.Dimension, "species", "form") || !member(r.Pool, "all_missing", "selected_missing") {
		return invalid("候选池或维度无效")
	}
	ids := append([]uint64{}, r.GameIDs...)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for i, id := range ids {
		if !validID(id) || i > 0 && ids[i-1] == id {
			return invalid("版本ID重复或无效")
		}
	}
	r.GameIDs = ids
	return nil
}

type PlannerChoice struct {
	GameID   uint64         `json:"game_id"`
	GameName string         `json:"game_name"`
	Evidence []huntdata.Row `json:"evidence"`
}
type PlannerCandidate struct {
	Key        string          `json:"key"`
	SpeciesID  uint64          `json:"species_id"`
	FormID     *uint64         `json:"form_id"`
	Name       string          `json:"name"`
	NameStatus string          `json:"name_status"`
	Games      []PlannerChoice `json:"games"`
}
type PlannerGameCoverage struct {
	GameID           uint64 `json:"game_id"`
	Name             string `json:"name"`
	ReviewedEntries  int    `json:"reviewed_entries"`
	VerifiedDirect   int    `json:"verified_direct"`
	AvailableTargets int    `json:"available_targets"`
}
type PlannerCoverage struct {
	Status           string                `json:"status"`
	Note             string                `json:"note"`
	Games            []PlannerGameCoverage `json:"games"`
	Versions         []string              `json:"versions"`
	VerifiedTargets  int                   `json:"verified_targets"`
	ExcludedObtained int                   `json:"excluded_obtained"`
	ExcludedActive   int                   `json:"excluded_active"`
	Available        int                   `json:"available"`
	Requested        int                   `json:"requested"`
	Actual           int                   `json:"actual"`
	Shortage         bool                  `json:"shortage"`
}
type PlannerPool struct {
	Rules      PlannerRules       `json:"rules"`
	Coverage   PlannerCoverage    `json:"coverage"`
	Candidates []PlannerCandidate `json:"candidates"`
}
type Planner struct {
	DB          *gorm.DB
	RandomIndex func(int) (int, error)
}

func plannerJSON(v any) string { b, _ := json.Marshal(v); return string(b) }
func (s Planner) random(n int) (int, error) {
	if n < 1 {
		return 0, invalid("没有候选目标")
	}
	var v int
	var e error
	if s.RandomIndex != nil {
		v, e = s.RandomIndex(n)
	} else {
		var b *big.Int
		b, e = cryptorand.Int(cryptorand.Reader, big.NewInt(int64(n)))
		if e == nil {
			v = int(b.Int64())
		}
	}
	if e != nil {
		return 0, e
	}
	if v < 0 || v >= n {
		return 0, invalid("随机索引无效")
	}
	return v, nil
}
func (s Planner) Pool(ctx context.Context, r PlannerRules) (PlannerPool, error) {
	var result PlannerPool
	if e := r.normalize(); e != nil {
		return result, e
	}
	e := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error { var e error; result, e = s.pool(ctx, tx, r); return e })
	return result, e
}

// The dex is the single historical-proof engine: evolutions, form changes,
// missing original games, soft deletes and separate collection systems agree.
func plannerObtained(ctx context.Context, tx *gorm.DB, r PlannerRules, system string) (map[uint64]bool, error) {
	found := map[uint64]bool{}
	scopes := []uint64{0}
	if r.Pool == "selected_missing" {
		scopes = r.GameIDs
	}
	for _, gameID := range scopes {
		q := url.Values{"dimension": {r.Dimension}, "basis": {"historical"}, "scope": {"all"}, "collection_system": {system}, "acquired": {"obtained"}, "page_size": {"100"}}
		if gameID != 0 {
			q.Set("scope", "game")
			q.Set("game_id", strconv.FormatUint(gameID, 10))
		}
		for page := 1; ; page++ {
			q.Set("page", strconv.Itoa(page))
			out, e := (Dex{DB: tx}).List(ctx, q.Encode())
			if e != nil {
				return nil, e
			}
			for _, v := range out.Data {
				found[v.ID] = true
			}
			if page*100 >= int(out.Meta.Total) {
				break
			}
		}
	}
	return found, nil
}
func (s Planner) pool(ctx context.Context, tx *gorm.DB, r PlannerRules) (PlannerPool, error) {
	out := PlannerPool{Rules: r, Candidates: []PlannerCandidate{}, Coverage: PlannerCoverage{Status: "partial", Note: "仅从本地已核实、可直接合法刷闪的持久目标抽取；覆盖不完整。未核验、锁闪、仅传送不补入。概率缺失不代表锁闪，攻略可能待完善。", Games: []PlannerGameCoverage{}, Versions: []string{}, Requested: r.Count}}
	games := map[uint64]model.Game{}
	system := ""
	for _, id := range r.GameIDs {
		var g model.Game
		if e := tx.First(&g, id).Error; e != nil {
			return out, invalid("所选游戏不存在")
		}
		if g.Kind == "storage" {
			return out, invalid("保存位置不能作为抽取版本")
		}
		if system != "" && system != g.CollectionSystem {
			return out, invalid("不同收藏体系须分别生成")
		}
		system = g.CollectionSystem
		games[id] = g
		name := g.Name
		if g.NameZh != nil && g.NameStatus != "conflict" {
			name = *g.NameZh
		}
		out.Coverage.Games = append(out.Coverage.Games, PlannerGameCoverage{GameID: id, Name: name})
	}
	rows, e := huntdata.Read(tx)
	if e != nil {
		return out, &Error{"reference_unavailable", "本地刷闪资料的来源或映射不一致，请核对资料；未补入未知目标"}
	}
	var entries []model.CatalogEntry
	if e = tx.Find(&entries).Error; e != nil {
		return out, e
	}
	details := map[string]model.CatalogEntry{}
	for _, d := range entries {
		key := fmt.Sprintf("species:%d", d.SpeciesID)
		if d.FormID != nil {
			key = fmt.Sprintf("form:%d", *d.FormID)
		}
		details[key] = d
	}
	names := func(key string, slug string) (string, string) {
		d := details[key]
		var n map[string]string
		_ = json.Unmarshal([]byte(d.NamesJSON), &n)
		status := d.NameStatus
		if status == "" {
			status = "unknown"
		}
		if d.CrossCheckStatus == "conflict" {
			status = "conflict"
		}
		return reference.DisplayCatalogName(n, status, slug), status
	}
	byKey := map[string]*PlannerCandidate{}
	versions := map[string]bool{}
	for _, row := range rows {
		if _, ok := games[row.GameID]; !ok {
			continue
		}
		for i := range out.Coverage.Games {
			if out.Coverage.Games[i].GameID == row.GameID {
				out.Coverage.Games[i].ReviewedEntries++
			}
		}
		versions[row.BatchVersion] = true
		if !row.Fact.Huntable() || details[fmt.Sprintf("form:%d", row.FormID)].Inclusion != "included" {
			continue
		}
		for i := range out.Coverage.Games {
			if out.Coverage.Games[i].GameID == row.GameID {
				out.Coverage.Games[i].VerifiedDirect++
			}
		}
		key := fmt.Sprintf("species:%d", row.SpeciesID)
		var fid *uint64
		slug := ""
		if r.Dimension == "form" {
			key = fmt.Sprintf("form:%d", row.FormID)
			id := row.FormID
			fid = &id
			var f model.Form
			if e = tx.First(&f, row.FormID).Error; e != nil {
				return out, e
			}
			slug = f.Slug
		} else {
			var p model.Species
			if e = tx.First(&p, row.SpeciesID).Error; e != nil {
				return out, e
			}
			slug = p.Slug
		}
		c := byKey[key]
		if c == nil {
			name, status := names(key, slug)
			c = &PlannerCandidate{Key: key, SpeciesID: row.SpeciesID, FormID: fid, Name: name, NameStatus: status, Games: []PlannerChoice{}}
			byKey[key] = c
		}
		gi := -1
		for i, g := range c.Games {
			if g.GameID == row.GameID {
				gi = i
				break
			}
		}
		if gi < 0 {
			var name string
			for _, g := range out.Coverage.Games {
				if g.GameID == row.GameID {
					name = g.Name
				}
			}
			c.Games = append(c.Games, PlannerChoice{GameID: row.GameID, GameName: name, Evidence: []huntdata.Row{}})
			gi = len(c.Games) - 1
		}
		c.Games[gi].Evidence = append(c.Games[gi].Evidence, row)
	}
	for v := range versions {
		out.Coverage.Versions = append(out.Coverage.Versions, v)
	}
	sort.Strings(out.Coverage.Versions)
	obtained, e := plannerObtained(ctx, tx, r, system)
	if e != nil {
		return out, e
	}
	var active []model.Hunt
	if *r.ExcludeActive {
		if e = tx.Where("status = ?", "in_progress").Find(&active).Error; e != nil {
			return out, e
		}
	}
	out.Coverage.VerifiedTargets = len(byKey)
	for _, c := range byKey {
		id := c.SpeciesID
		if c.FormID != nil {
			id = *c.FormID
		}
		if obtained[id] {
			out.Coverage.ExcludedObtained++
			continue
		}
		blocked := false
		for _, h := range active {
			if h.SpeciesID != c.SpeciesID {
				continue
			}
			if h.GameID != nil {
				var g model.Game
				if e = tx.First(&g, *h.GameID).Error; e != nil {
					return out, e
				}
				if g.CollectionSystem != system {
					continue
				}
			} else if system != "pokemon" {
				continue
			}
			if c.FormID == nil || h.FormID == nil || *c.FormID == *h.FormID {
				blocked = true
				break
			}
		}
		if blocked {
			out.Coverage.ExcludedActive++
			continue
		}
		sort.Slice(c.Games, func(i, j int) bool { return c.Games[i].GameID < c.Games[j].GameID })
		out.Candidates = append(out.Candidates, *c)
		for _, choice := range c.Games {
			for i := range out.Coverage.Games {
				if out.Coverage.Games[i].GameID == choice.GameID {
					out.Coverage.Games[i].AvailableTargets++
				}
			}
		}
	}
	sort.Slice(out.Candidates, func(i, j int) bool { return out.Candidates[i].Key < out.Candidates[j].Key })
	out.Coverage.Available = len(out.Candidates)
	out.Coverage.Actual = r.Count
	if out.Coverage.Actual > out.Coverage.Available {
		out.Coverage.Actual = out.Coverage.Available
	}
	out.Coverage.Shortage = out.Coverage.Actual < r.Count
	return out, nil
}

type PlannerPick struct {
	PlannerCandidate
	Choice PlannerChoice `json:"choice"`
}

// Fisher-Yates samples unique targets uniformly without replacement. Only after
// choosing a target do we choose uniformly among that target's eligible games.
func (s Planner) pick(pool PlannerPool) ([]PlannerPick, error) {
	candidates := append([]PlannerCandidate{}, pool.Candidates...)
	out := []PlannerPick{}
	for i := 0; i < pool.Coverage.Actual; i++ {
		j, e := s.random(len(candidates) - i)
		if e != nil {
			return nil, e
		}
		j += i
		candidates[i], candidates[j] = candidates[j], candidates[i]
		c := candidates[i]
		g, e := s.random(len(c.Games))
		if e != nil {
			return nil, e
		}
		out = append(out, PlannerPick{c, c.Games[g]})
	}
	return out, nil
}
