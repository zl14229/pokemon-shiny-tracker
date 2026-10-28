package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"regexp"
	"strconv"
	"time"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"gorm.io/gorm"
)

type PlannerDrawInput struct {
	RequestID string       `json:"request_id"`
	Rules     PlannerRules `json:"rules"`
}
type PlannerTargetView struct {
	model.HuntPlanTarget
	Snapshot PlannerPick `json:"snapshot"`
}
type PlannerView struct {
	model.HuntPlan
	Rules    PlannerRules        `json:"rules"`
	Coverage PlannerCoverage     `json:"coverage"`
	Targets  []PlannerTargetView `json:"targets"`
	Bingo    *BingoDetails       `json:"bingo,omitempty"`
}

var plannerRequestPattern = regexp.MustCompile(`^[a-zA-Z0-9-]{8,80}$`)

func plannerView(tx *gorm.DB, id uint64) (PlannerView, error) {
	var out PlannerView
	if e := tx.First(&out.HuntPlan, id).Error; e != nil {
		return out, e
	}
	if e := json.Unmarshal([]byte(out.RulesJSON), &out.Rules); e != nil {
		return out, e
	}
	if e := json.Unmarshal([]byte(out.SnapshotJSON), &out.Coverage); e != nil {
		return out, e
	}
	out.Targets = []PlannerTargetView{}
	var targets []model.HuntPlanTarget
	if e := tx.Where("plan_id = ?", id).Order("position ASC").Find(&targets).Error; e != nil {
		return out, e
	}
	for _, row := range targets {
		v := PlannerTargetView{HuntPlanTarget: row}
		if e := json.Unmarshal([]byte(row.SnapshotJSON), &v.Snapshot); e != nil {
			return out, e
		}
		out.Targets = append(out.Targets, v)
	}
	if out.Kind == "bingo" {
		if e := attachBingo(tx, &out); e != nil {
			return out, e
		}
	}
	return out, nil
}
func (s Planner) Get(ctx context.Context, id uint64) (PlannerView, error) {
	var out PlannerView
	e := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error { var e error; out, e = plannerView(tx, id); return e })
	return out, e
}
func (s Planner) Draw(ctx context.Context, in PlannerDrawInput) (PlannerView, error) {
	var out PlannerView
	if !plannerRequestPattern.MatchString(in.RequestID) {
		return out, invalid("生成请求标识无效")
	}
	if e := in.Rules.normalize(); e != nil {
		return out, e
	}
	e := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var prior model.HuntPlan
		e := tx.Where("request_id = ?", in.RequestID).First(&prior).Error
		if e == nil {
			if prior.Kind != "random" || prior.RulesJSON != plannerJSON(in.Rules) {
				return conflict("相同请求不能改变筛选")
			}
			out, e = plannerView(tx, prior.ID)
			return e
		}
		if !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		pool, e := s.pool(ctx, tx, in.Rules)
		if e != nil {
			return e
		}
		picks, e := s.pick(pool)
		if e != nil {
			return e
		}
		plan := model.HuntPlan{RequestID: in.RequestID, Kind: "random", Status: "draft", RulesJSON: plannerJSON(in.Rules), SnapshotJSON: plannerJSON(pool.Coverage)}
		if e = tx.Create(&plan).Error; e != nil {
			return e
		}
		for i, pick := range picks {
			target := model.HuntPlanTarget{PlanID: plan.ID, Position: i + 1, TargetKey: pick.Key, SpeciesID: pick.SpeciesID, FormID: pick.FormID, GameID: pick.Choice.GameID, Status: "pending", SnapshotJSON: plannerJSON(pick), ProofJSON: "{}"}
			if e = tx.Omit("Plan", "Species", "Form", "Game", "Hunt", "ShinyRecord").Create(&target).Error; e != nil {
				return e
			}
		}
		out, e = plannerView(tx, plan.ID)
		return e
	})
	return out, e
}
func (s Planner) Save(ctx context.Context, id uint64) (PlannerView, error) {
	var out PlannerView
	e := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		v, e := plannerView(tx, id)
		if e != nil {
			return e
		}
		if len(v.Targets) == 0 {
			return conflict("没有目标的结果不能保存")
		}
		if v.Status == "draft" {
			if e = tx.Model(&v.HuntPlan).Update("status", "active").Error; e != nil {
				return e
			}
		}
		out, e = plannerView(tx, id)
		return e
	})
	return out, e
}

type PlannerListItem struct {
	ID        uint64    `json:"id"`
	Kind      string    `json:"kind"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	Targets   int       `json:"target_count"`
}

func (s Planner) List(ctx context.Context, limit, offset int) ([]PlannerListItem, int64, error) {
	result := []PlannerListItem{}
	var total int64
	e := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Model(&model.HuntPlan{}).Where("status <> 'draft'").Count(&total).Error; e != nil {
			return e
		}
		return tx.Model(&model.HuntPlan{}).Select("hunt_plans.id,hunt_plans.kind,hunt_plans.status,hunt_plans.created_at,(SELECT count(*) FROM hunt_plan_targets WHERE plan_id=hunt_plans.id) AS targets").Where("status <> 'draft'").Order("hunt_plans.id DESC").Limit(limit).Offset(offset).Scan(&result).Error
	})
	return result, total, e
}

// currentPick checks fresh local evidence. Saved snapshots remain history after
// a reference retraction, but cannot create a Hunt from a retracted claim.
func (s Planner) currentPick(ctx context.Context, tx *gorm.DB, v PlannerView, t model.HuntPlanTarget) (PlannerPick, error) {
	rules := v.Rules
	rules.Count = 1
	pool, e := s.pool(ctx, tx, rules)
	if e != nil {
		return PlannerPick{}, e
	}
	for _, c := range pool.Candidates {
		if c.Key != t.TargetKey {
			continue
		}
		for _, g := range c.Games {
			if g.GameID == t.GameID {
				return PlannerPick{c, g}, nil
			}
		}
	}
	return PlannerPick{}, conflict("目标不再符合当前已核实资料或缺失池，请重新查看候选")
}
func (s Planner) StartHunt(ctx context.Context, planID, targetID uint64, existing *uint64, expectedKey ...string) (PlannerView, error) {
	var out PlannerView
	e := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		v, e := plannerView(tx, planID)
		if e != nil {
			return e
		}
		if v.Status != "active" {
			return conflict("先保存计划，结束的计划不能开始目标")
		}
		var target model.HuntPlanTarget
		if e = tx.Where("plan_id = ? AND id = ?", planID, targetID).First(&target).Error; e != nil {
			return e
		}
		if v.Kind == "bingo" && (len(expectedKey) != 1 || expectedKey[0] != target.TargetKey) {
			return conflict("格子目标已改变，请重新读取")
		}
		if target.HuntID != nil {
			if existing != nil && *existing != *target.HuntID {
				return conflict("目标已有关联Hunt")
			}
			var h model.Hunt
			if e = tx.First(&h, *target.HuntID).Error; e != nil {
				return conflict("关联Hunt已删除，请核对计划历史")
			}
			out, e = plannerView(tx, planID)
			return e
		}
		if target.Status != "pending" {
			return conflict("仅未开始目标可以关联")
		}
		if existing != nil {
			allow := false
			v.Rules.ExcludeActive = &allow
		}
		pick, e := s.currentPick(ctx, tx, v, target)
		if e != nil {
			return e
		}
		var hunt model.Hunt
		if existing != nil {
			if !validID(*existing) {
				return invalid("Hunt ID无效")
			}
			if e = tx.First(&hunt, *existing).Error; e != nil {
				return e
			}
			if hunt.Status != "in_progress" || hunt.SpeciesID != target.SpeciesID || hunt.GameID == nil || *hunt.GameID != target.GameID || target.FormID != nil && (hunt.FormID == nil || *hunt.FormID != *target.FormID) {
				return conflict("Hunt目标或游戏不符合计划")
			}
		} else {
			notes := "来自 Hunt Planner #" + strconv.FormatUint(planID, 10)
			input := HuntInput{SpeciesID: target.SpeciesID, FormID: target.FormID, GameID: &target.GameID, Status: "in_progress", Notes: &notes}
			methods := map[string]bool{}
			for _, row := range pick.Choice.Evidence {
				for _, m := range row.Fact.Methods {
					methods[m.Name] = true
				}
			}
			if len(methods) == 1 {
				for method := range methods {
					input.Method = &method
				}
			}
			hunt, e = (Hunts{DB: tx}).Create(ctx, input)
			if e != nil {
				return e
			}
		}
		beforeTarget := target
		if e = tx.Model(&target).Updates(map[string]any{"hunt_id": hunt.ID, "status": "started"}).Error; e != nil {
			return e
		}
		if v.Kind == "bingo" {
			after := target
			after.HuntID = &hunt.ID
			after.Status = "started"
			if e = bingoEvent(tx, planID, &target.ID, "hunt_linked", fmt.Sprintf("格子 %d 关联 Hunt #%d", target.Position, hunt.ID), beforeTarget, after, nil); e != nil {
				return e
			}
		}
		out, e = plannerView(tx, planID)
		return e
	})
	return out, e
}

// ValidatePlannerPaging is shared with the dedicated planner API.
func ValidatePlannerPaging(limit, offset int) error {
	if limit < 1 || limit > 100 || offset < 0 || offset > 2147483647 {
		return invalid("计划分页无效")
	}
	return nil
}
