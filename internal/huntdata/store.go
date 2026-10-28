package huntdata

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"gorm.io/gorm"
)

type Difference struct {
	GameVersionID int    `json:"game_version_id"`
	FormID        int    `json:"form_id"`
	Action        string `json:"action"`
	After         Target `json:"after"`
}
type Report struct {
	Checksum    string       `json:"checksum"`
	Differences []Difference `json:"differences"`
}

func encode(v any) string     { b, _ := json.Marshal(v); return string(b) }
func Digest(d Dataset) string { h := sha256.Sum256([]byte(encode(d))); return hex.EncodeToString(h[:]) }
func mapping(tx *gorm.DB, resource string, id int) (uint64, error) {
	var ref model.ExternalReference
	if e := tx.Where("source = ? AND resource = ? AND external_id = ? ", "pokeapi-v2", resource, id).First(&ref).Error; e != nil {
		if errors.Is(e, gorm.ErrRecordNotFound) && resource != "game" {
			var c model.CatalogEntry
			if err := tx.Where("kind = ? AND external_id = ?", resource, id).First(&c).Error; err == nil {
				if resource == "species" {
					return c.SpeciesID, nil
				}
				if c.FormID != nil {
					return *c.FormID, nil
				}
			}
		}
		return 0, fmt.Errorf("missing validated %s mapping %d: %w", resource, id, e)
	}
	if ref.VerificationStatus != "validated" {
		return 0, errors.New("conflicting or unverified external mapping")
	}
	switch resource {
	case "game":
		if ref.GameID != nil {
			return *ref.GameID, nil
		}
	case "species":
		if ref.SpeciesID != nil {
			return *ref.SpeciesID, nil
		}
	case "form":
		if ref.FormID != nil {
			return *ref.FormID, nil
		}
	}
	return 0, errors.New("invalid mapping")
}
func mapTarget(tx *gorm.DB, t Target) (model.ShinyAvailability, error) {
	a := model.ShinyAvailability{Status: t.Availability, VerificationStatus: t.Verification, Source: t.Evidence.Source, SourceURL: t.Evidence.URL, DataVersion: t.Evidence.Version}
	if !t.Evidence.CheckedAt.IsZero() {
		v := t.Evidence.CheckedAt
		a.CheckedAt = &v
	}
	var e error
	if a.GameID, e = mapping(tx, "game", t.GameVersionID); e != nil {
		return a, e
	}
	if a.SpeciesID, e = mapping(tx, "species", t.SpeciesID); e != nil {
		return a, e
	}
	if a.FormID, e = mapping(tx, "form", t.FormID); e != nil {
		return a, e
	}
	var n int64
	if e = tx.Model(&model.Form{}).Where("id = ? AND species_id = ?", a.FormID, a.SpeciesID).Count(&n).Error; e != nil {
		return a, e
	}
	if n != 1 {
		return a, errors.New("form/species mapping mismatch")
	}
	var g model.Game
	if e = tx.First(&g, a.GameID).Error; e != nil {
		return a, e
	}
	if g.Kind == "storage" {
		return a, errors.New("storage is not a hunt game")
	}
	return a, nil
}

// Import is deliberately add-only. Reviewed replacements and upstream syncing
// belong to T024; an overlapping existing claim requires explicit resolution.
// Preview executes the identical transactional path and always rolls it back.
func Import(db *gorm.DB, d Dataset, apply bool, reviewedSHA string) (Report, error) {
	report := Report{Checksum: Digest(d), Differences: []Difference{}}
	if e := d.Validate(); e != nil {
		return report, e
	}
	if len(d.Targets) == 0 {
		return report, errors.New("empty staged batch")
	}
	if apply && reviewedSHA != report.Checksum {
		return report, errors.New("apply requires the reviewed preview checksum")
	}
	preview := errors.New("preview rollback")
	e := db.Transaction(func(tx *gorm.DB) error {
		var prior model.HuntDataBatch
		e := tx.Where("version = ?", d.Version).First(&prior).Error
		if e == nil {
			if prior.Checksum != report.Checksum || prior.State != "applied" {
				return errors.New("batch version reused or rolled back")
			}
			_, err := Read(tx)
			return err
		}
		if !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		batch := model.HuntDataBatch{Version: d.Version, Checksum: report.Checksum, SnapshotJSON: encode(d), State: "applied"}
		if e := tx.Create(&batch).Error; e != nil {
			return e
		}
		for _, t := range d.Targets {
			a, e := mapTarget(tx, t)
			if e != nil {
				return e
			}
			var n int64
			if e = tx.Model(&model.ShinyAvailability{}).Where("game_id = ? AND form_id = ?", a.GameID, a.FormID).Count(&n).Error; e != nil {
				return e
			}
			if n > 0 {
				return errors.New("existing availability conflicts; no overwrite performed")
			}
			if e = tx.Omit("Game", "Species", "Form").Create(&a).Error; e != nil {
				return e
			}
			fact := model.HuntFact{AvailabilityID: a.ID, BatchID: batch.ID, DirectHunt: t.DirectHunt, GuideStatus: t.GuideStatus, TargetJSON: encode(t)}
			if e = tx.Omit("Availability", "Batch").Create(&fact).Error; e != nil {
				return e
			}
			for _, m := range t.Methods {
				row := model.PokemonHuntMethod{HuntFactID: fact.ID, Name: m.Name, Location: m.Location, Conditions: m.Conditions, EvidenceJSON: encode(m.Evidence)}
				if m.EncounterRate != nil {
					v := encode(m.EncounterRate)
					row.EncounterRateJSON = &v
				}
				if m.ShinyRate != nil {
					v := encode(m.ShinyRate)
					row.ShinyRateJSON = &v
				}
				if e = tx.Omit("HuntFact").Create(&row).Error; e != nil {
					return e
				}
			}
			report.Differences = append(report.Differences, Difference{t.GameVersionID, t.FormID, "add", t})
		}
		if !apply {
			return preview
		}
		return nil
	})
	if errors.Is(e, preview) {
		e = nil
	}
	return report, e
}

type Row struct {
	ID           uint64 `json:"id"`
	GameID       uint64 `json:"game_id"`
	SpeciesID    uint64 `json:"species_id"`
	FormID       uint64 `json:"form_id"`
	BatchVersion string `json:"batch_version"`
	Fact         Target `json:"fact"`
}

// Read returns only applied batches whose stable mappings, evidence and mirrored
// availability still agree. A damaged reference set fails closed.
func Read(db *gorm.DB) ([]Row, error) {
	rows := []Row{}
	var facts []model.HuntFact
	if e := db.Order("id").Find(&facts).Error; e != nil {
		return nil, e
	}
	for _, f := range facts {
		var b model.HuntDataBatch
		if e := db.First(&b, f.BatchID).Error; e != nil {
			return nil, e
		}
		if b.State != "applied" {
			continue
		}
		var snapshot Dataset
		if e := json.Unmarshal([]byte(b.SnapshotJSON), &snapshot); e != nil {
			return nil, e
		}
		if snapshot.Version != b.Version || Digest(snapshot) != b.Checksum {
			return nil, errors.New("batch snapshot checksum mismatch")
		}
		belongs := false
		for _, expected := range snapshot.Targets {
			if encode(expected) == f.TargetJSON {
				belongs = true
				break
			}
		}
		if !belongs {
			return nil, errors.New("fact differs from reviewed snapshot")
		}

		var t Target
		if e := json.Unmarshal([]byte(f.TargetJSON), &t); e != nil {
			return nil, e
		}
		if e := (Dataset{Version: b.Version, Targets: []Target{t}}).Validate(); e != nil {
			return nil, e
		}
		a, e := mapTarget(db, t)
		if e != nil {
			return nil, e
		}
		var stored model.ShinyAvailability
		if e = db.First(&stored, f.AvailabilityID).Error; e != nil {
			return nil, e
		}
		a.Base = stored.Base
		if encode(a) != encode(stored) || f.DirectHunt != t.DirectHunt || f.GuideStatus != t.GuideStatus {
			return nil, errors.New("reference evidence mismatch")
		}
		rows = append(rows, Row{f.ID, a.GameID, a.SpeciesID, a.FormID, b.Version, t})
	}
	return rows, nil
}

// Rollback retracts a batch's reference additions, preserving its immutable audit
// snapshot. It refuses to remove facts changed after application.
func Rollback(db *gorm.DB, version, checksum string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var b model.HuntDataBatch
		if e := tx.Where("version = ?", version).First(&b).Error; e != nil {
			return e
		}
		if b.Checksum != checksum {
			return errors.New("rollback checksum mismatch")
		}
		if b.State == "rolled_back" {
			return nil
		}
		if _, e := Read(tx); e != nil {
			return e
		}
		var facts []model.HuntFact
		if e := tx.Where("batch_id = ?", b.ID).Find(&facts).Error; e != nil {
			return e
		}
		for _, f := range facts {
			if e := tx.Where("hunt_fact_id = ?", f.ID).Delete(&model.PokemonHuntMethod{}).Error; e != nil {
				return e
			}
			if e := tx.Delete(&f).Error; e != nil {
				return e
			}
			if e := tx.Delete(&model.ShinyAvailability{}, f.AvailabilityID).Error; e != nil {
				return e
			}
		}
		return tx.Model(&b).Update("state", "rolled_back").Error
	})
}
