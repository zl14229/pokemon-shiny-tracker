package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"github.com/zl14229/pokemon-shiny-tracker/internal/reference"
	"gorm.io/gorm"
)

type DexQuery struct {
	Dimension string `json:"dimension"`
	Basis     string `json:"basis"`
	Scope     string `json:"scope"`
	GameID    uint64 `json:"game_id,omitempty"`
	System    string `json:"collection_system"`
	Display   string `json:"display"`
	Acquired  string `json:"acquired"`
	Holding   string `json:"holding"`
	Search    string `json:"q,omitempty"`
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
}

func parseDex(raw string) (DexQuery, error) {
	q := DexQuery{Dimension: "species", Basis: "historical", Scope: "all", Display: "all", Acquired: "all", Holding: "all", Page: 1, PageSize: 50}
	values, e := url.ParseQuery(raw)
	if e != nil {
		return q, invalid("图鉴查询编码无效")
	}
	for key, vs := range values {
		if len(vs) != 1 || vs[0] == "" || !utf8.ValidString(vs[0]) {
			return q, invalid("图鉴查询字段为空或重复")
		}
		v := vs[0]
		switch key {
		case "dimension":
			q.Dimension = v
		case "basis":
			q.Basis = v
		case "scope":
			q.Scope = v
		case "collection_system":
			if len(v) > 64 {
				return q, invalid("收藏体系过长")
			}
			q.System = v
		case "display":
			q.Display = v
		case "acquired":
			q.Acquired = v
		case "holding":
			q.Holding = v
		case "q":
			if utf8.RuneCountInString(v) > 100 {
				return q, invalid("搜索词过长")
			}
			q.Search = strings.ToLower(v)
		case "game_id":
			n, e := strconv.ParseUint(v, 10, 63)
			if e != nil || n == 0 {
				return q, invalid("game_id 不合法")
			}
			q.GameID = n
		case "page", "page_size":
			n, e := strconv.ParseUint(v, 10, 31)
			if e != nil || n == 0 {
				return q, invalid("分页不合法")
			}
			if key == "page" {
				q.Page = int(n)
			} else {
				q.PageSize = int(n)
			}
		default:
			return q, invalid("未知图鉴查询字段")
		}
	}
	if !member(q.Dimension, "species", "form") || !member(q.Basis, "historical", "current") || !member(q.Scope, "all", "mainline", "game") || !member(q.Display, "all", "legal") || !member(q.Acquired, "all", "obtained", "missing") || !member(q.Holding, "all", "held", "no_confirmed_held", "unknown") {
		return q, invalid("图鉴筛选值不合法")
	}
	if (q.Scope == "game") != (q.GameID != 0) || q.PageSize > 100 || int64(q.Page-1)*int64(q.PageSize) > 2147483647 {
		return q, invalid("指定游戏或分页不合法")
	}
	return q, nil
}

type DexTarget struct {
	ID                  uint64            `json:"id"`
	SpeciesID           uint64            `json:"species_id"`
	FormID              *uint64           `json:"form_id"`
	NationalDex         int               `json:"national_dex"`
	Slug                string            `json:"slug"`
	Name                string            `json:"name"`
	Names               map[string]string `json:"names"`
	NameStatus          string            `json:"name_status"`
	CrossCheckStatus    string            `json:"cross_check_status"`
	Inclusion           string            `json:"inclusion"`
	InclusionReason     string            `json:"inclusion_reason"`
	SourceURL           string            `json:"source_url"`
	CrossCheckURL       string            `json:"cross_check_url"`
	Revision            string            `json:"revision"`
	Availability        string            `json:"availability"`
	Eligible            bool              `json:"eligible"`
	HistoricalCount     int               `json:"historical_count"`
	HeldCount           int               `json:"held_count"`
	UnknownHoldingCount int               `json:"unknown_holding_count"`
	Obtained            bool              `json:"obtained"`
	RecordIDs           []uint64          `json:"record_ids"`
}
type DexSummary struct {
	KnownTargets               int      `json:"known_targets"`
	EligibleTargets            int      `json:"eligible_targets"`
	ObtainedEligible           int      `json:"obtained_eligible"`
	MissingEligible            int      `json:"missing_eligible"`
	ObservedTargets            int      `json:"observed_targets"`
	ObservedOutsideDenominator int      `json:"observed_outside_denominator"`
	Percentage                 *float64 `json:"percentage"`
	DenominatorRule            string   `json:"denominator_rule"`
}
type DexCoverage struct {
	Status                     string   `json:"status"`
	Species                    int      `json:"species"`
	Forms                      int      `json:"forms"`
	IncludedForms              int      `json:"included_forms"`
	UnknownForms               int      `json:"unknown_forms"`
	SelectedGames              int      `json:"selected_games"`
	UnclassifiedGames          int      `json:"unclassified_games"`
	VerifiedAvailability       int      `json:"verified_availability"`
	UnknownAvailabilityTargets int      `json:"unknown_availability_targets"`
	PendingChineseNames        int      `json:"pending_chinese_names"`
	Revisions                  []string `json:"revisions"`
	AvailabilityVersions       []string `json:"availability_versions"`
	Note                       string   `json:"note"`
}
type DexResult struct {
	Data     []DexTarget `json:"data"`
	Meta     Page        `json:"meta"`
	Filters  DexQuery    `json:"filters"`
	Summary  DexSummary  `json:"summary"`
	Coverage DexCoverage `json:"coverage"`
}
type Dex struct{ DB *gorm.DB }
type proofSet map[uint64]map[uint64]bool

func (p proofSet) add(target, id uint64) {
	if p[target] == nil {
		p[target] = map[uint64]bool{}
	}
	p[target][id] = true
}
func (s Dex) List(ctx context.Context, raw string) (DexResult, error) {
	q, e := parseDex(raw)
	result := DexResult{Data: []DexTarget{}, Filters: q, Meta: Page{Page: q.Page, PageSize: q.PageSize}, Coverage: DexCoverage{Status: "partial", Revisions: []string{}, AvailabilityVersions: []string{}, Note: "仅限本地已核实子集；中文、持久形态分类和游戏合法性尚未完整核验。历史按原始获得游戏，当前按保存游戏；Sleep等收藏体系分别统计。"}}
	if e != nil {
		return result, e
	}
	e = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var species []model.Species
		var forms []model.Form
		var games []model.Game
		var entries []model.CatalogEntry
		var availability []model.ShinyAvailability
		var records []model.ShinyRecord
		for _, v := range []any{&species, &forms, &games, &entries, &availability, &records} {
			if e := tx.Find(v).Error; e != nil {
				return e
			}
		}
		speciesMap := map[uint64]model.Species{}
		formMap := map[uint64]model.Form{}
		gameMap := map[uint64]model.Game{}
		for _, v := range species {
			speciesMap[v.ID] = v
		}
		for _, v := range forms {
			formMap[v.ID] = v
		}
		for _, v := range games {
			gameMap[v.ID] = v
			if v.Kind == "unknown" {
				result.Coverage.UnclassifiedGames++
			}
		}
		if q.GameID != 0 {
			g, ok := gameMap[q.GameID]
			if !ok {
				return invalid("指定游戏不存在")
			}
			if q.System == "" {
				q.System = g.CollectionSystem
			} else if q.System != g.CollectionSystem {
				return invalid("游戏与收藏体系不匹配")
			}
		}
		if q.System == "" {
			q.System = "pokemon"
		}
		result.Filters = q
		selected := map[uint64]bool{}
		for _, g := range games {
			if g.CollectionSystem == q.System && (q.Scope == "all" || q.Scope == "mainline" && g.Kind == "mainline" || q.Scope == "game" && g.ID == q.GameID) {
				selected[g.ID] = true
			}
		}
		availabilityGameCount := 0
		for id := range selected {
			if gameMap[id].Kind != "storage" {
				availabilityGameCount++
			}
		}
		result.Coverage.SelectedGames = len(selected)
		result.Coverage.Species = len(species)
		result.Coverage.Forms = len(forms)
		// A missing original game is never replaced with a current game for a game filter.
		// Current game can only classify an otherwise unknown collection system in all scope.
		inScope := func(r model.ShinyRecord, historical bool) bool {
			game := r.CurrentGameID
			if historical {
				game = r.OriginalGameID
			}
			if game != nil {
				return selected[*game]
			}
			if q.Scope != "all" {
				return false
			}
			otherGame := r.OriginalGameID
			if historical {
				otherGame = r.CurrentGameID
			}
			if otherGame != nil {
				return gameMap[*otherGame].CollectionSystem == q.System
			}
			return q.System == "pokemon"
		}
		historyProof, heldProof, unknownProof := proofSet{}, proofSet{}, proofSet{}
		add := func(p proofSet, recordID, sid uint64, fid *uint64) {
			if _, ok := speciesMap[sid]; !ok {
				return
			}
			if q.Dimension == "species" {
				p.add(sid, recordID)
			} else if fid != nil {
				if f, ok := formMap[*fid]; ok && f.SpeciesID == sid {
					p.add(*fid, recordID)
				}
			}
		}
		active := map[uint64]model.ShinyRecord{}

		for _, r := range records {
			active[r.ID] = r

			if inScope(r, true) {
				add(historyProof, r.ID, r.CurrentSpeciesID, r.CurrentFormID)
				if r.OriginalSpeciesID != nil {
					add(historyProof, r.ID, *r.OriginalSpeciesID, r.OriginalFormID)
				}
			}
			if inScope(r, false) {
				if r.HoldingStatus == "held" {
					add(heldProof, r.ID, r.CurrentSpeciesID, r.CurrentFormID)
				} else if r.HoldingStatus == "unknown" {
					add(unknownProof, r.ID, r.CurrentSpeciesID, r.CurrentFormID)
				}
			}
		}
		// Join avoids a SQLite variable limit for large personal collections.
		var histories []model.ShinyHistory
		if len(records) > 0 {
			if e := tx.Joins("JOIN shiny_records ON shiny_records.id = shiny_histories.shiny_record_id AND shiny_records.deleted_at IS NULL").Where("shiny_histories.kind IN ?", []string{"evolution", "form_change"}).Find(&histories).Error; e != nil {
				return e
			}
		}
		for _, h := range histories {
			root, ok := active[h.ShinyRecordID]
			if !ok || !inScope(root, true) {
				continue
			}
			for _, raw := range []string{h.BeforeJSON, h.AfterJSON} {
				var state model.ShinyRecord
				if e := json.Unmarshal([]byte(raw), &state); e != nil {
					return fmt.Errorf("invalid shiny history snapshot: %w", e)
				}
				if state.ID != root.ID {
					continue
				}
				add(historyProof, root.ID, state.CurrentSpeciesID, state.CurrentFormID)
			}
		}
		details := map[string]model.CatalogEntry{}
		versions := map[string]bool{}
		for _, v := range entries {
			key := fmt.Sprintf("species:%d", v.SpeciesID)
			if v.FormID != nil {
				key = fmt.Sprintf("form:%d", *v.FormID)
			}
			details[key] = v
			versions[v.SourceRevision] = true
			if v.Kind == "form" {
				if v.Inclusion == "included" {
					result.Coverage.IncludedForms++
				} else if v.Inclusion == "unknown" {
					result.Coverage.UnknownForms++
				}
			}
		}
		for _, f := range forms {
			if _, ok := details[fmt.Sprintf("form:%d", f.ID)]; !ok {
				result.Coverage.UnknownForms++
			}
		}
		for v := range versions {
			result.Coverage.Revisions = append(result.Coverage.Revisions, v)
		}
		sort.Strings(result.Coverage.Revisions)
		type availableForm struct {
			available                  bool
			lockedGames, transferGames map[uint64]bool
		}
		statusByForm := map[uint64]*availableForm{}
		availabilityVersions := map[string]bool{}
		for _, a := range availability {
			g := gameMap[a.GameID]
			if !selected[a.GameID] || g.Kind == "storage" || a.VerificationStatus != "verified" || a.CheckedAt == nil || a.CheckedAt.IsZero() || a.Source == "" || !validEvidenceURL(a.SourceURL) || a.DataVersion == "" {
				continue
			}
			result.Coverage.VerifiedAvailability++
			availabilityVersions[a.DataVersion] = true
			state := statusByForm[a.FormID]
			if state == nil {
				state = &availableForm{lockedGames: map[uint64]bool{}, transferGames: map[uint64]bool{}}
				statusByForm[a.FormID] = state
			}
			switch a.Status {
			case "confirmed_available":
				state.available = true
			case "transfer_only":
				state.transferGames[a.GameID] = true
			case "confirmed_locked":
				state.lockedGames[a.GameID] = true
			}
		}
		for v := range availabilityVersions {
			result.Coverage.AvailabilityVersions = append(result.Coverage.AvailabilityVersions, v)
		}
		sort.Strings(result.Coverage.AvailabilityVersions)
		availabilityFor := func(formID uint64) (string, bool) {
			state := statusByForm[formID]
			if state == nil {
				return "unknown", false
			}
			if state.available {
				return "confirmed_available", true
			}
			if availabilityGameCount > 0 && len(state.lockedGames)+len(state.transferGames) == availabilityGameCount {
				if len(state.transferGames) > 0 {
					return "transfer_only", false
				}
				return "confirmed_locked", false
			}
			return "unknown", false
		}
		eligibleSpecies := map[uint64]bool{}
		speciesStatuses := map[uint64]string{}
		includedBySpecies, lockedBySpecies, transferBySpecies := map[uint64]int{}, map[uint64]int{}, map[uint64]int{}
		for _, f := range forms {
			status, eligible := availabilityFor(f.ID)
			if details[fmt.Sprintf("form:%d", f.ID)].Inclusion != "included" {
				continue
			}
			includedBySpecies[f.SpeciesID]++
			if status == "confirmed_locked" {
				lockedBySpecies[f.SpeciesID]++
			}
			if eligible {
				eligibleSpecies[f.SpeciesID] = true
			}
			if status == "confirmed_available" {
				speciesStatuses[f.SpeciesID] = status
			} else if status == "transfer_only" {
				transferBySpecies[f.SpeciesID]++
			}
		}
		for sid, count := range includedBySpecies {
			if count > 0 && lockedBySpecies[sid]+transferBySpecies[sid] == count {
				speciesStatuses[sid] = "confirmed_locked"
				if transferBySpecies[sid] > 0 {
					speciesStatuses[sid] = "transfer_only"
				}
			}
		}
		targets := []DexTarget{}
		addTarget := func(id, sid uint64, fid *uint64, labels model.Labels) {
			kind := "species"
			if fid != nil {
				kind = "form"
			}
			detail, hasDetail := details[fmt.Sprintf("%s:%d", kind, id)]
			t := DexTarget{ID: id, SpeciesID: sid, FormID: fid, NationalDex: speciesMap[sid].NationalDex, Slug: labels.Slug, Name: reference.DisplayCatalogName(map[string]string{"en": labels.Name}, "unknown", labels.Slug), Names: map[string]string{}, NameStatus: "unknown", CrossCheckStatus: "unknown", Inclusion: "unknown", InclusionReason: "unreviewed-reference", Availability: "unknown", RecordIDs: []uint64{}}
			if fid == nil {
				t.Inclusion = "included"
				t.InclusionReason = "national-species"
			}
			if hasDetail {
				_ = json.Unmarshal([]byte(detail.NamesJSON), &t.Names)
				t.NameStatus = detail.NameStatus
				t.CrossCheckStatus = detail.CrossCheckStatus
				if t.CrossCheckStatus == "conflict" {
					t.NameStatus = "conflict"
				}
				t.Inclusion = detail.Inclusion
				t.InclusionReason = detail.Reason
				t.Name = reference.DisplayCatalogName(t.Names, t.NameStatus, labels.Slug)
				t.SourceURL = detail.SourceURL
				t.CrossCheckURL = detail.CrossCheckURL
				t.Revision = detail.SourceRevision
			}
			if fid == nil {
				t.Eligible = eligibleSpecies[sid]
				if value := speciesStatuses[sid]; value != "" {
					t.Availability = value
				}
			} else {
				var eligible bool
				t.Availability, eligible = availabilityFor(*fid)
				t.Eligible = eligible && t.Inclusion == "included"
			}
			if t.CrossCheckStatus != "verified" || t.NameStatus != "verified" {
				result.Coverage.PendingChineseNames++
			}
			if t.Availability == "unknown" {
				result.Coverage.UnknownAvailabilityTargets++
			}
			t.HistoricalCount = len(historyProof[id])
			t.HeldCount = len(heldProof[id])
			t.UnknownHoldingCount = len(unknownProof[id])
			t.Obtained = t.HistoricalCount > 0
			if q.Basis == "current" {
				t.Obtained = t.HeldCount > 0
			}
			proof := historyProof[id]
			if q.Basis == "current" {
				proof = heldProof[id]
			}
			for recordID := range proof {
				t.RecordIDs = append(t.RecordIDs, recordID)
			}
			sort.Slice(t.RecordIDs, func(i, j int) bool { return t.RecordIDs[i] < t.RecordIDs[j] })
			targets = append(targets, t)
		}
		if q.Dimension == "species" {
			for _, v := range species {
				addTarget(v.ID, v.ID, nil, v.Labels)
			}
		} else {
			for _, v := range forms {
				id := v.ID
				addTarget(v.ID, v.SpeciesID, &id, v.Labels)
			}
		}
		result.Summary = DexSummary{KnownTargets: len(targets), DenominatorRule: "v1: scoped verified confirmed_available + included persistent form; species deduplicated; unknown/conflict/locked/transfer_only excluded; numerator intersects denominator"}
		filtered := []DexTarget{}
		for _, t := range targets {
			if t.Eligible {
				result.Summary.EligibleTargets++
				if t.Obtained {
					result.Summary.ObtainedEligible++
				}
			}
			if t.Obtained {
				result.Summary.ObservedTargets++
				if !t.Eligible {
					result.Summary.ObservedOutsideDenominator++
				}
			}
			if q.Display == "legal" && !t.Eligible || q.Acquired == "obtained" && !t.Obtained || q.Acquired == "missing" && t.Obtained || q.Holding == "held" && t.HeldCount == 0 || q.Holding == "no_confirmed_held" && t.HeldCount > 0 || q.Holding == "unknown" && t.UnknownHoldingCount == 0 {
				continue
			}
			if q.Search != "" {
				hay := strings.ToLower(t.Name + " " + t.Slug + " " + strconv.Itoa(t.NationalDex))
				for _, name := range t.Names {
					hay += " " + strings.ToLower(name)
				}
				if !strings.Contains(hay, q.Search) {
					continue
				}
			}
			filtered = append(filtered, t)
		}
		result.Summary.MissingEligible = result.Summary.EligibleTargets - result.Summary.ObtainedEligible
		if result.Summary.EligibleTargets > 0 {
			percent := 100 * float64(result.Summary.ObtainedEligible) / float64(result.Summary.EligibleTargets)
			result.Summary.Percentage = &percent
		}
		sort.Slice(filtered, func(i, j int) bool {
			if filtered[i].NationalDex != filtered[j].NationalDex {
				return filtered[i].NationalDex < filtered[j].NationalDex
			}
			return filtered[i].ID < filtered[j].ID
		})
		result.Meta.Total = int64(len(filtered))
		offset := (q.Page - 1) * q.PageSize
		if offset < len(filtered) {
			end := offset + q.PageSize
			if end > len(filtered) {
				end = len(filtered)
			}
			result.Data = filtered[offset:end]
		}
		return nil
	})
	return result, e
}

func validEvidenceURL(value string) bool {
	u, e := url.Parse(value)
	return e == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Hostname() != "" && u.User == nil
}
