package service

import (
	"context"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type HuntInput struct {
	SpeciesID       uint64  `json:"species_id"`
	FormID          *uint64 `json:"form_id"`
	GameID          *uint64 `json:"game_id"`
	Method          *string `json:"method"`
	StartDate       *string `json:"start_date"`
	EndDate         *string `json:"end_date"`
	EncounterCount  *int64  `json:"encounter_count"`
	CountUnit       *string `json:"count_unit"`
	DurationSeconds *int64  `json:"duration_seconds"`
	Status          string  `json:"status"`
	Notes           *string `json:"notes"`
}

func (in *HuntInput) validate(db *gorm.DB, create bool) error {
	if create && in.Status == "" {
		in.Status = "in_progress"
	}
	if !member(in.Status, "in_progress", "completed", "abandoned") {
		return invalid("status 必须为 in_progress、completed 或 abandoned")
	}
	for _, field := range []struct {
		p    **string
		name string
		max  int
	}{{&in.Method, "method", 128}, {&in.CountUnit, "count_unit", 32}, {&in.Notes, "notes", 5000}} {
		if err := text(field.p, field.name, field.max); err != nil {
			return err
		}
	}
	if in.EncounterCount != nil && (*in.EncounterCount < 0 || in.CountUnit == nil) {
		return invalid("次数须为非负整数并提供 count_unit")
	}
	if in.DurationSeconds != nil && *in.DurationSeconds < 0 {
		return invalid("duration_seconds 须为非负整数")
	}
	if err := date(in.StartDate); err != nil {
		return err
	}
	if err := date(in.EndDate); err != nil {
		return err
	}
	if in.StartDate != nil && in.EndDate != nil {
		start, _, _ := DateBounds(*in.StartDate)
		_, end, _ := DateBounds(*in.EndDate)
		if start.After(end) {
			return invalid("起止日期范围倒置")
		}
	}
	return references(db, &in.SpeciesID, in.FormID, in.GameID, false)
}
func (in HuntInput) row() model.Hunt {
	return model.Hunt{SpeciesID: in.SpeciesID, FormID: in.FormID, GameID: in.GameID, Method: in.Method, StartDate: in.StartDate, EndDate: in.EndDate, EncounterCount: in.EncounterCount, CountUnit: in.CountUnit, DurationSeconds: in.DurationSeconds, Status: in.Status, Notes: in.Notes}
}

type Hunts struct{ DB *gorm.DB }

func (s Hunts) Get(ctx context.Context, id uint64) (model.Hunt, error) {
	var row model.Hunt
	err := s.DB.WithContext(ctx).First(&row, id).Error
	return row, err
}
func (s Hunts) Create(ctx context.Context, in HuntInput) (model.Hunt, error) {
	var row model.Hunt
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := in.validate(tx, true); err != nil {
			return err
		}
		if in.Status == "completed" {
			return conflict("Hunt 至少关联一个有效成果后才能完成")
		}
		row = in.row()
		return tx.Omit(clause.Associations).Create(&row).Error
	})
	return row, err
}
func (s Hunts) Update(ctx context.Context, id uint64, in HuntInput) (model.Hunt, error) {
	var row model.Hunt
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&row, id).Error; err != nil {
			return err
		}
		if err := in.validate(tx, false); err != nil {
			return err
		}
		if in.Status == "completed" {
			var n int64
			if err := tx.Model(&model.ShinyRecord{}).Where("hunt_id = ?", id).Count(&n).Error; err != nil {
				return err
			}
			if n == 0 {
				return conflict("Hunt 至少关联一个有效成果后才能完成")
			}
		}
		replacement := in.row()
		replacement.Base = row.Base
		if err := tx.Model(&row).Select("SpeciesID", "FormID", "GameID", "Method", "StartDate", "EndDate", "EncounterCount", "CountUnit", "DurationSeconds", "Status", "Notes").Omit(clause.Associations).Updates(&replacement).Error; err != nil {
			return err
		}
		return tx.First(&row, id).Error
	})
	return row, err
}
func (s Hunts) Delete(ctx context.Context, id uint64) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.Hunt
		if err := tx.First(&row, id).Error; err != nil {
			return err
		}
		var n int64
		if err := tx.Unscoped().Model(&model.ShinyRecord{}).Where("hunt_id = ?", id).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return conflict("Hunt 仍有关联个体或历史，请保留该 Hunt")
		}
		return tx.Delete(&row).Error
	})
}
