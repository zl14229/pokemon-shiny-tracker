package huntdata

import (
	"testing"
	"time"
)

func TestValidateStagedFacts(t *testing.T) {
	e := Evidence{Source: "synthetic", URL: "https://example.org/fact", Version: "v1", Scope: "test game and form", CheckedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	base := Target{GameVersionID: 1, SpeciesID: 25, FormID: 25, Availability: "confirmed_available", DirectHunt: "yes", Verification: "verified", GuideStatus: "unknown", Evidence: e}
	tests := []struct {
		name            string
		change          func(*Dataset)
		valid, huntable bool
	}{
		{"verified with unknown guide", func(*Dataset) {}, true, true},
		{"missing evidence", func(d *Dataset) { d.Targets[0].Evidence.URL = "" }, false, false},
		{"unverified direct", func(d *Dataset) { d.Targets[0].Verification = "unknown" }, false, false},
		{"transfer only", func(d *Dataset) { d.Targets[0].Availability = "transfer_only"; d.Targets[0].DirectHunt = "no" }, true, false},
		{"locked", func(d *Dataset) { d.Targets[0].Availability = "confirmed_locked"; d.Targets[0].DirectHunt = "no" }, true, false},
		{"unknown", func(d *Dataset) {
			d.Targets[0].Availability = "unknown"
			d.Targets[0].DirectHunt = "unknown"
			d.Targets[0].Verification = "unknown"
		}, true, false},
		{"duplicate", func(d *Dataset) { d.Targets = append(d.Targets, d.Targets[0]) }, false, true},
		{"no method for complete guide", func(d *Dataset) { d.Targets[0].GuideStatus = "complete" }, false, false},
		{"conditional rate", func(d *Dataset) {
			d.Targets[0].Methods = []Method{{Name: "synthetic", Conditions: "test only", Evidence: e, ShinyRate: &Probability{Numerator: 1, Denominator: 4096, Conditions: "test only", Evidence: e}}}
		}, true, true},
		{"unscoped rate", func(d *Dataset) {
			d.Targets[0].Methods = []Method{{Name: "synthetic", Conditions: "test only", Evidence: e, ShinyRate: &Probability{Numerator: 1, Denominator: 4096, Evidence: e}}}
		}, false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := Dataset{Version: "fixture", Targets: []Target{base}}
			tc.change(&d)
			if got := d.Validate() == nil; got != tc.valid {
				t.Errorf("valid = %v", got)
			}
			if got := d.Targets[0].Huntable(); got != tc.huntable {
				t.Errorf("huntable = %v", got)
			}
		})
	}
}
