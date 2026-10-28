// Package huntdata validates staged, source-backed game-specific hunt facts.
// It never performs network access; persistence is isolated in store.go.
package huntdata

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
)

// Evidence describes the scope of a particular reviewed claim.
type Evidence struct {
	Source    string    `json:"source"`
	URL       string    `json:"url"`
	Version   string    `json:"version"`
	Scope     string    `json:"scope"`
	CheckedAt time.Time `json:"checked_at"`
}

// Probability is a conditional fraction; nil means unknown, never zero.
type Probability struct {
	Numerator   uint64   `json:"numerator"`
	Denominator uint64   `json:"denominator"`
	Conditions  string   `json:"conditions"`
	Evidence    Evidence `json:"evidence"`
}

type Method struct {
	Name          string       `json:"name"`
	Location      string       `json:"location"`
	Conditions    string       `json:"conditions"`
	Evidence      Evidence     `json:"evidence"`
	EncounterRate *Probability `json:"encounter_rate"`
	ShinyRate     *Probability `json:"shiny_rate"`
}

// Target uses stable PokéAPI v2 IDs, not installation-specific database IDs.
type Target struct {
	GameVersionID int        `json:"game_version_id"`
	SpeciesID     int        `json:"species_id"`
	FormID        int        `json:"form_id"`
	Availability  string     `json:"availability"`
	DirectHunt    string     `json:"direct_hunt"`
	Verification  string     `json:"verification"`
	GuideStatus   string     `json:"guide_status"`
	Evidence      Evidence   `json:"evidence"`
	CrossChecks   []Evidence `json:"cross_checks"`
	Methods       []Method   `json:"methods"`
}

type Dataset struct {
	Version string   `json:"version"`
	Targets []Target `json:"targets"`
}

func validEvidence(e Evidence) bool {
	u, err := url.Parse(e.URL)
	return strings.TrimSpace(e.Source) != "" && strings.TrimSpace(e.Version) != "" && strings.TrimSpace(e.Scope) != "" && !e.CheckedAt.IsZero() && err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Hostname() != "" && u.User == nil
}

func validProbability(p *Probability) bool {
	return p == nil || p.Denominator > 0 && p.Numerator <= p.Denominator && strings.TrimSpace(p.Conditions) != "" && validEvidence(p.Evidence)
}

// Validate rejects ambiguous, unsupported and internally inconsistent claims.
// It checks structure only; independent review establishes factual truth.
func (d Dataset) Validate() error {
	if strings.TrimSpace(d.Version) == "" || len(d.Version) > 128 || len(d.Targets) > 10000 {
		return errors.New("dataset version required")
	}
	seen := map[[2]int]bool{}
	for i, t := range d.Targets {
		fail := func(reason string) error { return fmt.Errorf("target %d: %s", i, reason) }
		if t.GameVersionID <= 0 || t.SpeciesID <= 0 || t.FormID <= 0 {
			return fail("invalid external identity")
		}
		key := [2]int{t.GameVersionID, t.FormID}
		if seen[key] {
			return fail("duplicate game and form")
		}
		seen[key] = true
		if !oneOf(t.Availability, "confirmed_available", "confirmed_locked", "transfer_only", "unknown") || !oneOf(t.DirectHunt, "yes", "no", "unknown") || !oneOf(t.Verification, "verified", "unknown", "conflict") || !oneOf(t.GuideStatus, "complete", "partial", "unknown") {
			return fail("invalid status")
		}
		if t.Verification == "verified" && (!validEvidence(t.Evidence) || t.Availability == "unknown") {
			return fail("verified claim lacks evidence or is unknown")
		}
		if t.Availability == "unknown" && (t.DirectHunt != "unknown" || t.Verification == "verified") {
			return fail("unknown availability cannot assert direct hunt")
		}
		if t.DirectHunt == "yes" && (t.Verification != "verified" || t.Availability != "confirmed_available") {
			return fail("direct hunt requires verified availability")
		}
		if (t.Availability == "confirmed_locked" || t.Availability == "transfer_only") && t.DirectHunt != "no" {
			return fail("locked or transfer-only cannot be directly hunted")
		}
		if t.GuideStatus == "complete" && len(t.Methods) == 0 {
			return fail("complete guide lacks methods")
		}
		for _, e := range t.CrossChecks {
			if !validEvidence(e) {
				return fail("invalid cross check")
			}
		}
		for _, m := range t.Methods {
			if strings.TrimSpace(m.Name) == "" || strings.TrimSpace(m.Conditions) == "" || !validEvidence(m.Evidence) || !validProbability(m.EncounterRate) || !validProbability(m.ShinyRate) {
				return fail("invalid method or probability evidence")
			}
		}
	}
	return nil
}

func oneOf(s string, values ...string) bool {
	for _, v := range values {
		if s == v {
			return true
		}
	}
	return false
}

// Huntable deliberately does not infer direct huntability from availability.
func (t Target) Huntable() bool {
	return t.Verification == "verified" && t.Availability == "confirmed_available" && t.DirectHunt == "yes" && (Dataset{Version: "candidate", Targets: []Target{t}}).Validate() == nil
}

// Decode accepts a bounded staged document with no silently ignored fields.
func Decode(r io.Reader) (Dataset, error) {
	var d Dataset
	dec := json.NewDecoder(io.LimitReader(r, 8<<20))
	dec.DisallowUnknownFields()
	if e := dec.Decode(&d); e != nil {
		return d, e
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return d, errors.New("extra data")
	}
	return d, d.Validate()
}
