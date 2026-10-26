package model

import "time"

// ShinyAvailability is the minimum evidence contract consumed by the dex.
// M4 ships no game availability facts; detailed source/method ingestion is T026.
type ShinyAvailability struct {
	Base
	GameID             uint64     `gorm:"not null;uniqueIndex:uidx_availability,priority:1" json:"game_id"`
	Game               Game       `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	SpeciesID          uint64     `gorm:"not null" json:"species_id"`
	Species            Species    `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	FormID             uint64     `gorm:"not null;uniqueIndex:uidx_availability,priority:2" json:"form_id"`
	Form               Form       `gorm:"foreignKey:FormID,SpeciesID;references:ID,SpeciesID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	Status             string     `gorm:"not null;default:unknown;check:status IN ('confirmed_available','confirmed_locked','transfer_only','unknown')" json:"status"`
	VerificationStatus string     `gorm:"not null;default:unknown;check:verification_status IN ('verified','unknown','conflict')" json:"verification_status"`
	Source             string     `gorm:"not null" json:"source"`
	SourceURL          string     `gorm:"not null" json:"source_url"`
	CheckedAt          *time.Time `json:"checked_at"`
	DataVersion        string     `gorm:"not null;check:ck_availability_evidence,verification_status <> 'verified' OR (source <> '' AND source_url <> '' AND data_version <> '' AND checked_at IS NOT NULL)" json:"data_version"`
}
