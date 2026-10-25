package model

import "gorm.io/gorm"

type Attachment struct {
	Base
	ShinyRecordID uint64         `gorm:"not null;index" json:"shiny_record_id"`
	ShinyRecord   ShinyRecord    `gorm:"constraint:OnDelete:RESTRICT,OnUpdate:RESTRICT" json:"-"`
	Kind          string         `gorm:"not null;check:kind IN ('file','link')" json:"kind"`
	Name          string         `gorm:"not null" json:"name"`
	MediaType     string         `gorm:"not null" json:"media_type"`
	StorageKey    *string        `gorm:"uniqueIndex;check:(kind = 'file' AND storage_key IS NOT NULL AND storage_key <> '' AND sha256 IS NOT NULL AND sha256 <> '' AND size_bytes IS NOT NULL AND size_bytes > 0 AND media_type <> '' AND url IS NULL) OR (kind = 'link' AND storage_key IS NULL AND sha256 IS NULL AND size_bytes IS NULL AND media_type = '' AND url IS NOT NULL AND url <> '')" json:"-"`
	SHA256        *string        `json:"sha256"`
	SizeBytes     *int64         `json:"size_bytes"`
	URL           *string        `json:"url"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}
