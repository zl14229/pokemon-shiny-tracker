// Package reference stages and imports validated local reference facts.
package reference

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"time"
)

const Source = "pokeapi-v2"
const BaseURL = "https://pokeapi.co/api/v2/"

type Facts struct {
	Slug              string     `json:"slug"`
	Name              string     `json:"name"`
	NameZh            *string    `json:"name_zh"`
	NameStatus        string     `json:"name_status"`
	NationalDex       int        `json:"national_dex,omitempty"`
	SpeciesExternalID int        `json:"species_external_id,omitempty"`
	PokemonURL        string     `json:"pokemon_url,omitempty"`
	PokemonSHA256     string     `json:"pokemon_sha256,omitempty"`
	PokemonFetchedAt  *time.Time `json:"pokemon_fetched_at,omitempty"`
}

type Resource struct {
	Kind       string    `json:"kind"`
	ExternalID int       `json:"external_id"`
	SourceURL  string    `json:"source_url"`
	RawSHA256  string    `json:"raw_sha256"`
	FetchedAt  time.Time `json:"fetched_at"`
	Facts      Facts     `json:"facts"`
}

type Dataset struct {
	Format    int        `json:"format"`
	Source    string     `json:"source"`
	Resources []Resource `json:"resources"`
}

func Hash(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func Endpoint(kind string, id int) string {
	resource := map[string]string{"species": "pokemon-species", "form": "pokemon-form", "game": "version"}[kind]
	return fmt.Sprintf("%s%s/%d/", BaseURL, resource, id)
}

func Decode(data []byte) (Dataset, error) {
	if len(data) > 8<<20 {
		return Dataset{}, errors.New("reference dataset is too large")
	}
	var d Dataset
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&d); err != nil {
		return d, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return d, errors.New("expected one dataset object")
	}
	return d, d.Validate()
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var hashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func (d Dataset) Validate() error {
	if d.Format != 1 || d.Source != Source || len(d.Resources) == 0 || len(d.Resources) > 5000 {
		return errors.New("invalid dataset format, source or size")
	}
	seen := map[string]bool{}
	dexes := map[int]bool{}
	for _, r := range d.Resources {
		if r.Kind != "species" && r.Kind != "form" && r.Kind != "game" {
			return errors.New("unknown resource kind")
		}
		key := fmt.Sprintf("%s:%d", r.Kind, r.ExternalID)
		if seen[key] || r.ExternalID < 1 || r.SourceURL != Endpoint(r.Kind, r.ExternalID) || !hashPattern.MatchString(r.RawSHA256) || r.FetchedAt.IsZero() {
			return errors.New("invalid or duplicate source identity/evidence")
		}
		seen[key] = true
		f := r.Facts
		if !slugPattern.MatchString(f.Slug) || len(f.Slug) > 128 || f.Name == "" || len(f.Name) > 256 {
			return errors.New("invalid reference name")
		}
		if f.NameStatus != "unknown" && f.NameStatus != "source_provided" && f.NameStatus != "conflict" {
			return errors.New("unsupported localization verification status")
		}
		if f.NameZh != nil && (*f.NameZh == "" || len(*f.NameZh) > 256 || f.NameStatus != "source_provided") {
			return errors.New("invalid Chinese name evidence")
		}
		if f.NameStatus == "source_provided" && f.NameZh == nil {
			return errors.New("missing Chinese source value")
		}
		if r.Kind == "species" {
			if f.NationalDex < 1 || dexes[f.NationalDex] {
				return errors.New("invalid or duplicate national dex")
			}
			dexes[f.NationalDex] = true
		} else if f.NationalDex != 0 {
			return errors.New("non-species cannot have a national dex")
		}
		if r.Kind == "form" {
			if f.SpeciesExternalID < 1 || f.PokemonFetchedAt == nil || f.PokemonFetchedAt.IsZero() || !hashPattern.MatchString(f.PokemonSHA256) || !regexp.MustCompile(`^https://pokeapi\.co/api/v2/pokemon/[1-9][0-9]*/$`).MatchString(f.PokemonURL) {
				return errors.New("form lacks validated Pokemon-to-species evidence")
			}
		} else if f.SpeciesExternalID != 0 || f.PokemonURL != "" || f.PokemonSHA256 != "" || f.PokemonFetchedAt != nil {
			return errors.New("unexpected form mapping")
		}
	}
	return nil
}

func (d Dataset) Canonical() []byte {
	copyDataset := d
	copyDataset.Resources = append([]Resource(nil), d.Resources...)
	sort.Slice(copyDataset.Resources, func(i, j int) bool {
		a, b := copyDataset.Resources[i], copyDataset.Resources[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.ExternalID < b.ExternalID
	})
	data, _ := json.Marshal(copyDataset)
	return data
}
