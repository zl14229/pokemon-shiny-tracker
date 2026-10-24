package reference

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"gorm.io/gorm"
)

type Difference struct {
	Kind       string `json:"kind"`
	ExternalID int    `json:"external_id"`
	Before     string `json:"before"`
	After      string `json:"after"`
}
type Report struct {
	Added       int          `json:"added"`
	Unchanged   int          `json:"unchanged"`
	Differences []Difference `json:"differences"`
	Applied     bool         `json:"applied"`
}
type ConflictError struct{ Differences []Difference }

func (e *ConflictError) Error() string {
	return "reference differences require review; no data applied"
}

func labelsModel(f Facts) model.Labels {
	return model.Labels{Slug: f.Slug, Name: f.Name, NameZh: f.NameZh, NameStatus: f.NameStatus}
}

// Evidence refresh timestamps/hashes do not change a local entity's identity/facts.
func factsJSON(f Facts) []byte {
	f.PokemonSHA256 = ""
	f.PokemonFetchedAt = nil
	data, _ := json.Marshal(f)
	return data
}

func localDifference(m model.ExternalReference, before, after string) error {
	return &ConflictError{Differences: []Difference{{m.Resource, m.ExternalID, before, after}}}
}

// Import is add-only in M1. Changed reference facts require the later T024 review flow.
// A dry run validates all inserts/constraints inside a transaction and rolls it back.
func Import(db *gorm.DB, d Dataset, apply bool) (Report, error) {
	report := Report{Differences: []Difference{}}
	if err := d.Validate(); err != nil {
		return report, err
	}
	data := d.Canonical()
	checksum := Hash(data)
	previewRollback := errors.New("preview rollback")
	err := db.Transaction(func(tx *gorm.DB) error {
		var pending []Resource
		for _, r := range d.Resources {
			var mapping model.ExternalReference
			err := tx.Where("source = ? AND resource = ? AND external_id = ?", Source, r.Kind, r.ExternalID).First(&mapping).Error
			normalized := factsJSON(r.Facts)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				pending = append(pending, r)
				continue
			}
			if err != nil {
				return err
			}
			if mapping.DataJSON != string(normalized) {
				report.Differences = append(report.Differences, Difference{r.Kind, r.ExternalID, mapping.DataJSON, string(normalized)})
			} else {
				// Detect local reference edits as well as upstream snapshot differences.
				if err := checkLocal(tx, mapping, r.Facts); err != nil {
					var conflict *ConflictError
					if !errors.As(err, &conflict) {
						return err
					}
					report.Differences = append(report.Differences, conflict.Differences...)
				} else {
					report.Unchanged++
				}
			}
		}
		if len(report.Differences) > 0 {
			return &ConflictError{Differences: report.Differences}
		}
		if len(pending) == 0 {
			return nil
		}
		batch := model.ReferenceBatch{Checksum: checksum, Source: Source, SnapshotJSON: string(data)}
		if err := tx.Where("checksum = ?", checksum).FirstOrCreate(&batch).Error; err != nil {
			return err
		}
		// Species/game before forms; input order cannot affect parent mapping validity.
		ordered := append([]Resource(nil), pending...)
		for _, kind := range []string{"species", "game", "form"} {
			for _, r := range ordered {
				if r.Kind != kind {
					continue
				}
				if err := checkUnmappedIdentity(tx, r); err != nil {
					return err
				}
				normalized := factsJSON(r.Facts)
				mapping := model.ExternalReference{Source: Source, Resource: r.Kind, ExternalID: r.ExternalID, BatchID: batch.ID, SourceURL: r.SourceURL, RawSHA256: r.RawSHA256, FetchedAt: r.FetchedAt, VerifiedAt: time.Now().UTC(), VerificationStatus: "validated", DataJSON: string(normalized)}
				switch r.Kind {
				case "species":
					row := model.Species{Labels: labelsModel(r.Facts), NationalDex: r.Facts.NationalDex}
					if err := tx.Create(&row).Error; err != nil {
						return fmt.Errorf("local species identity conflict; no data applied: %w", err)
					}
					mapping.SpeciesID = &row.ID
				case "game":
					row := model.Game{Labels: labelsModel(r.Facts), Kind: "unknown"}
					if err := tx.Create(&row).Error; err != nil {
						return fmt.Errorf("local game identity conflict; no data applied: %w", err)
					}
					mapping.GameID = &row.ID
				case "form":
					var parent model.ExternalReference
					if err := tx.Where("source = ? AND resource = ? AND external_id = ?", Source, "species", r.Facts.SpeciesExternalID).First(&parent).Error; err != nil {
						return errors.New("form parent species has no validated local mapping")
					}
					if parent.SpeciesID == nil {
						return errors.New("invalid parent mapping")
					}
					row := model.Form{Labels: labelsModel(r.Facts), SpeciesID: *parent.SpeciesID}
					if err := tx.Create(&row).Error; err != nil {
						return fmt.Errorf("local form identity conflict; no data applied: %w", err)
					}
					mapping.FormID = &row.ID
				}
				if err := tx.Create(&mapping).Error; err != nil {
					return err
				}
				report.Added++
			}
		}
		if !apply {
			return previewRollback
		}
		return nil
	})
	if errors.Is(err, previewRollback) {
		return report, nil
	}
	if err != nil {
		var conflict *ConflictError
		if errors.As(err, &conflict) {
			report.Differences = conflict.Differences
		}
		return report, err
	}
	report.Applied = apply
	return report, nil
}

func checkUnmappedIdentity(db *gorm.DB, r Resource) error {
	var before any
	var err error
	switch r.Kind {
	case "species":
		var row model.Species
		err = db.Where("slug = ? OR national_dex = ?", r.Facts.Slug, r.Facts.NationalDex).First(&row).Error
		before = row
	case "form":
		var row model.Form
		err = db.Where("slug = ?", r.Facts.Slug).First(&row).Error
		before = row
	case "game":
		var row model.Game
		err = db.Where("slug = ?", r.Facts.Slug).First(&row).Error
		before = row
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	encoded, _ := json.Marshal(before)
	return &ConflictError{Differences: []Difference{{r.Kind, r.ExternalID, string(encoded), string(factsJSON(r.Facts))}}}
}

func checkLocal(db *gorm.DB, m model.ExternalReference, f Facts) error {
	want := labelsModel(f)
	var got model.Labels
	switch m.Resource {
	case "species":
		var row model.Species
		if m.SpeciesID == nil {
			return errors.New("invalid species target")
		}
		if err := db.First(&row, *m.SpeciesID).Error; err != nil {
			return err
		}
		if row.NationalDex != f.NationalDex {
			return localDifference(m, fmt.Sprintf("national_dex=%d", row.NationalDex), fmt.Sprintf("national_dex=%d", f.NationalDex))
		}
		got = row.Labels
	case "game":
		var row model.Game
		if m.GameID == nil {
			return errors.New("invalid game target")
		}
		if err := db.First(&row, *m.GameID).Error; err != nil {
			return err
		}
		got = row.Labels
	case "form":
		var row model.Form
		if m.FormID == nil {
			return errors.New("invalid form target")
		}
		if err := db.First(&row, *m.FormID).Error; err != nil {
			return err
		}
		got = row.Labels
		var parent model.ExternalReference
		if err := db.Where("source = ? AND resource = ? AND external_id = ?", Source, "species", f.SpeciesExternalID).First(&parent).Error; err != nil {
			return err
		}
		if parent.SpeciesID == nil || row.SpeciesID != *parent.SpeciesID {
			return localDifference(m, fmt.Sprintf("species_id=%d", row.SpeciesID), fmt.Sprintf("species_external_id=%d", f.SpeciesExternalID))
		}
	}
	a, _ := json.Marshal(got)
	b, _ := json.Marshal(want)
	if string(a) != string(b) {
		return localDifference(m, string(a), string(b))
	}
	return nil
}
