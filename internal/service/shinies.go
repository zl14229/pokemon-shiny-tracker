package service

import (
	"context"
	"encoding/json"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ShinyInput struct {
	HuntID            *uint64 `json:"hunt_id"`
	OriginalSpeciesID *uint64 `json:"original_species_id"`
	OriginalFormID    *uint64 `json:"original_form_id"`
	CurrentSpeciesID  uint64  `json:"current_species_id"`
	CurrentFormID     *uint64 `json:"current_form_id"`
	OriginalGameID    *uint64 `json:"original_game_id"`
	CurrentGameID     *uint64 `json:"current_game_id"`
	CurrentLocation   *string `json:"current_location"`
	ObtainedDate      *string `json:"obtained_date"`
	Source            string  `json:"source"`
	HoldingStatus     string  `json:"holding_status"`
	Nickname          *string `json:"nickname"`
	Gender            *string `json:"gender"`
	Nature            *string `json:"nature"`
	Ability           *string `json:"ability"`
	Ball              *string `json:"ball"`
	Level             *int    `json:"level"`
	Notes             *string `json:"notes"`
	ChangeType        string  `json:"change_type"`
	EventDate         *string `json:"event_date"`
	HistoryNote       *string `json:"history_note"`
}

func (in *ShinyInput) validate(db *gorm.DB, create bool) error {
	if create {
		if in.Source == "" {
			in.Source = "unknown"
		}
		if in.HoldingStatus == "" {
			in.HoldingStatus = "unknown"
		}
		if in.ChangeType != "" {
			return invalid("创建时不能声明进化或纠错")
		}
	}
	if !member(in.Source, "self_hunted", "trade", "event", "other", "unknown") {
		return invalid("source 无效")
	}
	if !member(in.HoldingStatus, "held", "traded", "released", "unknown") {
		return invalid("holding_status 无效")
	}
	if !member(in.ChangeType, "", "evolution", "correction") {
		return invalid("change_type 须为空、evolution 或 correction")
	}
	for _, f := range []struct {
		p    **string
		name string
		max  int
	}{{&in.CurrentLocation, "current_location", 256}, {&in.Nickname, "nickname", 128}, {&in.Gender, "gender", 32}, {&in.Nature, "nature", 64}, {&in.Ability, "ability", 128}, {&in.Ball, "ball", 128}, {&in.Notes, "notes", 5000}, {&in.HistoryNote, "history_note", 1000}} {
		if err := text(f.p, f.name, f.max); err != nil {
			return err
		}
	}
	if in.Level != nil && (*in.Level < 1 || *in.Level > 100) {
		return invalid("level 须在 1 至 100 之间")
	}
	if err := date(in.ObtainedDate); err != nil {
		return err
	}
	if err := date(in.EventDate); err != nil {
		return err
	}
	if err := references(db, in.OriginalSpeciesID, in.OriginalFormID, in.OriginalGameID, true); err != nil {
		return err
	}
	if err := references(db, &in.CurrentSpeciesID, in.CurrentFormID, in.CurrentGameID, true); err != nil {
		return err
	}
	if in.HuntID != nil {
		if !validID(*in.HuntID) {
			return invalid("hunt_id 无效")
		}
		var n int64
		if err := db.Model(&model.Hunt{}).Where("id = ?", *in.HuntID).Count(&n).Error; err != nil {
			return err
		}
		if n != 1 {
			return invalid("关联 Hunt 不存在或已删除")
		}
	}
	// Distinct collection systems cannot describe a transfer of one individual.
	if in.OriginalGameID != nil && in.CurrentGameID != nil {
		var games []model.Game
		if err := db.Where("id IN ?", []uint64{*in.OriginalGameID, *in.CurrentGameID}).Find(&games).Error; err != nil {
			return err
		}
		if len(games) == 2 && games[0].CollectionSystem != "unknown" && games[1].CollectionSystem != "unknown" && games[0].CollectionSystem != games[1].CollectionSystem {
			return invalid("不同收藏体系不能作为同一个体的转移")
		}
	}
	return nil
}
func (in ShinyInput) row() model.ShinyRecord {
	return model.ShinyRecord{HuntID: in.HuntID, OriginalSpeciesID: in.OriginalSpeciesID, OriginalFormID: in.OriginalFormID, CurrentSpeciesID: in.CurrentSpeciesID, CurrentFormID: in.CurrentFormID, OriginalGameID: in.OriginalGameID, CurrentGameID: in.CurrentGameID, CurrentLocation: in.CurrentLocation, ObtainedDate: in.ObtainedDate, Source: in.Source, HoldingStatus: in.HoldingStatus, Nickname: in.Nickname, Gender: in.Gender, Nature: in.Nature, Ability: in.Ability, Ball: in.Ball, Level: in.Level, Notes: in.Notes}
}
func same[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
func snapshot(value any) (string, error) { b, err := json.Marshal(value); return string(b), err }
func history(tx *gorm.DB, rowID uint64, kind, before, after string, in ShinyInput) error {
	return tx.Omit(clause.Associations).Create(&model.ShinyHistory{ShinyRecordID: rowID, Kind: kind, BeforeJSON: before, AfterJSON: after, EventDate: in.EventDate, Note: in.HistoryNote}).Error
}
func protectOutcome(tx *gorm.DB, oldHunt, newHunt *uint64, recordID uint64) error {
	if oldHunt == nil || same(oldHunt, newHunt) {
		return nil
	}
	var h model.Hunt
	if err := tx.First(&h, *oldHunt).Error; err != nil {
		return err
	}
	if h.Status != "completed" {
		return nil
	}
	var n int64
	if err := tx.Model(&model.ShinyRecord{}).Where("hunt_id = ? AND id <> ?", *oldHunt, recordID).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return conflict("不能移除已完成 Hunt 的最后有效成果，请先重新打开或放弃 Hunt")
	}
	return nil
}

type Shinies struct{ DB *gorm.DB }

func (s Shinies) Get(ctx context.Context, id uint64) (model.ShinyRecord, error) {
	var row model.ShinyRecord
	err := s.DB.WithContext(ctx).First(&row, id).Error
	return row, err
}
func (s Shinies) List(ctx context.Context) ([]model.ShinyRecord, int64, error) {
	rows := []model.ShinyRecord{}
	var n int64
	db := s.DB.WithContext(ctx)
	if err := db.Model(&model.ShinyRecord{}).Count(&n).Error; err != nil {
		return rows, 0, err
	}
	err := db.Order("id ASC").Limit(100).Find(&rows).Error
	return rows, n, err
}
func (s Shinies) Create(ctx context.Context, in ShinyInput) (model.ShinyRecord, error) {
	var row model.ShinyRecord
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := in.validate(tx, true); err != nil {
			return err
		}
		row = in.row()
		if err := tx.Omit(clause.Associations).Create(&row).Error; err != nil {
			return err
		}
		after, err := snapshot(row)
		if err != nil {
			return err
		}
		return history(tx, row.ID, "registered", "null", after, in)
	})
	return row, err
}
func (s Shinies) Update(ctx context.Context, id uint64, in ShinyInput) (model.ShinyRecord, error) {
	var row model.ShinyRecord
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&row, id).Error; err != nil {
			return err
		}
		if err := in.validate(tx, false); err != nil {
			return err
		}
		speciesChanged := row.CurrentSpeciesID != in.CurrentSpeciesID
		if speciesChanged && in.ChangeType == "" {
			return invalid("当前物种变化须声明 change_type=evolution 或 correction")
		}
		if !speciesChanged && in.ChangeType == "evolution" {
			return invalid("进化事件须改变当前物种")
		}
		if err := protectOutcome(tx, row.HuntID, in.HuntID, id); err != nil {
			return err
		}
		replacement := in.row()
		replacement.Base = row.Base
		before, err := snapshot(row)
		if err != nil {
			return err
		}
		predicted, err := snapshot(replacement)
		if err != nil {
			return err
		}
		if before == predicted {
			return nil
		}
		kinds := []string{}
		if speciesChanged {
			kind := "updated"
			if in.ChangeType == "evolution" {
				kind = "evolution"
			}
			kinds = append(kinds, kind)
		} else if !same(row.CurrentFormID, in.CurrentFormID) {
			kinds = append(kinds, "form_change")
		}
		if !same(row.CurrentGameID, in.CurrentGameID) || !same(row.CurrentLocation, in.CurrentLocation) {
			kinds = append(kinds, "location_change")
		}
		if row.HoldingStatus != in.HoldingStatus {
			kinds = append(kinds, "holding_change")
		}
		if len(kinds) == 0 {
			kinds = append(kinds, "updated")
		}
		if err := tx.Model(&row).Select("HuntID", "OriginalSpeciesID", "OriginalFormID", "CurrentSpeciesID", "CurrentFormID", "OriginalGameID", "CurrentGameID", "CurrentLocation", "ObtainedDate", "Source", "HoldingStatus", "Nickname", "Gender", "Nature", "Ability", "Ball", "Level", "Notes").Omit(clause.Associations).Updates(&replacement).Error; err != nil {
			return err
		}
		if err := tx.First(&row, id).Error; err != nil {
			return err
		}
		after, err := snapshot(row)
		if err != nil {
			return err
		}
		for _, kind := range kinds {
			if err := history(tx, id, kind, before, after, in); err != nil {
				return err
			}
		}
		return nil
	})
	return row, err
}
func (s Shinies) Delete(ctx context.Context, id uint64) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.ShinyRecord
		if err := tx.First(&row, id).Error; err != nil {
			return err
		}
		if err := protectOutcome(tx, row.HuntID, nil, id); err != nil {
			return err
		}
		before, err := snapshot(row)
		if err != nil {
			return err
		}
		if err := tx.Delete(&row).Error; err != nil {
			return err
		}
		return history(tx, id, "deleted", before, "null", ShinyInput{})
	})
}
func (s Shinies) History(ctx context.Context, id uint64, limit, offset int) ([]model.ShinyHistory, bool, int64, error) {
	rows := []model.ShinyHistory{}
	var deleted bool
	var n int64
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.ShinyRecord
		if err := tx.Unscoped().First(&row, id).Error; err != nil {
			return err
		}
		deleted = row.DeletedAt.Valid
		q := tx.Model(&model.ShinyHistory{}).Where("shiny_record_id = ?", id)
		if err := q.Count(&n).Error; err != nil {
			return err
		}
		return q.Order("id ASC").Limit(limit).Offset(offset).Find(&rows).Error
	})
	return rows, deleted, n, err
}
