package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	_ "time/tzdata"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"gorm.io/gorm"
)

type BingoCreateInput struct {
	RequestID    string       `json:"request_id"`
	Rules        PlannerRules `json:"rules"`
	Size         int          `json:"size"`
	CalendarZone string       `json:"calendar_zone"`
}
type BingoRerollInput struct {
	RequestID         string `json:"request_id"`
	ExpectedTargetKey string `json:"expected_target_key"`
}
type BingoDetails struct {
	Size         int                 `json:"size"`
	StartedAt    time.Time           `json:"started_at"`
	CalendarZone string              `json:"calendar_zone"`
	Lines        [][]int             `json:"lines"`
	Cells        []BingoCellEvidence `json:"cells"`
	Events       []model.BingoEvent  `json:"events"`
}
type BingoReview struct {
	RecordID    uint64 `json:"record_id"`
	Code        string `json:"code"`
	Reason      string `json:"reason"`
	NeedsReview bool   `json:"needs_review"`
}
type BingoCellEvidence struct {
	TargetID uint64        `json:"target_id"`
	Kind     string        `json:"kind"`
	Reviews  []BingoReview `json:"reviews"`
	Proof    *BingoProof   `json:"proof,omitempty"`
}

func (s Planner) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}
func bingoEvent(tx *gorm.DB, plan uint64, target *uint64, kind, summary string, before, after any, request *string) error {
	return tx.Omit("Plan", "Target").Create(&model.BingoEvent{PlanID: plan, TargetID: target, Kind: kind, Summary: summary, RequestID: request, BeforeJSON: plannerJSON(before), AfterJSON: plannerJSON(after)}).Error
}

// BingoLines returns positions, never an implicit free cell.
func BingoLines(size int, completed map[int]bool) [][]int {
	lines := [][]int{}
	if size != 3 && size != 5 {
		return lines
	}
	check := func(p []int) {
		for _, v := range p {
			if !completed[v] {
				return
			}
		}
		lines = append(lines, p)
	}
	for i := 0; i < size; i++ {
		row, col := []int{}, []int{}
		for j := 0; j < size; j++ {
			row = append(row, i*size+j+1)
			col = append(col, j*size+i+1)
		}
		check(row)
		check(col)
	}
	a, b := []int{}, []int{}
	for i := 0; i < size; i++ {
		a = append(a, i*size+i+1)
		b = append(b, i*size+size-i)
	}
	check(a)
	check(b)
	return lines
}
func attachBingo(tx *gorm.DB, v *PlannerView) error {
	var challenge model.BingoChallenge
	if e := tx.Where("plan_id = ?", v.ID).First(&challenge).Error; e != nil {
		return e
	}
	if len(v.Targets) != challenge.Size*challenge.Size {
		return errors.New("incomplete bingo grid")
	}
	info := &BingoDetails{Size: challenge.Size, StartedAt: challenge.StartedAt, CalendarZone: challenge.CalendarZone, Cells: []BingoCellEvidence{}, Events: []model.BingoEvent{}}
	complete := map[int]bool{}
	for _, t := range v.Targets {
		if t.Status == "completed" && t.ShinyRecordID != nil {
			complete[t.Position] = true
		}
		cell := BingoCellEvidence{TargetID: t.ID, Kind: "none", Reviews: []BingoReview{}}
		if e := json.Unmarshal([]byte(t.ProofJSON), &cell); e != nil {
			return e
		}
		cell.TargetID = t.ID
		if cell.Kind == "" {
			cell.Kind = "none"
		}
		if cell.Reviews == nil {
			cell.Reviews = []BingoReview{}
		}
		info.Cells = append(info.Cells, cell)
	}
	info.Lines = BingoLines(challenge.Size, complete)
	if e := tx.Where("plan_id = ?", v.ID).Order("id ASC").Find(&info.Events).Error; e != nil {
		return e
	}
	v.Bingo = info
	return nil
}
func (s Planner) CreateBingo(ctx context.Context, in BingoCreateInput) (PlannerView, error) {
	var out PlannerView
	if (in.Size != 3 && in.Size != 5) || !plannerRequestPattern.MatchString(in.RequestID) {
		return out, invalid("Bingo尺寸或请求标识无效")
	}
	if in.CalendarZone == "" {
		in.CalendarZone = "UTC"
	}
	if len(in.CalendarZone) > 64 || in.CalendarZone == "Local" {
		return out, invalid("日历时区无效")
	}
	if _, e := time.LoadLocation(in.CalendarZone); e != nil {
		return out, invalid("日历时区无效")
	}
	in.Rules.Count = in.Size * in.Size
	if e := in.Rules.normalize(); e != nil {
		return out, e
	}
	e := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var prior model.HuntPlan
		e := tx.Where("request_id = ?", in.RequestID).First(&prior).Error
		if e == nil {
			if prior.Kind != "bingo" || prior.RulesJSON != plannerJSON(in.Rules) {
				return conflict("同一请求不可改变挑战规则")
			}
			out, e = plannerView(tx, prior.ID)
			if e == nil && (out.Bingo.Size != in.Size || out.Bingo.CalendarZone != in.CalendarZone) {
				return conflict("同一请求不可改变尺寸或时区")
			}
			return e
		}
		if !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		pool, e := s.pool(ctx, tx, in.Rules)
		if e != nil {
			return e
		}
		if pool.Coverage.Shortage {
			return &Error{"insufficient_candidates", "已核实唯一候选不足，不能创建完整Bingo"}
		}
		picks, e := s.pick(pool)
		if e != nil {
			return e
		}
		p := model.HuntPlan{RequestID: in.RequestID, Kind: "bingo", Status: "active", RulesJSON: plannerJSON(in.Rules), SnapshotJSON: plannerJSON(pool.Coverage)}
		if e = tx.Create(&p).Error; e != nil {
			return e
		}
		challenge := model.BingoChallenge{PlanID: p.ID, Size: in.Size, StartedAt: s.now(), CalendarZone: in.CalendarZone}
		if e = tx.Omit("Plan").Create(&challenge).Error; e != nil {
			return e
		}
		for i, pick := range picks {
			t := model.HuntPlanTarget{PlanID: p.ID, Position: i + 1, TargetKey: pick.Key, SpeciesID: pick.SpeciesID, FormID: pick.FormID, GameID: pick.Choice.GameID, Status: "pending", SnapshotJSON: plannerJSON(pick), ProofJSON: "{}"}
			if e = tx.Omit("Plan", "Species", "Form", "Game", "Hunt", "ShinyRecord").Create(&t).Error; e != nil {
				return e
			}
		}
		if e = bingoEvent(tx, p.ID, nil, "created", fmt.Sprintf("创建 %d×%d 挑战（无免费格）", in.Size, in.Size), nil, challenge, nil); e != nil {
			return e
		}
		out, e = plannerView(tx, p.ID)
		return e
	})
	return out, e
}
func (s Planner) RerollBingo(ctx context.Context, planID, targetID uint64, in BingoRerollInput) (PlannerView, error) {
	var out PlannerView
	if !plannerRequestPattern.MatchString(in.RequestID) || in.ExpectedTargetKey == "" {
		return out, invalid("重抽需提供请求标识和当前目标")
	}
	e := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		key := "reroll:" + in.RequestID
		var event model.BingoEvent
		e := tx.Where("request_id = ?", key).First(&event).Error
		if e == nil {
			var before model.HuntPlanTarget
			if e = json.Unmarshal([]byte(event.BeforeJSON), &before); e != nil {
				return e
			}
			if event.PlanID != planID || event.TargetID == nil || *event.TargetID != targetID || before.TargetKey != in.ExpectedTargetKey {
				return conflict("重抽请求标识已被使用")
			}
			out, e = plannerView(tx, planID)
			return e
		}
		if !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		v, e := s.refreshBingo(ctx, tx, planID)
		if e != nil {
			return e
		}
		if v.Kind != "bingo" || v.Status != "active" {
			return conflict("已结束的挑战不能重抽")
		}
		var target model.HuntPlanTarget
		if e = tx.Where("id = ? AND plan_id = ?", targetID, planID).First(&target).Error; e != nil {
			return e
		}
		if target.TargetKey != in.ExpectedTargetKey || target.Status != "pending" || target.HuntID != nil || target.ShinyRecordID != nil {
			return conflict("只能重抽未开始且未完成的当前格子")
		}
		rules := v.Rules
		rules.Count = 1
		pool, e := s.pool(ctx, tx, rules)
		if e != nil {
			return e
		}
		excluded := map[string]bool{}
		for _, t := range v.Targets {
			excluded[t.TargetKey] = true
		}
		candidates := []PlannerCandidate{}
		for _, c := range pool.Candidates {
			if !excluded[c.Key] {
				candidates = append(candidates, c)
			}
		}
		if len(candidates) == 0 {
			return &Error{"insufficient_candidates", "没有不与当前卡片重复的新目标"}
		}
		pool.Candidates = candidates
		pool.Coverage.Actual = 1
		picks, e := s.pick(pool)
		if e != nil {
			return e
		}
		pick := picks[0]
		before := target
		if e = tx.Model(&target).Updates(map[string]any{"target_key": pick.Key, "species_id": pick.SpeciesID, "form_id": pick.FormID, "game_id": pick.Choice.GameID, "snapshot_json": plannerJSON(pick), "proof_json": "{}"}).Error; e != nil {
			return e
		}
		var after model.HuntPlanTarget
		if e = tx.First(&after, targetID).Error; e != nil {
			return e
		}
		var old PlannerPick
		_ = json.Unmarshal([]byte(before.SnapshotJSON), &old)
		if e = bingoEvent(tx, planID, &target.ID, "rerolled", fmt.Sprintf("格子 %d：%s → %s", target.Position, old.Name, pick.Name), before, after, &key); e != nil {
			return e
		}
		out, e = plannerView(tx, planID)
		return e
	})
	return out, e
}
func (s Planner) CloseBingo(ctx context.Context, id uint64, status string) (PlannerView, error) {
	var out PlannerView
	if !member(status, "completed", "abandoned") {
		return out, invalid("挑战结束状态无效")
	}
	e := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		v, e := plannerView(tx, id)
		if e != nil {
			return e
		}
		if v.Kind != "bingo" {
			return conflict("这不是Bingo挑战")
		}
		if v.Status == status {
			out = v
			return nil
		}
		if v.Status != "active" {
			return conflict("已结束挑战不能改写")
		}
		if status == "completed" && len(v.Bingo.Lines) == 0 {
			return conflict("至少形成一条连线后才能结束为完成")
		}
		if e = tx.Model(&v.HuntPlan).Update("status", status).Error; e != nil {
			return e
		}
		summary := "挑战已放弃，历史保留"
		if status == "completed" {
			summary = "连线后结束挑战，历史保留"
		}
		if e = bingoEvent(tx, id, nil, status, summary, nil, map[string]string{"status": status}, nil); e != nil {
			return e
		}
		out, e = plannerView(tx, id)
		return e
	})
	return out, e
}
