package service

import (
	"context"
	"encoding/json"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"gorm.io/gorm"
)

const maxChartInteger int64 = 9007199254740991

type statisticsQuery struct {
	filters        map[string]string
	hunts, records listQuery
	linkedOnly     bool
}

func parseStatistics(raw string) (statisticsQuery, error) {
	q := statisticsQuery{filters: map[string]string{"collection_system": "pokemon"}}
	values, e := url.ParseQuery(raw)
	if e != nil {
		return q, invalid("统计查询编码无效")
	}
	h, r := url.Values{}, url.Values{}
	for k, vs := range values {
		if len(vs) != 1 || strings.TrimSpace(vs[0]) == "" || !utf8.ValidString(vs[0]) {
			return q, invalid("统计字段为空或重复")
		}
		v := strings.TrimSpace(vs[0])
		q.filters[k] = v
		switch k {
		case "collection_system":
			if len(v) > 64 {
				return q, invalid("收藏体系过长")
			}
		case "game_id":
			h.Set(k, v)
			r.Set("original_game_id", v)
		case "source":
			r.Set(k, v)
		case "from", "to":
			if len(v) != 10 {
				return q, invalid("统计日期边界须精确到日")
			}
			suffix := "_from"
			if k == "to" {
				suffix = "_to"
			}
			h.Set("start"+suffix, v)
			r.Set("obtained"+suffix, v)
		case "method", "status", "count_unit", "min_count", "max_count", "min_duration", "max_duration":
			h.Set(k, v)
			q.linkedOnly = true
		default:
			return q, invalid("未知统计筛选字段")
		}
	}
	q.hunts, e = parseList(h.Encode(), "hunt")
	if e != nil {
		return q, e
	}
	q.records, e = parseList(r.Encode(), "shiny")
	return q, e
}

type StatisticsDistribution struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Hunts       int    `json:"hunts"`
	Individuals int    `json:"individuals"`
	SelfHunted  int    `json:"self_hunted"`
}
type StatisticsMonth struct {
	Month        string `json:"month"`
	StartedHunts int    `json:"started_hunts"`
	Individuals  int    `json:"individuals"`
	SelfHunted   int    `json:"self_hunted"`
}
type StatisticsCoverage struct {
	UnknownObtainedDate int `json:"unknown_obtained_date"`
	PartialObtainedDate int `json:"partial_obtained_date"`
	UnknownStartDate    int `json:"unknown_start_date"`
	PartialStartDate    int `json:"partial_start_date"`
	UnplacedRecords     int `json:"unplaced_records"`
	UnplacedHunts       int `json:"unplaced_hunts"`
	UnknownOriginalGame int `json:"unknown_original_game"`
	UnknownMethod       int `json:"unknown_method"`
}
type StatisticsEffort struct {
	Key                   string         `json:"key"`
	GameID                *uint64        `json:"game_id"`
	Game                  string         `json:"game"`
	Method                *string        `json:"method"`
	Unit                  *string        `json:"unit"`
	Hunts                 int            `json:"hunts"`
	KnownCountHunts       int            `json:"known_count_hunts"`
	KnownDurationHunts    int            `json:"known_duration_hunts"`
	Count                 *int64         `json:"count"`
	Seconds               *int64         `json:"seconds"`
	CountSamples          int            `json:"count_samples"`
	TimeSamples           int            `json:"time_samples"`
	CountOutcomes         int            `json:"count_outcomes"`
	TimeOutcomes          int            `json:"time_outcomes"`
	CountPerShiny         *float64       `json:"count_per_shiny"`
	SecondsPerShiny       *float64       `json:"seconds_per_shiny"`
	Overflow              bool           `json:"overflow"`
	Note                  string         `json:"note"`
	CountSampleStatus     string         `json:"count_sample_status"`
	TimeSampleStatus      string         `json:"time_sample_status"`
	MedianCountPerShiny   *float64       `json:"median_count_per_shiny"`
	MedianSecondsPerShiny *float64       `json:"median_seconds_per_shiny"`
	Exclusions            map[string]int `json:"exclusions"`
}
type StatisticsResult struct {
	Filters     map[string]string        `json:"filters"`
	Individuals int                      `json:"individuals"`
	Hunts       int                      `json:"hunts"`
	ByGame      []StatisticsDistribution `json:"by_game"`
	ByMethod    []StatisticsDistribution `json:"by_method"`
	BySource    []StatisticsDistribution `json:"by_source"`
	Trend       []StatisticsMonth        `json:"trend"`
	Effort      []StatisticsEffort       `json:"effort"`
	Coverage    StatisticsCoverage       `json:"coverage"`
	Rule        string                   `json:"rule"`
	Quality     StatisticsQuality        `json:"quality"`
}

// Nullable sums never turn absent input into zero, overflow, or an unsafe JS integer.
func chartSum(sum **int64, n *int64, overflow *bool) {
	if n == nil || *overflow {
		return
	}
	if *n < 0 || *n > maxChartInteger || (*sum != nil && **sum > maxChartInteger-*n) {
		*overflow = true
		*sum = nil
		return
	}
	if *sum == nil {
		v := int64(0)
		*sum = &v
	}
	**sum += *n
}

func statisticsMonth(date *string, filters map[string]string) string {
	if date == nil || len(*date) < 7 {
		return ""
	}
	lo, hi, e := DateBounds(*date)
	if e != nil {
		return ""
	}
	if v := filters["from"]; v != "" {
		start, _, _ := DateBounds(v)
		if lo.Before(start) {
			return ""
		}
	}
	if v := filters["to"]; v != "" {
		_, end, _ := DateBounds(v)
		if hi.After(end) {
			return ""
		}
	}
	return lo.Format("2006-01")
}
func gameLabel(id *uint64, games map[uint64]model.Game) (string, string) {
	if id == nil {
		return "unknown", "游戏未知"
	}
	g := games[*id]
	return strconv.FormatUint(*id, 10), g.Name
}
func distribution(rows map[string]*StatisticsDistribution, key, label string) *StatisticsDistribution {
	if rows[key] == nil {
		rows[key] = &StatisticsDistribution{Key: key, Label: label}
	}
	return rows[key]
}
func orderedDistribution(rows map[string]*StatisticsDistribution) []StatisticsDistribution {
	out := []StatisticsDistribution{}
	for _, v := range rows {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func (s Analytics) Statistics(ctx context.Context, raw string) (StatisticsResult, error) {
	q, err := parseStatistics(raw)
	v := StatisticsResult{Filters: q.filters, ByGame: []StatisticsDistribution{}, ByMethod: []StatisticsDistribution{}, BySource: []StatisticsDistribution{}, Trend: []StatisticsMonth{}, Effort: []StatisticsEffort{}, Rule: "永久个体ID去重，软删除排除；成果按原始获得游戏、Hunt按目标游戏。日期按已有区间重叠筛选，未知不匹配；月份仅放置可确定月份且完整落在筛选内的记录，非每日投入。方法／状态／数值筛选要求关联匹配Hunt。累计次数按游戏+方法+原样单位分组，不跨单位相加；已知合计注明样本覆盖。效率仅完整已完成Hunt的明确自刷关联成果，含意外闪光；游戏或来源不明、混入其他来源、进行中／放弃、子期间筛选不计算效率；不估计真实概率或运气。"}
	if err != nil {
		return v, err
	}
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		data, e := readAnalytics(tx)
		if e != nil {
			return e
		}
		system := q.filters["collection_system"]
		selected := map[uint64]bool{}
		for id, g := range data.Games {
			if g.CollectionSystem == system {
				selected[id] = true
			}
		}
		if x := q.filters["game_id"]; x != "" {
			id, _ := strconv.ParseUint(x, 10, 63)
			g, ok := data.Games[id]
			if !ok || g.CollectionSystem != system {
				return invalid("游戏不存在或与收藏体系不匹配")
			}
		}
		var huntIDs, recordIDs []uint64
		if e := q.hunts.apply(tx.Model(&model.Hunt{}), "hunt").Pluck("hunts.id", &huntIDs).Error; e != nil {
			return e
		}
		if e := q.records.apply(tx.Model(&model.ShinyRecord{}), "shiny").Pluck("shiny_records.id", &recordIDs).Error; e != nil {
			return e
		}
		huntSet, recordSet := map[uint64]bool{}, map[uint64]bool{}
		for _, id := range huntIDs {
			huntSet[id] = true
		}
		for _, h := range data.Hunts {
			if !huntInCollectionSystem(h, system, data.Games) {
				delete(huntSet, h.ID)
			}
		}
		for _, id := range recordIDs {
			recordSet[id] = true
		}
		allHunts := map[uint64]model.Hunt{}
		outcomes := map[uint64][]model.ShinyRecord{}
		for _, h := range data.Hunts {
			allHunts[h.ID] = h
		}
		for _, r := range data.Records {
			if r.HuntID != nil {
				outcomes[*r.HuntID] = append(outcomes[*r.HuntID], r)
			}
		}
		byGame, byMethod, bySource := map[string]*StatisticsDistribution{}, map[string]*StatisticsDistribution{}, map[string]*StatisticsDistribution{}
		months := map[string]*StatisticsMonth{}
		month := func(key string) *StatisticsMonth {
			if months[key] == nil {
				months[key] = &StatisticsMonth{Month: key}
			}
			return months[key]
		}
		efforts := map[string]*StatisticsEffort{}
		type ratioSum struct {
			count, time                 int64
			countOverflow, timeOverflow bool
			countValues, timeValues     []float64
		}
		ratios := map[string]*ratioSum{}
		for _, h := range data.Hunts {
			if !huntSet[h.ID] {
				continue
			}
			if h.GameID == nil {
				if system != "pokemon" {
					continue
				}
			} else if !selected[*h.GameID] {
				continue
			}
			v.Hunts++
			gkey, gname := gameLabel(h.GameID, data.Games)
			distribution(byGame, gkey, gname).Hunts++
			mkey, mname := "unknown", "方法未知"
			if h.Method != nil {
				mkey = "method:" + *h.Method
				mname = *h.Method
			} else {
				v.Coverage.UnknownMethod++
			}
			distribution(byMethod, mkey, mname).Hunts++
			if h.StartDate == nil {
				v.Coverage.UnknownStartDate++
			} else if len(*h.StartDate) < 10 {
				v.Coverage.PartialStartDate++
			}
			if key := statisticsMonth(h.StartDate, q.filters); key != "" {
				month(key).StartedHunts++
			} else {
				v.Coverage.UnplacedHunts++
			}
			encoded, _ := json.Marshal([]any{h.GameID, h.Method, h.CountUnit})
			key := string(encoded)
			if efforts[key] == nil {
				efforts[key] = &StatisticsEffort{Key: key, GameID: h.GameID, Game: gname, Method: h.Method, Unit: h.CountUnit, Exclusions: map[string]int{}, Note: "合计仅已知值；均值按成果加权，中位数为逐Hunt每成果值、至少2个Hunt才提供。仅描述明确自刷完成子集，不代表概率；排除原因可重叠，不能相加当作总数。"}
				ratios[key] = &ratioSum{}
			}
			group := efforts[key]
			group.Hunts++
			if h.EncounterCount != nil {
				group.KnownCountHunts++
			}
			if h.DurationSeconds != nil {
				group.KnownDurationHunts++
			}
			// Suppress unsafe cumulative totals for this group; ratios have separate cohorts.
			chartSum(&group.Count, h.EncounterCount, &group.Overflow)
			chartSum(&group.Seconds, h.DurationSeconds, &group.Overflow)
			reasons := efficiencyExclusions(h, outcomes[h.ID], q.filters)
			for _, reason := range reasons {
				group.Exclusions[reason]++
			}
			if h.EncounterCount == nil {
				group.Exclusions["missing_count"]++
			}
			if h.CountUnit == nil {
				group.Exclusions["missing_unit"]++
			}
			if h.DurationSeconds == nil {
				group.Exclusions["missing_duration"]++
			} else if *h.DurationSeconds == 0 {
				group.Exclusions["zero_duration"]++
			}
			if h.EncounterCount != nil && *h.EncounterCount > maxChartInteger {
				group.Exclusions["unsafe_count"]++
			}
			if h.DurationSeconds != nil && *h.DurationSeconds > maxChartInteger {
				group.Exclusions["unsafe_duration"]++
			}
			n := len(outcomes[h.ID])
			if len(reasons) > 0 {
				continue
			}
			if h.EncounterCount != nil && h.CountUnit != nil && *h.EncounterCount <= maxChartInteger {
				group.CountSamples++
				group.CountOutcomes += n
				ratios[key].countValues = append(ratios[key].countValues, float64(*h.EncounterCount)/float64(n))
				if ratios[key].count > maxChartInteger-*h.EncounterCount {
					ratios[key].countOverflow = true
				} else {
					ratios[key].count += *h.EncounterCount
				}
			}
			if h.DurationSeconds != nil && *h.DurationSeconds > 0 && *h.DurationSeconds <= maxChartInteger {
				group.TimeSamples++
				group.TimeOutcomes += n
				ratios[key].timeValues = append(ratios[key].timeValues, float64(*h.DurationSeconds)/float64(n))
				if ratios[key].time > maxChartInteger-*h.DurationSeconds {
					ratios[key].timeOverflow = true
				} else {
					ratios[key].time += *h.DurationSeconds
				}
			}
		}
		selectedRecords := []model.ShinyRecord{}
		for _, r := range data.Records {
			if !recordSet[r.ID] || !recordInDexScope(r, true, DexQuery{Scope: "all", System: system}, selected, data.Games) {
				continue
			}
			if q.linkedOnly && (r.HuntID == nil || !huntSet[*r.HuntID]) {
				continue
			}
			v.Individuals++
			selectedRecords = append(selectedRecords, r)
			gkey, gname := gameLabel(r.OriginalGameID, data.Games)
			mkey, mname := "unknown", "方法未知"
			if r.HuntID != nil {
				if h, ok := allHunts[*r.HuntID]; ok && h.Method != nil {
					mkey = "method:" + *h.Method
					mname = *h.Method
				}
			}
			for _, d := range []*StatisticsDistribution{distribution(byGame, gkey, gname), distribution(byMethod, mkey, mname), distribution(bySource, r.Source, r.Source)} {
				d.Individuals++
				if r.Source == "self_hunted" {
					d.SelfHunted++
				}
			}
			if r.OriginalGameID == nil {
				v.Coverage.UnknownOriginalGame++
			}
			if r.ObtainedDate == nil {
				v.Coverage.UnknownObtainedDate++
			} else if len(*r.ObtainedDate) < 10 {
				v.Coverage.PartialObtainedDate++
			}
			if key := statisticsMonth(r.ObtainedDate, q.filters); key != "" {
				m := month(key)
				m.Individuals++
				if r.Source == "self_hunted" {
					m.SelfHunted++
				}
			} else {
				v.Coverage.UnplacedRecords++
			}
		}
		v.ByGame = orderedDistribution(byGame)
		v.ByMethod = orderedDistribution(byMethod)
		v.BySource = orderedDistribution(bySource)
		for _, m := range months {
			v.Trend = append(v.Trend, *m)
		}
		sort.Slice(v.Trend, func(i, j int) bool { return v.Trend[i].Month < v.Trend[j].Month })
		for key, g := range efforts {
			if g.Overflow {
				g.Count = nil
				g.Seconds = nil
			}
			r := ratios[key]
			g.CountSampleStatus = sampleStatus(g.CountSamples)
			g.TimeSampleStatus = sampleStatus(g.TimeSamples)
			g.MedianCountPerShiny = sampleMedian(r.countValues)
			g.MedianSecondsPerShiny = sampleMedian(r.timeValues)
			if g.CountOutcomes > 0 && !r.countOverflow {
				x := float64(r.count) / float64(g.CountOutcomes)
				g.CountPerShiny = &x
			}
			if g.TimeOutcomes > 0 && !r.timeOverflow {
				x := float64(r.time) / float64(g.TimeOutcomes)
				g.SecondsPerShiny = &x
			}
			v.Effort = append(v.Effort, *g)
		}
		sort.Slice(v.Effort, func(i, j int) bool { return v.Effort[i].Key < v.Effort[j].Key })
		v.Quality = statisticsQuality(selectedRecords, len(v.Effort))
		return nil
	})
	return v, err
}
