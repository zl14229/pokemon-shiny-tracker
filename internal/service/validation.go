// Package service owns personal-record validation and transactional invariants.
package service

import (
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"gorm.io/gorm"
)

type Error struct{ Code, Message string }

func (e *Error) Error() string      { return e.Message }
func invalid(message string) error  { return &Error{"invalid_request", message} }
func conflict(message string) error { return &Error{"conflict", message} }
func validID(id uint64) bool        { return id > 0 && id <= math.MaxInt64 }
func member(value string, values ...string) bool {
	for _, v := range values {
		if value == v {
			return true
		}
	}
	return false
}
func text(value **string, name string, max int) error {
	if *value == nil {
		return nil
	}
	v := strings.TrimSpace(**value)
	if utf8.RuneCountInString(v) > max {
		return invalid(fmt.Sprintf("%s 超过 %d 字符", name, max))
	}
	if v == "" {
		*value = nil
	} else {
		*value = &v
	}
	return nil
}

// DateBounds retains user precision. Bounds are for validation, never stored as facts.
func DateBounds(value string) (time.Time, time.Time, error) {
	layout := ""
	switch len(value) {
	case 4:
		layout = "2006"
	case 7:
		layout = "2006-01"
	case 10:
		layout = "2006-01-02"
	}
	if layout == "" {
		return time.Time{}, time.Time{}, invalid("日期须为 YYYY、YYYY-MM 或 YYYY-MM-DD")
	}
	start, err := time.Parse(layout, value)
	if err != nil || start.Year() < 1 {
		return time.Time{}, time.Time{}, invalid("日期无效")
	}
	end := start
	if len(value) == 4 {
		end = time.Date(start.Year(), 12, 31, 0, 0, 0, 0, time.UTC)
	}
	if len(value) == 7 {
		end = start.AddDate(0, 1, -1)
	}
	return start, end, nil
}
func date(value *string) error {
	if value == nil {
		return nil
	}
	_, _, err := DateBounds(*value)
	return err
}

func references(db *gorm.DB, speciesID *uint64, formID *uint64, gameID *uint64, allowStorage bool) error {
	if speciesID != nil {
		if !validID(*speciesID) {
			return invalid("物种 ID 无效")
		}
		var n int64
		if err := db.Model(&model.Species{}).Where("id = ?", *speciesID).Count(&n).Error; err != nil {
			return err
		}
		if n != 1 {
			return invalid("物种不存在")
		}
	}
	if formID != nil {
		if speciesID == nil || !validID(*formID) {
			return invalid("形态必须关联有效物种")
		}
		var n int64
		if err := db.Model(&model.Form{}).Where("id = ? AND species_id = ?", *formID, *speciesID).Count(&n).Error; err != nil {
			return err
		}
		if n != 1 {
			return invalid("形态不存在或不属于该物种")
		}
	}
	if gameID != nil {
		if !validID(*gameID) {
			return invalid("游戏 ID 无效")
		}
		var game model.Game
		if err := db.First(&game, *gameID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return invalid("游戏不存在")
			}
			return err
		}
		if !allowStorage && game.Kind == "storage" {
			return invalid("保存位置不能作为普通 Hunt 游戏")
		}
	}
	return nil
}
