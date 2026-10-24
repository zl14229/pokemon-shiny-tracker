package reference

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"github.com/zl14229/pokemon-shiny-tracker/internal/testutil"
)

func sourceFixture(path string) string {
	values := map[string]string{
		"/api/v2/pokemon-species/25/": `{"id":25,"name":"pikachu","names":[{"language":{"name":"zh-hans"},"name":"测试皮卡丘"}],"pokedex_numbers":[{"entry_number":25,"pokedex":{"name":"national"}}],"varieties":[{"is_default":true,"pokemon":{"url":"https://pokeapi.co/api/v2/pokemon/25/"}}]}`,
		"/api/v2/pokemon/25/":         `{"id":25,"name":"pikachu","species":{"url":"https://pokeapi.co/api/v2/pokemon-species/25/"},"forms":[{"url":"https://pokeapi.co/api/v2/pokemon-form/10001/"}]}`,
		"/api/v2/pokemon-form/10001/": `{"id":10001,"name":"test-form","names":[],"pokemon":{"url":"https://pokeapi.co/api/v2/pokemon/25/"}}`,
		"/api/v2/version/1/":          `{ "id": 1, "name":"red", "names":[{"language":{"name":"en"},"name":"Test & Game"}] }`,
	}
	return values[path]
}

func TestSourceCacheOfflineAndPacing(t *testing.T) {
	var mu sync.Mutex
	starts := []time.Time{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		starts = append(starts, time.Now())
		mu.Unlock()
		if !strings.Contains(r.UserAgent(), "PokemonShinyTracker") {
			t.Error("unidentified source client")
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, sourceFixture(r.URL.Path))
	}))
	defer server.Close()
	client, err := NewClient(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	client.baseURL = server.URL + "/api/v2/"
	client.interval = 10 * time.Millisecond
	d, err := client.Stage(context.Background(), []int{25}, []int{1})
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Resources) != 3 || d.Resources[0].Facts.NameStatus != "source_provided" || *d.Resources[0].Facts.NameZh != "测试皮卡丘" {
		t.Fatal("localized source value lost")
	}
	for i := 1; i < len(starts); i++ {
		if starts[i].Sub(starts[i-1]) < 8*time.Millisecond {
			t.Fatal("unpaced requests")
		}
	}
	if len(starts) != 4 {
		t.Fatal(len(starts))
	}
	server.Close()
	client.Offline = true
	offline, err := client.Stage(context.Background(), []int{25}, []int{1})
	if err != nil {
		t.Fatal(err)
	}
	if string(d.Canonical()) != string(offline.Canonical()) {
		t.Fatal("cache changed raw evidence or facts")
	}
	path := filepath.Join(client.CacheDir, "version-1.json")
	data, _ := os.ReadFile(path)
	var cached cachedResponse
	json.Unmarshal(data, &cached)
	cached.Body = []byte(`{}`)
	data, _ = json.Marshal(cached)
	os.WriteFile(path, data, 0600)
	if _, err := client.Stage(context.Background(), nil, []int{1}); err == nil {
		t.Fatal("corrupt cache accepted")
	}
}

func TestSourceFailuresAndNameConflict(t *testing.T) {
	for _, test := range []struct {
		status int
		body   string
	}{{503, `{}`}, {200, `{"id":26,"name":"wrong"}`}, {200, `not-json`}} {
		t.Run(fmt.Sprint(test.status, test.body), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(test.status)
				fmt.Fprint(w, test.body)
			}))
			defer server.Close()
			client, _ := NewClient(t.TempDir())
			client.baseURL = server.URL + "/api/v2/"
			if _, err := client.Stage(context.Background(), []int{25}, nil); err == nil {
				t.Fatal("bad source accepted")
			}
		})
	}
	client, _ := NewClient(t.TempDir())
	client.Offline = true
	if _, err := client.Stage(context.Background(), []int{25}, nil); err == nil {
		t.Fatal("uncached offline source accepted")
	}
	f := labels(sourceResource{Name: "test", Names: []localized{{"甲", link{Name: "zh-Hans"}}, {"乙", link{Name: "zh-hans"}}}})
	if f.NameStatus != "conflict" || f.NameZh != nil {
		t.Fatal("conflict silently resolved")
	}
}

func TestImportStableIDsDryRunAndAtomicConflict(t *testing.T) {
	d, err := Bundled()
	if err != nil {
		t.Fatal(err)
	}
	db := testutil.DB(t)
	report, err := Import(db, d, false)
	if err != nil || report.Applied || report.Added != len(d.Resources) {
		t.Fatalf("preview %+v %v", report, err)
	}
	var count int64
	db.Model(&model.Species{}).Count(&count)
	if count != 0 {
		t.Fatal("dry run committed")
	}
	report, err = Import(db, d, true)
	if err != nil || !report.Applied {
		t.Fatalf("apply %+v %v", report, err)
	}
	var mapping model.ExternalReference
	db.Where("resource = ? AND external_id = ?", "species", 25).First(&mapping)
	if mapping.SpeciesID == nil || *mapping.SpeciesID == 25 {
		t.Fatal("internal ID conflated with source ID")
	}
	personal := model.ShinyRecord{CurrentSpeciesID: *mapping.SpeciesID}
	if err := db.Create(&personal).Error; err != nil {
		t.Fatal(err)
	}
	report, err = Import(db, d, true)
	if err != nil || report.Added != 0 || report.Unchanged != len(d.Resources) {
		t.Fatal("not idempotent", err)
	}
	db.Model(&model.ReferenceBatch{}).Count(&count)
	if count != 1 {
		t.Fatal("duplicate batch")
	}
	refreshed, _ := Decode(d.Canonical())
	for i := range refreshed.Resources {
		refreshed.Resources[i].FetchedAt = refreshed.Resources[i].FetchedAt.Add(time.Second)
		if refreshed.Resources[i].Facts.PokemonFetchedAt != nil {
			value := refreshed.Resources[i].Facts.PokemonFetchedAt.Add(time.Second)
			refreshed.Resources[i].Facts.PokemonFetchedAt = &value
			refreshed.Resources[i].Facts.PokemonSHA256 = Hash([]byte("refreshed proof"))
		}
	}
	if report, err := Import(db, refreshed, true); err != nil || report.Unchanged != len(d.Resources) {
		t.Fatal("metadata refresh changed identity/facts", err)
	}

	changed, _ := Decode(d.Canonical())
	changed.Resources[0].Facts.Name = "Changed source name"
	report, err = Import(db, changed, true)
	if err == nil || len(report.Differences) != 1 || report.Applied {
		t.Fatal("changed facts were applied")
	}
	var got model.ShinyRecord
	if err := db.First(&got, personal.ID).Error; err != nil || got.CurrentSpeciesID != personal.CurrentSpeciesID {
		t.Fatal("personal record changed")
	}
	// Add a valid new species followed by a form whose parent is missing.
	var species, form Resource
	for _, r := range d.Resources {
		if r.Kind == "species" {
			species = r
		}
		if r.Kind == "form" {
			form = r
		}
	}
	species.ExternalID = 90001
	species.SourceURL = Endpoint("species", 90001)
	species.Facts.Slug = "test-new-species"
	species.Facts.NationalDex = 90001
	form.ExternalID = 90002
	form.SourceURL = Endpoint("form", 90002)
	form.Facts.Slug = "test-new-form"
	form.Facts.SpeciesExternalID = 99999
	broken := Dataset{Format: 1, Source: Source, Resources: []Resource{species, form}}
	if _, err := Import(db, broken, true); err == nil {
		t.Fatal("missing parent imported")
	}
	db.Model(&model.Species{}).Where("national_dex = ?", 90001).Count(&count)
	if count != 0 {
		t.Fatal("failed batch partially applied")
	}
	db.Model(&model.ReferenceBatch{}).Count(&count)
	if count != 1 {
		t.Fatal("failed batch left evidence rows")
	}
	// A source outage has no database dependency and cannot affect local records.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer server.Close()
	client, _ := NewClient(t.TempDir())
	client.baseURL = server.URL + "/api/v2/"
	if _, err := client.Stage(context.Background(), []int{25}, nil); err == nil {
		t.Fatal("source should fail")
	}
	if err := db.First(&got, personal.ID).Error; err != nil {
		t.Fatal("source failure damaged personal record")
	}
}

func TestDatasetRejectsUnverifiedMappingAndEnsureSeedPreservesExisting(t *testing.T) {
	d, _ := Bundled()
	for _, change := range []func(*Dataset){
		func(x *Dataset) { x.Source = "unknown" },
		func(x *Dataset) { x.Resources[0].SourceURL = "https://example.invalid/" },
		func(x *Dataset) { x.Resources = append(x.Resources, x.Resources[0]) },
		func(x *Dataset) { x.Resources[0].RawSHA256 = "bad" },
	} {
		copy, _ := Decode(d.Canonical())
		change(&copy)
		if copy.Validate() == nil {
			t.Fatal("invalid dataset accepted")
		}
	}
	db := testutil.DB(t)
	r := testutil.Seed(t, db)
	if err := EnsureSeed(db); err != nil {
		t.Fatal(err)
	}
	var species model.Species
	db.First(&species, r.Species.ID)
	if report, err := Import(db, d, true); err == nil || len(report.Differences) == 0 {
		t.Fatal("unmapped identity collision lacked a reviewable diff")
	}
	if species.Slug != "test-species" {
		t.Fatal("existing local references overwritten")
	}
}
