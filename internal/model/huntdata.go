package model

// HuntDataBatch is an immutable, reviewed reference snapshot. Rolling it back
// removes only its additions; identities and personal data are never rewritten.
type HuntDataBatch struct {
	Base
	Version      string `gorm:"not null;uniqueIndex" json:"version"`
	Checksum     string `gorm:"not null" json:"checksum"`
	SnapshotJSON string `gorm:"not null;check:json_valid(snapshot_json)" json:"-"`
	State        string `gorm:"not null;check:state IN ('applied','rolled_back')" json:"state"`
}

type HuntFact struct {
	Base
	AvailabilityID uint64            `gorm:"not null;uniqueIndex" json:"availability_id"`
	Availability   ShinyAvailability `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	BatchID        uint64            `gorm:"not null;index" json:"batch_id"`
	Batch          HuntDataBatch     `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	DirectHunt     string            `gorm:"not null;check:direct_hunt IN ('yes','no','unknown')" json:"direct_hunt"`
	GuideStatus    string            `gorm:"not null;check:guide_status IN ('complete','partial','unknown')" json:"guide_status"`
	TargetJSON     string            `gorm:"not null;check:json_valid(target_json)" json:"-"`
}

type PokemonHuntMethod struct {
	Base
	HuntFactID        uint64   `gorm:"not null;index" json:"hunt_fact_id"`
	HuntFact          HuntFact `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	Name              string   `gorm:"not null" json:"name"`
	Location          string   `gorm:"not null" json:"location"`
	Conditions        string   `gorm:"not null" json:"conditions"`
	EvidenceJSON      string   `gorm:"not null;check:json_valid(evidence_json)" json:"-"`
	EncounterRateJSON *string  `gorm:"check:encounter_rate_json IS NULL OR json_valid(encounter_rate_json)" json:"-"`
	ShinyRateJSON     *string  `gorm:"check:shiny_rate_json IS NULL OR json_valid(shiny_rate_json)" json:"-"`
}
