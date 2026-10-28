package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/zl14229/pokemon-shiny-tracker/internal/huntdata"
	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"gorm.io/gorm"
)

type BingoProof struct {
	Method            string       `json:"method"`
	RecordID          uint64       `json:"record_id"`
	ObtainedDate      *string      `json:"obtained_date"`
	ObtainedAt        *time.Time   `json:"obtained_at"`
	EvidenceNote      string       `json:"evidence_note"`
	ConfirmedAt       time.Time    `json:"confirmed_at"`
	OriginalSpeciesID uint64       `json:"original_species_id"`
	OriginalFormID    uint64       `json:"original_form_id"`
	OriginalGameID    uint64       `json:"original_game_id"`
	Source            string       `json:"source"`
	Availability      huntdata.Row `json:"availability"`
}
type BingoDateInput struct {
	RecordID     uint64    `json:"record_id"`
	ObtainedAt   time.Time `json:"obtained_at"`
	EvidenceNote string    `json:"evidence_note"`
	Confirmed    bool      `json:"confirmed"`
}

func bingoLegal(tx *gorm.DB) (map[[2]uint64]huntdata.Row, error) {
	rows, e := huntdata.Read(tx)
	if e != nil {
		return nil, &Error{"reference_unavailable", "刷闪资料来源不一致，未自动确认成果"}
	}
	var entries []model.CatalogEntry
	if e = tx.Where("kind = 'form' AND inclusion = 'included'").Find(&entries).Error; e != nil {
		return nil, e
	}
	included := map[uint64]bool{}
	for _, v := range entries {
		if v.FormID != nil {
			included[*v.FormID] = true
		}
	}
	result := map[[2]uint64]huntdata.Row{}
	for _, r := range rows {
		if r.Fact.Huntable() && included[r.FormID] {
			result[[2]uint64{r.GameID, r.FormID}] = r
		}
	}
	return result, nil
}
func bingoPotential(t model.HuntPlanTarget, r model.ShinyRecord) bool {
	if r.CurrentSpeciesID != t.SpeciesID && (r.OriginalSpeciesID == nil || *r.OriginalSpeciesID != t.SpeciesID) {
		return false
	}
	if t.FormID != nil && r.OriginalFormID != nil && r.CurrentFormID != nil && *r.OriginalFormID != *t.FormID && *r.CurrentFormID != *t.FormID {
		return false
	}
	return true
}
func inspectBingoRecord(t model.HuntPlanTarget, r model.ShinyRecord, c model.BingoChallenge, legal map[[2]uint64]huntdata.Row, now time.Time, skipDate bool) (BingoReview, *huntdata.Row) {
	review := BingoReview{RecordID: r.ID}
	fail := func(code, reason string, needs bool) (BingoReview, *huntdata.Row) {
		review.Code = code
		review.Reason = reason
		review.NeedsReview = needs
		return review, nil
	}
	if r.CreatedAt.Before(c.StartedAt) {
		return fail("old_identity", "挑战创建前已有的个体不自动计入", false)
	}
	if r.CreatedAt.After(now) {
		return fail("future_identity", "记录创建时间在未来，需核对", true)
	}
	if r.Source != "self_hunted" {
		if r.Source == "unknown" {
			return fail("source_missing", "获得来源不明，需先核对个人记录", true)
		}
		return fail("not_self_hunted", "交换、活动或其他来源不计入亲自刷闪成果", false)
	}
	if r.OriginalSpeciesID == nil {
		return fail("original_missing", "原始获得物种未知，不以当前物种补推", true)
	}
	if *r.OriginalSpeciesID != t.SpeciesID {
		return fail("wrong_original_target", "原始获得目标不匹配，不从后续进化推断该目标的新获得", true)
	}
	if r.OriginalGameID == nil {
		return fail("original_game_missing", "原始获得游戏未知，不以当前保存位置代替", true)
	}
	if *r.OriginalGameID != t.GameID {
		return fail("wrong_game", "原始获得游戏不匹配，单纯传送不计入", false)
	}
	if r.OriginalFormID == nil {
		return fail("original_form_missing", "原始形态未知，无法核对该形态的合法可刷证据", true)
	}
	if t.FormID != nil && *r.OriginalFormID != *t.FormID {
		return fail("wrong_form", "原始获得形态不符合目标", false)
	}
	fact, ok := legal[[2]uint64{t.GameID, *r.OriginalFormID}]
	if !ok || fact.SpeciesID != t.SpeciesID {
		return fail("legality_missing", "尚无该游戏和原始形态的已核实直接可刷证据", true)
	}
	if !skipDate {
		if r.ObtainedDate == nil {
			return fail("date_missing", "获得日期未知，不能自动判断为挑战后获得", true)
		}
		if len(*r.ObtainedDate) != 10 {
			return fail("date_precision", "获得日期只有年或月，需补充具体获得时间证据", true)
		}
		if _, _, e := DateBounds(*r.ObtainedDate); e != nil {
			return fail("date_invalid", "获得日期无效，请先核对个人记录", true)
		}
		loc, e := time.LoadLocation(c.CalendarZone)
		if e != nil {
			return fail("zone_invalid", "挑战日历时区无效", true)
		}
		start, today := c.StartedAt.In(loc).Format("2006-01-02"), now.In(loc).Format("2006-01-02")
		if *r.ObtainedDate < start {
			return fail("old_date", "获得日期早于挑战，历史收藏不计入", false)
		}
		if *r.ObtainedDate > today {
			return fail("future_date", "获得日期在未来，不能自动完成", true)
		}
		if *r.ObtainedDate == start {
			return fail("same_day", "同日只有日期，无法证明晚于挑战创建；需人工核验获得时间", true)
		}
	}
	return review, &fact
}
func completeBingoTarget(tx *gorm.DB, plan uint64, t model.HuntPlanTarget, r model.ShinyRecord, fact huntdata.Row, method, note string, obtainedAt *time.Time, now time.Time) error {
	proof := BingoProof{Method: method, RecordID: r.ID, ObtainedDate: r.ObtainedDate, ObtainedAt: obtainedAt, EvidenceNote: note, ConfirmedAt: now, OriginalSpeciesID: *r.OriginalSpeciesID, OriginalFormID: *r.OriginalFormID, OriginalGameID: *r.OriginalGameID, Source: r.Source, Availability: fact}
	cell := BingoCellEvidence{TargetID: t.ID, Kind: "completed", Reviews: []BingoReview{}, Proof: &proof}
	if e := tx.Model(&model.HuntPlanTarget{}).Where("id = ?", t.ID).Updates(map[string]any{"status": "completed", "shiny_record_id": r.ID, "proof_json": plannerJSON(cell)}).Error; e != nil {
		return e
	}
	label := "获得日期"
	if method == "manual_timestamp" {
		label = "用户补充时间证据"
	}
	return bingoEvent(tx, plan, &t.ID, "record_completed", fmt.Sprintf("格子 %d 由个体 #%d 完成（%s）", t.Position, r.ID, label), t, cell, nil)
}
func closeFullBingo(tx *gorm.DB, id uint64) error {
	v, e := plannerView(tx, id)
	if e != nil {
		return e
	}
	if v.Status != "active" {
		return nil
	}
	for _, t := range v.Targets {
		if t.Status != "completed" {
			return nil
		}
	}
	if e = tx.Model(&v.HuntPlan).Update("status", "completed").Error; e != nil {
		return e
	}
	return bingoEvent(tx, id, nil, "completed", "挑战全格完成，历史保留", nil, map[string]string{"status": "completed"}, nil)
}
func (s Planner) refreshBingo(ctx context.Context, tx *gorm.DB, id uint64) (PlannerView, error) {
	v, e := plannerView(tx, id)
	if e != nil {
		return v, e
	}
	if v.Kind != "bingo" {
		return v, conflict("这不是Bingo挑战")
	}
	if v.Status != "active" {
		return v, nil
	}
	var c model.BingoChallenge
	if e = tx.Where("plan_id = ?", id).First(&c).Error; e != nil {
		return v, e
	}
	legal, e := bingoLegal(tx)
	if e != nil {
		return v, e
	}
	var records []model.ShinyRecord
	if e = tx.Order("id ASC").Find(&records).Error; e != nil {
		return v, e
	}
	now := s.now()
	used := map[uint64]bool{}
	for _, t := range v.Targets {
		if t.ShinyRecordID != nil {
			used[*t.ShinyRecordID] = true
		}
	}
	for _, t := range v.Targets {
		if t.Status == "completed" {
			continue
		}
		cell := BingoCellEvidence{TargetID: t.ID, Kind: "none", Reviews: []BingoReview{}}
		done := false
		for _, r := range records {
			if used[r.ID] || !bingoPotential(t.HuntPlanTarget, r) {
				continue
			}
			review, fact := inspectBingoRecord(t.HuntPlanTarget, r, c, legal, now, false)
			if fact != nil {
				if e = completeBingoTarget(tx, id, t.HuntPlanTarget, r, *fact, "record_date", "", nil, now); e != nil {
					return v, e
				}
				used[r.ID] = true
				done = true
				break
			}
			cell.Reviews = append(cell.Reviews, review)
			if review.NeedsReview {
				cell.Kind = "needs_review"
			} else if cell.Kind == "none" {
				cell.Kind = "excluded"
			}
		}
		if !done {
			raw := "{}"
			if len(cell.Reviews) > 0 {
				raw = plannerJSON(cell)
			}
			if raw != t.ProofJSON {
				if e = tx.Model(&model.HuntPlanTarget{}).Where("id = ?", t.ID).Update("proof_json", raw).Error; e != nil {
					return v, e
				}
			}
		}
	}
	if e = closeFullBingo(tx, id); e != nil {
		return v, e
	}
	return plannerView(tx, id)
}
func (s Planner) RefreshBingo(ctx context.Context, id uint64) (PlannerView, error) {
	var out PlannerView
	e := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error { var e error; out, e = s.refreshBingo(ctx, tx, id); return e })
	return out, e
}

// VerifyBingoDate is an explicit user attestation, never an automatic exception.
// It can resolve time precision only; identity, source, game, form and current
// local legal evidence remain mandatory and cannot be overridden.
func (s Planner) VerifyBingoDate(ctx context.Context, planID, targetID uint64, in BingoDateInput) (PlannerView, error) {
	var out PlannerView
	in.EvidenceNote = strings.TrimSpace(in.EvidenceNote)
	if !validID(in.RecordID) || in.ObtainedAt.IsZero() || !in.Confirmed || in.EvidenceNote == "" || utf8.RuneCountInString(in.EvidenceNote) > 500 {
		return out, invalid("须提供个体、具体获得时间、核验依据并明确确认")
	}
	in.ObtainedAt = in.ObtainedAt.UTC()
	e := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		v, e := plannerView(tx, planID)
		if e != nil {
			return e
		}
		if v.Kind != "bingo" {
			return conflict("这不是Bingo挑战")
		}
		var target model.HuntPlanTarget
		if e = tx.Where("plan_id = ? AND id = ?", planID, targetID).First(&target).Error; e != nil {
			return e
		}
		if target.Status == "completed" {
			var cell BingoCellEvidence
			_ = json.Unmarshal([]byte(target.ProofJSON), &cell)
			if cell.Proof != nil && cell.Proof.Method == "manual_timestamp" && cell.Proof.RecordID == in.RecordID && cell.Proof.ObtainedAt != nil && cell.Proof.ObtainedAt.Equal(in.ObtainedAt) && cell.Proof.EvidenceNote == in.EvidenceNote {
				out = v
				return nil
			}
			return conflict("已完成格子的证据不可静默替换")
		}
		if v.Status != "active" {
			return conflict("挑战已结束")
		}
		var c model.BingoChallenge
		if e = tx.Where("plan_id = ?", planID).First(&c).Error; e != nil {
			return e
		}
		now := s.now()
		if !in.ObtainedAt.After(c.StartedAt) || in.ObtainedAt.After(now) {
			return invalid("获得时间必须晚于挑战创建且不能在未来")
		}
		var r model.ShinyRecord
		if e = tx.First(&r, in.RecordID).Error; e != nil {
			return e
		}
		legal, e := bingoLegal(tx)
		if e != nil {
			return e
		}
		review, fact := inspectBingoRecord(target, r, c, legal, now, true)
		if fact == nil {
			return conflict(review.Reason)
		}
		if r.ObtainedDate != nil {
			if _, _, e = DateBounds(*r.ObtainedDate); e != nil {
				return conflict("先修正个人记录中无效的日期")
			}
			loc, e := time.LoadLocation(c.CalendarZone)
			if e != nil {
				return e
			}
			day := in.ObtainedAt.In(loc).Format("2006-01-02")
			if !strings.HasPrefix(day, *r.ObtainedDate) {
				return conflict("补充时间与原记录日期矛盾")
			}
		}
		var n int64
		if e = tx.Model(&model.HuntPlanTarget{}).Where("plan_id = ? AND shiny_record_id = ?", planID, r.ID).Count(&n).Error; e != nil {
			return e
		}
		if n > 0 {
			return conflict("同一个体已完成本卡其他格子")
		}
		if e = completeBingoTarget(tx, planID, target, r, *fact, "manual_timestamp", in.EvidenceNote, &in.ObtainedAt, now); e != nil {
			return e
		}
		if e = closeFullBingo(tx, planID); e != nil {
			return e
		}
		out, e = plannerView(tx, planID)
		return e
	})
	return out, e
}
