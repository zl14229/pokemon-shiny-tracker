package model

// Confirmed batches retain provenance; unconfirmed previews are memory-only.
type ImportBatch struct {
	Base
	Token        string `gorm:"not null;uniqueIndex" json:"token"`
	FileSHA256   string `gorm:"not null" json:"file_sha256"`
	Sheet        string `gorm:"not null" json:"sheet"`
	MappingJSON  string `gorm:"not null;check:json_valid(mapping_json)" json:"mapping_json"`
	SelectedJSON string `gorm:"not null;check:json_valid(selected_json)" json:"selected_json"`
}
type ImportRow struct {
	Base
	ImportBatchID uint64      `gorm:"not null;index;uniqueIndex:uidx_import_row" json:"import_batch_id"`
	ImportBatch   ImportBatch `gorm:"constraint:OnDelete:RESTRICT" json:"-"`
	RowNumber     int         `gorm:"not null;uniqueIndex:uidx_import_row" json:"row_number"`
	ShinyRecordID uint64      `gorm:"not null;uniqueIndex" json:"shiny_record_id"`
	ShinyRecord   ShinyRecord `gorm:"constraint:OnDelete:RESTRICT" json:"-"`
	RawJSON       string      `gorm:"not null;check:json_valid(raw_json)" json:"raw_json"`
}
