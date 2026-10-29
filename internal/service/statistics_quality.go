package service

import (
	"encoding/json"
	"sort"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
)

type StatisticsCandidate struct {
	RecordIDs []uint64 `json:"record_ids"`
	Count     int      `json:"count"`
}
type StatisticsQuality struct {
	DistinctRecords     int                   `json:"distinct_records"`
	CandidateGroups     int                   `json:"candidate_groups"`
	CandidateRecords    int                   `json:"candidate_records"`
	Candidates          []StatisticsCandidate `json:"candidates"`
	CandidatesTruncated bool                  `json:"candidates_truncated"`
	CountComparison     string                `json:"count_comparison"`
	Note                string                `json:"note"`
}

func statisticsQuality(records []model.ShinyRecord, contexts int) StatisticsQuality {
	v := StatisticsQuality{Candidates: []StatisticsCandidate{}, CountComparison: "unavailable", Note: "永久ID分别计数。相似记录沿用Excel导入候选规则（当前物种、原始游戏、获得日期相同，含未知）；只是核对线索，可能是不同个体，不自动合并、扣除或修改。最多显示20组／每组100个ID，候选总数保留。次数只在同一游戏／方法／原样单位上下文中展示，不推断相同单位名就语义兼容。"}
	if contexts == 1 {
		v.CountComparison = "single_context"
	} else if contexts > 1 {
		v.CountComparison = "separate_contexts"
	}
	buckets := map[string][]uint64{}
	representatives := map[string]model.ShinyRecord{}
	seen := map[uint64]bool{}
	for _, r := range records {
		if seen[r.ID] {
			continue
		}
		seen[r.ID] = true
		keyJSON, _ := json.Marshal([]any{r.CurrentSpeciesID, r.OriginalGameID, r.ObtainedDate})
		key := string(keyJSON)
		if old, ok := representatives[key]; ok {
			// Reuse the import comparison: a candidate is never an identity assertion.
			if !duplicateImport(ShinyInput{CurrentSpeciesID: r.CurrentSpeciesID, OriginalGameID: r.OriginalGameID, ObtainedDate: r.ObtainedDate}, old) {
				continue
			}
		} else {
			representatives[key] = r
		}
		buckets[key] = append(buckets[key], r.ID)
	}
	v.DistinctRecords = len(seen)
	groups := []StatisticsCandidate{}
	for _, ids := range buckets {
		if len(ids) < 2 {
			continue
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		v.CandidateGroups++
		v.CandidateRecords += len(ids)
		n := len(ids)
		if n > 100 {
			ids = ids[:100]
			v.CandidatesTruncated = true
		}
		groups = append(groups, StatisticsCandidate{RecordIDs: ids, Count: n})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].RecordIDs[0] < groups[j].RecordIDs[0] })
	if len(groups) > 20 {
		groups = groups[:20]
		v.CandidatesTruncated = true
	}
	v.Candidates = groups
	return v
}

func efficiencyExclusions(h model.Hunt, outcomes []model.ShinyRecord, filters map[string]string) []string {
	reasons := []string{}
	if h.Status != "completed" {
		reasons = append(reasons, "unfinished_hunt")
	}
	if h.GameID == nil {
		reasons = append(reasons, "unknown_game")
	}
	if h.Method == nil {
		reasons = append(reasons, "unknown_method")
	}
	if filters["from"] != "" || filters["to"] != "" {
		reasons = append(reasons, "date_filter")
	}
	if filters["source"] != "" && filters["source"] != "self_hunted" {
		reasons = append(reasons, "source_filter")
	}
	if len(outcomes) == 0 {
		reasons = append(reasons, "no_outcomes")
	}
	for _, r := range outcomes {
		if r.Source != "self_hunted" || !same(r.OriginalGameID, h.GameID) {
			reasons = append(reasons, "outcome_evidence")
			break
		}
	}
	return reasons
}

func sampleStatus(n int) string {
	if n == 0 {
		return "no_eligible_samples"
	}
	if n == 1 {
		return "single_sample"
	}
	return "descriptive_subset"
}
func sampleMedian(values []float64) *float64 {
	if len(values) < 2 {
		return nil
	}
	sort.Float64s(values)
	n := len(values)
	v := values[n/2]
	if n%2 == 0 {
		v = values[n/2-1]/2 + v/2
	}
	return &v
}
