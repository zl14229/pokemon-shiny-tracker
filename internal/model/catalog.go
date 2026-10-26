package model

import "time"

// CatalogEntry enriches stable reference identities without rewriting personal data.
type CatalogEntry struct {
	Base
	Kind             string    `gorm:"not null;uniqueIndex:uidx_catalog_external,priority:1;check:kind IN ('species','form')" json:"kind"`
	ExternalID       int       `gorm:"not null;uniqueIndex:uidx_catalog_external,priority:2;check:external_id > 0" json:"external_id"`
	SpeciesID        uint64    `gorm:"not null;index" json:"species_id"`
	Species          Species   `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	FormID           *uint64   `gorm:"uniqueIndex;check:(kind='species' AND form_id IS NULL) OR (kind='form' AND form_id IS NOT NULL)" json:"form_id"`
	Form             *Form     `gorm:"foreignKey:FormID,SpeciesID;references:ID,SpeciesID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	NamesJSON        string    `gorm:"not null;check:json_valid(names_json)" json:"-"`
	NameStatus       string    `gorm:"not null;check:name_status IN ('unknown','source_provided','verified','conflict')" json:"name_status"`
	Inclusion        string    `gorm:"not null;check:inclusion IN ('included','excluded','unknown')" json:"inclusion"`
	Reason           string    `gorm:"not null" json:"reason"`
	SourceURL        string    `gorm:"not null" json:"source_url"`
	CrossCheckURL    string    `gorm:"not null" json:"cross_check_url"`
	CrossCheckStatus string    `gorm:"not null;default:unknown;check:cross_check_status IN ('unknown','verified','conflict')" json:"cross_check_status"`
	SourceRevision   string    `gorm:"not null" json:"source_revision"`
	SourceSHA256     string    `gorm:"not null" json:"source_sha256"`
	FetchedAt        time.Time `gorm:"not null" json:"fetched_at"`
	CheckedAt        time.Time `gorm:"not null" json:"checked_at"`
}
