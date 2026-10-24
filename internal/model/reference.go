package model

import "time"

type ReferenceBatch struct {
	Base
	Checksum     string `gorm:"not null;uniqueIndex" json:"checksum"`
	Source       string `gorm:"not null" json:"source"`
	SnapshotJSON string `gorm:"not null;check:json_valid(snapshot_json)" json:"-"`
}

type ExternalReference struct {
	Base
	Source             string         `gorm:"not null;uniqueIndex:uidx_external_identity,priority:1" json:"source"`
	Resource           string         `gorm:"not null;uniqueIndex:uidx_external_identity,priority:2;check:resource IN ('species','form','game')" json:"resource"`
	ExternalID         int            `gorm:"not null;uniqueIndex:uidx_external_identity,priority:3;check:external_id > 0" json:"external_id"`
	SpeciesID          *uint64        `json:"species_id"`
	Species            *Species       `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	FormID             *uint64        `json:"form_id"`
	Form               *Form          `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	GameID             *uint64        `json:"game_id"`
	Game               *Game          `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	BatchID            uint64         `gorm:"not null;index" json:"batch_id"`
	Batch              ReferenceBatch `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	SourceURL          string         `gorm:"not null" json:"source_url"`
	RawSHA256          string         `gorm:"not null" json:"raw_sha256"`
	FetchedAt          time.Time      `gorm:"not null" json:"fetched_at"`
	VerifiedAt         time.Time      `gorm:"not null" json:"verified_at"`
	VerificationStatus string         `gorm:"not null;default:validated;check:verification_status IN ('validated','unknown','conflict')" json:"verification_status"`
	DataJSON           string         `gorm:"not null;check:ck_external_target,json_valid(data_json) AND ((resource='species' AND species_id IS NOT NULL AND form_id IS NULL AND game_id IS NULL) OR (resource='form' AND form_id IS NOT NULL AND species_id IS NULL AND game_id IS NULL) OR (resource='game' AND game_id IS NOT NULL AND species_id IS NULL AND form_id IS NULL))" json:"-"`
}
