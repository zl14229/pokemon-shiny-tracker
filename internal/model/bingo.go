package model

import "time"

type BingoChallenge struct {
	Base
	PlanID       uint64    `gorm:"not null;uniqueIndex" json:"plan_id"`
	Plan         HuntPlan  `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	Size         int       `gorm:"not null;check:size IN (3,5)" json:"size"`
	StartedAt    time.Time `gorm:"not null" json:"started_at"`
	CalendarZone string    `gorm:"not null" json:"calendar_zone"`
}

// Events retain replaced targets and completion evidence without changing any
// personal identity. Optional request IDs make rerolls safely retryable.
type BingoEvent struct {
	Base
	PlanID     uint64          `gorm:"not null;index" json:"plan_id"`
	Plan       HuntPlan        `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	TargetID   *uint64         `json:"target_id"`
	Target     *HuntPlanTarget `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	Kind       string          `gorm:"not null;check:kind IN ('created','hunt_linked','rerolled','record_completed','completed','abandoned')" json:"kind"`
	Summary    string          `gorm:"not null" json:"summary"`
	RequestID  *string         `gorm:"uniqueIndex" json:"-"`
	BeforeJSON string          `gorm:"not null;check:json_valid(before_json)" json:"-"`
	AfterJSON  string          `gorm:"not null;check:json_valid(after_json)" json:"-"`
}
