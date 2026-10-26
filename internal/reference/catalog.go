package reference

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"time"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"gorm.io/gorm"
)

//go:embed catalog.json
var catalogBytes []byte

const CatalogSource = "pokeapi-official-v2-csv"

type CatalogDocument struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}
type CatalogRecord struct {
	Kind              string            `json:"kind"`
	ExternalID        int               `json:"external_id"`
	SpeciesExternalID int               `json:"species_external_id"`
	NationalDex       int               `json:"national_dex"`
	Slug              string            `json:"slug"`
	Names             map[string]string `json:"names"`
	Inclusion         string            `json:"inclusion"`
	Reason            string            `json:"reason"`
	CSVFile           string            `json:"csv_file"`
}
type CatalogSnapshot struct {
	Format    int                        `json:"format"`
	Source    string                     `json:"source"`
	Revision  string                     `json:"revision"`
	FetchedAt time.Time                  `json:"fetched_at"`
	Documents map[string]CatalogDocument `json:"documents"`
	Records   []CatalogRecord            `json:"records"`
}

func CatalogBundled() (CatalogSnapshot, error) {
	var s CatalogSnapshot
	err := json.Unmarshal(catalogBytes, &s)
	if err != nil {
		return s, err
	}
	return s, s.Validate()
}
func (s CatalogSnapshot) Validate() error {
	if s.Format != 1 || s.Source != CatalogSource || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(s.Revision) || s.FetchedAt.IsZero() || len(s.Records) == 0 || len(s.Records) > 10000 {
		return errors.New("invalid catalog snapshot")
	}
	for file, d := range s.Documents {
		suffix := "data/v2/csv/" + file
		if file == "LICENSE.md" {
			suffix = file
		}
		if d.URL != "https://raw.githubusercontent.com/PokeAPI/pokeapi/"+s.Revision+"/"+suffix || !hashPattern.MatchString(d.SHA256) {
			return errors.New("invalid catalog source document")
		}
	}
	seen := map[string]bool{}
	parents := map[int]bool{}
	dexes := map[int]bool{}
	for _, r := range s.Records {
		key := fmt.Sprintf("%s:%d", r.Kind, r.ExternalID)
		if (r.Kind != "species" && r.Kind != "form") || r.ExternalID < 1 || r.SpeciesExternalID < 1 || seen[key] || !slugPattern.MatchString(r.Slug) || len(r.Slug) > 128 {
			return errors.New("invalid/duplicate catalog identity")
		}
		seen[key] = true
		if r.Inclusion != "included" && r.Inclusion != "excluded" && r.Inclusion != "unknown" {
			return errors.New("invalid catalog inclusion")
		}
		if r.Reason == "" || len(r.Names) > 30 {
			return errors.New("missing catalog policy")
		}
		for lang, name := range r.Names {
			if !slugPattern.MatchString(lang) || name == "" || len(name) > 256 {
				return errors.New("invalid localized name")
			}
		}
		file := "pokemon_forms.csv"
		if r.Kind == "species" {
			file = "pokemon_species.csv"
			if r.NationalDex < 1 || dexes[r.NationalDex] || r.SpeciesExternalID != r.ExternalID {
				return errors.New("invalid national species mapping")
			}
			parents[r.ExternalID] = true
			dexes[r.NationalDex] = true
		} else if r.NationalDex != 0 {
			return errors.New("form is not a national species")
		}
		if r.CSVFile != file || s.Documents[file].URL == "" {
			return errors.New("missing source evidence")
		}
	}
	for _, r := range s.Records {
		if !parents[r.SpeciesExternalID] {
			return errors.New("missing catalog parent")
		}
	}
	for _, name := range []string{"pokemon.csv", "pokemon_species_names.csv", "pokemon_form_names.csv", "languages.csv", "LICENSE.md"} {
		if s.Documents[name].URL == "" {
			return errors.New("missing mapping/name/license provenance")
		}
	}
	return nil
}

// ImportCatalog only adds identities and enrichment. An upstream change is rejected,
// pending T024 review; personal tables are never updated. Preview uses rollback.
func ImportCatalog(db *gorm.DB, s CatalogSnapshot, apply bool) (Report, error) {
	report := Report{Differences: []Difference{}}
	if err := s.Validate(); err != nil {
		return report, err
	}
	raw, _ := json.Marshal(s)
	checksum := Hash(raw)
	preview := errors.New("catalog preview rollback")
	err := db.Transaction(func(tx *gorm.DB) error {
		var batch model.ReferenceBatch
		if e := tx.Where("checksum = ? AND source = ?", checksum, CatalogSource).First(&batch).Error; e == nil {
			report.Unchanged = len(s.Records)
			return nil
		} else if !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		ids := map[int]uint64{}
		for _, kind := range []string{"species", "form"} {
			for _, r := range s.Records {
				if r.Kind != kind {
					continue
				}
				var old model.CatalogEntry
				if e := tx.Where("kind = ? AND external_id = ?", kind, r.ExternalID).First(&old).Error; e == nil {
					return errors.New("catalog already exists at another revision; staged update requires review")
				} else if !errors.Is(e, gorm.ErrRecordNotFound) {
					return e
				}
				labels := model.Labels{Slug: r.Slug, Name: r.Slug, NameStatus: "unknown"}
				if name := r.Names["en"]; name != "" {
					labels.Name = name
				}
				if zh := r.Names["zh-hans"]; zh != "" {
					labels.NameZh = &zh
					labels.NameStatus = "source_provided"
				}
				nameStatus := labels.NameStatus
				var sid uint64
				var fid *uint64
				if kind == "species" {
					var row model.Species
					e := tx.Where("national_dex = ? OR slug = ?", r.NationalDex, r.Slug).First(&row).Error
					if errors.Is(e, gorm.ErrRecordNotFound) {
						row = model.Species{Labels: labels, NationalDex: r.NationalDex}
						e = tx.Create(&row).Error
					} else if e == nil && (row.NationalDex != r.NationalDex || row.Slug != r.Slug) {
						return errors.New("catalog species identity conflict")
					}
					if e != nil {
						return e
					}
					if row.NameStatus == "conflict" || (row.NameZh != nil && labels.NameZh != nil && *row.NameZh != *labels.NameZh) {
						nameStatus = "conflict"
					}
					sid = row.ID
					ids[r.ExternalID] = sid
				} else {
					sid = ids[r.SpeciesExternalID]
					var row model.Form
					e := tx.Where("slug = ?", r.Slug).First(&row).Error
					if errors.Is(e, gorm.ErrRecordNotFound) {
						row = model.Form{Labels: labels, SpeciesID: sid}
						e = tx.Create(&row).Error
					} else if e == nil && row.SpeciesID != sid {
						return errors.New("catalog form identity conflict")
					}
					if e != nil {
						return e
					}
					if row.NameStatus == "conflict" || (row.NameZh != nil && labels.NameZh != nil && *row.NameZh != *labels.NameZh) {
						nameStatus = "conflict"
					}
					fid = &row.ID
				}
				// Existing v2 source mappings must agree with the stable target.
				var mapped model.ExternalReference
				if e := tx.Where("source = ? AND resource = ? AND external_id = ?", Source, kind, r.ExternalID).First(&mapped).Error; e == nil {
					if kind == "species" && (mapped.SpeciesID == nil || *mapped.SpeciesID != sid) || kind == "form" && (mapped.FormID == nil || *mapped.FormID != *fid) {
						return errors.New("external catalog mapping conflict")
					}
				} else if !errors.Is(e, gorm.ErrRecordNotFound) {
					return e
				}
				names, _ := json.Marshal(r.Names)
				cross := "https://wiki.52poke.com/zh-hans/" + url.PathEscape(r.Names["zh-hans"])
				if r.Names["zh-hans"] == "" {
					cross = "https://wiki.52poke.com/wiki/全国图鉴"
				}
				if kind == "form" {
					for _, parent := range s.Records {
						if parent.Kind == "species" && parent.ExternalID == r.SpeciesExternalID && parent.Names["zh-hans"] != "" {
							cross = "https://wiki.52poke.com/zh-hans/" + url.PathEscape(parent.Names["zh-hans"])
							break
						}
					}
				}
				doc := s.Documents[r.CSVFile]
				entry := model.CatalogEntry{Kind: kind, ExternalID: r.ExternalID, SpeciesID: sid, FormID: fid, NamesJSON: string(names), NameStatus: nameStatus, Inclusion: r.Inclusion, Reason: r.Reason, SourceURL: doc.URL, CrossCheckURL: cross, CrossCheckStatus: "unknown", SourceRevision: s.Revision, SourceSHA256: doc.SHA256, FetchedAt: s.FetchedAt, CheckedAt: time.Now().UTC()}
				if e := tx.Create(&entry).Error; e != nil {
					return e
				}
				encoded, _ := json.Marshal(entry)
				report.Differences = append(report.Differences, Difference{Kind: r.Kind, ExternalID: r.ExternalID, Before: "", After: string(encoded)})
				report.Added++
			}
		}
		batch = model.ReferenceBatch{Checksum: checksum, Source: CatalogSource, SnapshotJSON: string(raw)}
		if e := tx.Create(&batch).Error; e != nil {
			return e
		}
		if !apply {
			return preview
		}
		return nil
	})
	if errors.Is(err, preview) {
		return report, nil
	}
	if err != nil {
		report.Added = 0
		return report, err
	}
	report.Applied = apply
	return report, nil
}
func EnsureCatalog(db *gorm.DB) error {
	s, e := CatalogBundled()
	if e != nil {
		return e
	}
	_, e = ImportCatalog(db, s, true)
	return e
}

// DisplayCatalogName never labels an unreviewed name as an official translation.
func DisplayCatalogName(names map[string]string, status, slug string) string {
	if status == "verified" || status == "source_provided" {
		if value := names["zh-hans"]; value != "" {
			return value
		}
	}
	if value := names["en"]; value != "" {
		return value
	}
	return slug
}
