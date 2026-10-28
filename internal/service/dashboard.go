package service

import (
	"context"
	"net/url"
	"unicode/utf8"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"gorm.io/gorm"
)

// analyticsSnapshot is shared by read-only summaries; no history join can multiply identities.
type analyticsSnapshot struct {
	Records []model.ShinyRecord
	Hunts   []model.Hunt
	Games   map[uint64]model.Game
}

func readAnalytics(tx *gorm.DB) (analyticsSnapshot, error) {
	v := analyticsSnapshot{Games: map[uint64]model.Game{}}
	var games []model.Game
	for _, rows := range []any{&v.Records, &v.Hunts, &games} {
		if err := tx.Order("id ASC").Find(rows).Error; err != nil {
			return v, err
		}
	}
	for _, g := range games {
		v.Games[g.ID] = g
	}
	return v, nil
}

func recordInDexScope(r model.ShinyRecord, historical bool, q DexQuery, selected map[uint64]bool, games map[uint64]model.Game) bool {
	game, fallback := r.CurrentGameID, r.OriginalGameID
	if historical {
		game, fallback = r.OriginalGameID, r.CurrentGameID
	}
	if game != nil {
		return selected[*game]
	}
	if q.Scope != "all" {
		return false
	}
	if fallback != nil {
		return games[*fallback].CollectionSystem == q.System
	}
	return q.System == "pokemon"
}

type CollectionOverview struct {
	Historical          int `json:"historical_individuals"`
	Held                int `json:"held_individuals"`
	UnknownHolding      int `json:"unknown_holding_individuals"`
	SelfHunted          int `json:"self_hunted_individuals"`
	UnknownSource       int `json:"unknown_source_individuals"`
	UnknownOriginalGame int `json:"unknown_original_game_individuals"`
}
type HuntOverview struct {
	Total           int `json:"total"`
	InProgress      int `json:"in_progress"`
	Completed       int `json:"completed"`
	Abandoned       int `json:"abandoned"`
	UnknownGame     int `json:"unknown_game"`
	UnknownCount    int `json:"unknown_count"`
	UnknownDuration int `json:"unknown_duration"`
}
type DashboardDex struct {
	Dimension string      `json:"dimension"`
	Basis     string      `json:"basis"`
	Summary   DexSummary  `json:"summary"`
	Coverage  DexCoverage `json:"coverage"`
}
type DashboardResult struct {
	System     string             `json:"collection_system"`
	Collection CollectionOverview `json:"collection"`
	Hunts      HuntOverview       `json:"hunts"`
	Dex        []DashboardDex     `json:"dex"`
	Rule       string             `json:"rule"`
}
type Analytics struct{ DB *gorm.DB }

func huntInCollectionSystem(h model.Hunt, system string, games map[uint64]model.Game) bool {
	if h.GameID == nil {
		return system == "pokemon"
	}
	return games[*h.GameID].CollectionSystem == system
}

func dashboardSystem(raw string) (string, error) {
	q, err := url.ParseQuery(raw)
	if err != nil {
		return "", invalid("总览查询编码无效")
	}
	system := "pokemon"
	for k, vs := range q {
		if k != "collection_system" || len(vs) != 1 || vs[0] == "" || len(vs[0]) > 64 || !utf8.ValidString(vs[0]) {
			return "", invalid("总览仅接受单个收藏体系")
		}
		system = vs[0]
	}
	return system, nil
}

func (s Analytics) Dashboard(ctx context.Context, raw string) (DashboardResult, error) {
	system, err := dashboardSystem(raw)
	v := DashboardResult{System: system, Dex: []DashboardDex{}, Rule: "永久个体按ID计数，软删除排除；历史依原始游戏、当前依保存游戏，未知游戏仅在全范围回退体系。进化或转移不新增个体。次数与耗时未知不补0；图鉴复用M4引擎，分母仅本地已核实合法子集。"}
	if err != nil {
		return v, err
	}
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		snapshot, err := readAnalytics(tx)
		if err != nil {
			return err
		}
		selected := map[uint64]bool{}
		for id, game := range snapshot.Games {
			if game.CollectionSystem == system {
				selected[id] = true
			}
		}
		q := DexQuery{Scope: "all", System: system}
		for _, r := range snapshot.Records {
			if recordInDexScope(r, true, q, selected, snapshot.Games) {
				v.Collection.Historical++
				if r.Source == "self_hunted" {
					v.Collection.SelfHunted++
				}
				if r.Source == "unknown" {
					v.Collection.UnknownSource++
				}
				if r.OriginalGameID == nil {
					v.Collection.UnknownOriginalGame++
				}
			}
			if recordInDexScope(r, false, q, selected, snapshot.Games) {
				if r.HoldingStatus == "held" {
					v.Collection.Held++
				}
				if r.HoldingStatus == "unknown" {
					v.Collection.UnknownHolding++
				}
			}
		}
		for _, h := range snapshot.Hunts {
			if !huntInCollectionSystem(h, system, snapshot.Games) {
				continue
			}
			v.Hunts.Total++
			switch h.Status {
			case "in_progress":
				v.Hunts.InProgress++
			case "completed":
				v.Hunts.Completed++
			case "abandoned":
				v.Hunts.Abandoned++
			}
			if h.GameID == nil {
				v.Hunts.UnknownGame++
			}
			if h.EncounterCount == nil {
				v.Hunts.UnknownCount++
			}
			if h.DurationSeconds == nil {
				v.Hunts.UnknownDuration++
			}
		}
		for _, dimension := range []string{"species", "form"} {
			for _, basis := range []string{"historical", "current"} {
				d, err := (Dex{DB: tx}).List(ctx, "dimension="+dimension+"&basis="+basis+"&collection_system="+url.QueryEscape(system)+"&page_size=1")
				if err != nil {
					return err
				}
				v.Dex = append(v.Dex, DashboardDex{Dimension: dimension, Basis: basis, Summary: d.Summary, Coverage: d.Coverage})
			}
		}
		return nil
	})
	return v, err
}
