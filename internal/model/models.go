// Package model contains stable local identities; external source IDs are separate.
package model

import (
	"time"

	"gorm.io/gorm"
)

type Base struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Labels struct {
	Slug       string  `gorm:"not null;uniqueIndex;check:slug <> ''" json:"slug"`
	Name       string  `gorm:"not null" json:"name"`
	NameZh     *string `json:"name_zh"`
	NameStatus string  `gorm:"not null;default:unknown;check:name_status IN ('unknown','source_provided','verified','conflict')" json:"name_status"`
}

type Game struct {
	Base
	Labels
	Kind             string `gorm:"not null;default:unknown;check:kind IN ('mainline','spin_off','storage','other','unknown')" json:"kind"`
	CollectionSystem string `gorm:"not null;default:pokemon;check:collection_system <> ''" json:"collection_system"`
}

type Species struct {
	Base
	Labels
	NationalDex int `gorm:"not null;uniqueIndex;check:national_dex > 0" json:"national_dex"`
}

type Form struct {
	Base
	Labels
	SpeciesID uint64  `gorm:"not null;index" json:"species_id"`
	Species   Species `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
}

type Hunt struct {
	Base
	SpeciesID       uint64         `gorm:"not null;index" json:"species_id"`
	Species         Species        `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	FormID          *uint64        `json:"form_id"`
	Form            *Form          `gorm:"foreignKey:FormID,SpeciesID;references:ID,SpeciesID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	GameID          *uint64        `gorm:"index" json:"game_id"`
	Game            *Game          `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	Method          *string        `json:"method"`
	StartDate       *string        `json:"start_date"`
	EndDate         *string        `json:"end_date"`
	EncounterCount  *int64         `gorm:"check:encounter_count IS NULL OR (encounter_count >= 0 AND count_unit IS NOT NULL AND length(trim(count_unit)) > 0)" json:"encounter_count"`
	CountUnit       *string        `json:"count_unit"`
	DurationSeconds *int64         `gorm:"check:duration_seconds IS NULL OR duration_seconds >= 0" json:"duration_seconds"`
	Status          string         `gorm:"not null;default:in_progress;index;check:status IN ('in_progress','completed','abandoned')" json:"status"`
	Notes           *string        `json:"notes"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

type ShinyRecord struct {
	Base
	HuntID            *uint64        `gorm:"index" json:"hunt_id"`
	Hunt              *Hunt          `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	OriginalSpeciesID *uint64        `json:"original_species_id"`
	OriginalSpecies   *Species       `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	OriginalFormID    *uint64        `gorm:"check:original_form_id IS NULL OR original_species_id IS NOT NULL" json:"original_form_id"`
	OriginalForm      *Form          `gorm:"foreignKey:OriginalFormID,OriginalSpeciesID;references:ID,SpeciesID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	CurrentSpeciesID  uint64         `gorm:"not null;index" json:"current_species_id"`
	CurrentSpecies    Species        `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	CurrentFormID     *uint64        `json:"current_form_id"`
	CurrentForm       *Form          `gorm:"foreignKey:CurrentFormID,CurrentSpeciesID;references:ID,SpeciesID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	OriginalGameID    *uint64        `gorm:"index" json:"original_game_id"`
	OriginalGame      *Game          `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	CurrentGameID     *uint64        `gorm:"index" json:"current_game_id"`
	CurrentGame       *Game          `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	CurrentLocation   *string        `json:"current_location"`
	ObtainedDate      *string        `json:"obtained_date"`
	Source            string         `gorm:"not null;default:unknown;index;check:source IN ('self_hunted','trade','event','other','unknown')" json:"source"`
	HoldingStatus     string         `gorm:"not null;default:held;index;check:holding_status IN ('held','traded','released','unknown')" json:"holding_status"`
	Nickname          *string        `json:"nickname"`
	Gender            *string        `json:"gender"`
	Nature            *string        `json:"nature"`
	Ability           *string        `json:"ability"`
	Ball              *string        `json:"ball"`
	Level             *int           `gorm:"check:level IS NULL OR level BETWEEN 1 AND 100" json:"level"`
	Notes             *string        `json:"notes"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

type ShinyHistory struct {
	Base
	ShinyRecordID uint64      `gorm:"not null;index" json:"shiny_record_id"`
	ShinyRecord   ShinyRecord `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	Kind          string      `gorm:"not null;check:kind IN ('registered','evolution','form_change','location_change','holding_change','updated','deleted')" json:"kind"`
	BeforeJSON    string      `gorm:"not null;check:json_valid(before_json)" json:"before_json"`
	AfterJSON     string      `gorm:"not null;check:json_valid(after_json)" json:"after_json"`
	EventDate     *string     `json:"event_date"`
	Note          *string     `json:"note"`
}
