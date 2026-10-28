package model

// Plans retain their generation rules and source snapshot independently of later
// reference updates. A draft is never a Hunt and is absent from saved history.
type HuntPlan struct {
	Base
	RequestID    string `gorm:"not null;uniqueIndex" json:"-"`
	Kind         string `gorm:"not null;check:kind IN ('random','bingo')" json:"kind"`
	Status       string `gorm:"not null;check:status IN ('draft','active','completed','abandoned')" json:"status"`
	RulesJSON    string `gorm:"not null;check:json_valid(rules_json)" json:"-"`
	SnapshotJSON string `gorm:"not null;check:json_valid(snapshot_json)" json:"-"`
}
type HuntPlanTarget struct {
	Base
	PlanID        uint64       `gorm:"not null;uniqueIndex:uidx_plan_position,priority:1;uniqueIndex:uidx_plan_target,priority:1" json:"plan_id"`
	Plan          HuntPlan     `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	Position      int          `gorm:"not null;uniqueIndex:uidx_plan_position,priority:2;check:position > 0" json:"position"`
	TargetKey     string       `gorm:"not null;uniqueIndex:uidx_plan_target,priority:2" json:"target_key"`
	SpeciesID     uint64       `gorm:"not null" json:"species_id"`
	Species       Species      `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	FormID        *uint64      `json:"form_id"`
	Form          *Form        `gorm:"foreignKey:FormID,SpeciesID;references:ID,SpeciesID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	GameID        uint64       `gorm:"not null" json:"game_id"`
	Game          Game         `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	Status        string       `gorm:"not null;check:status IN ('pending','started','completed')" json:"status"`
	HuntID        *uint64      `json:"hunt_id"`
	Hunt          *Hunt        `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	ShinyRecordID *uint64      `json:"shiny_record_id"`
	ShinyRecord   *ShinyRecord `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	SnapshotJSON  string       `gorm:"not null;check:json_valid(snapshot_json)" json:"-"`
	ProofJSON     string       `gorm:"not null;default:'{}';check:json_valid(proof_json)" json:"-"`
}
