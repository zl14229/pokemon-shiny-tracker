package reference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zl14229/pokemon-shiny-tracker/internal/database"
)

type cachedResponse struct {
	URL       string    `json:"url"`
	SHA256    string    `json:"sha256"`
	FetchedAt time.Time `json:"fetched_at"`
	Body      []byte    `json:"body"`
}

type Client struct {
	CacheDir            string
	Offline             bool
	Refresh             bool
	baseURL             string
	interval            time.Duration
	http                *http.Client
	mu                  sync.Mutex
	lastRequestFinished time.Time
}

func NewClient(cacheDir string) (*Client, error) {
	if err := database.ValidatePath(filepath.Join(cacheDir, "resource.json")); err != nil {
		return nil, err
	}
	return &Client{CacheDir: cacheDir, baseURL: BaseURL, interval: time.Second, http: &http.Client{
		Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("source redirect rejected") },
	}}, nil
}

func DefaultCacheDir() (string, error) {
	dir, err := os.UserCacheDir()
	return filepath.Join(dir, "pokemon-shiny-tracker", "pokeapi-v2"), err
}

func (c *Client) fetch(ctx context.Context, resource string, id int) (cachedResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if id < 1 || (resource != "pokemon-species" && resource != "pokemon" && resource != "pokemon-form" && resource != "version") {
		return cachedResponse{}, errors.New("unsupported source request")
	}
	canonical := fmt.Sprintf("%s%s/%d/", BaseURL, resource, id)
	path := filepath.Join(c.CacheDir, fmt.Sprintf("%s-%d.json", resource, id))
	if err := database.ValidatePath(path); err != nil {
		return cachedResponse{}, err
	}
	if data, err := os.ReadFile(path); err == nil {
		var entry cachedResponse
		if json.Unmarshal(data, &entry) != nil || entry.URL != canonical || !hashPattern.MatchString(entry.SHA256) || entry.FetchedAt.IsZero() {
			return entry, errors.New("refusing to overwrite an unrecognized cache file")
		}
		if !c.Refresh || c.Offline {
			if entry.SHA256 != Hash(entry.Body) {
				return entry, errors.New("cache integrity validation failed")
			}
			return entry, nil
		}
	} else if !os.IsNotExist(err) {
		return cachedResponse{}, err
	}
	if c.Offline {
		return cachedResponse{}, errors.New("offline source resource is not cached")
	}
	if wait := time.Until(c.lastRequestFinished.Add(c.interval)); wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return cachedResponse{}, ctx.Err()
		case <-timer.C:
		}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s%s/%d/", c.baseURL, resource, id), nil)
	if err != nil {
		return cachedResponse{}, err
	}
	request.Header.Set("User-Agent", "PokemonShinyTracker/0.1 (+https://github.com/zl14229/pokemon-shiny-tracker)")
	request.Header.Set("Accept", "application/json")
	// Pace from the end of the previous network attempt, after its body is
	// closed. A slow connection or response must not consume the cooldown
	// and cause an immediate follow-up request at the source.
	defer func() { c.lastRequestFinished = time.Now() }()
	response, err := c.http.Do(request)
	if err != nil {
		return cachedResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return cachedResponse{}, fmt.Errorf("source HTTP %d", response.StatusCode)
	}
	if !strings.HasPrefix(strings.ToLower(response.Header.Get("Content-Type")), "application/json") {
		return cachedResponse{}, errors.New("source response is not JSON")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (2<<20)+1))
	if err != nil {
		return cachedResponse{}, err
	}
	if len(body) > 2<<20 || !json.Valid(body) {
		return cachedResponse{}, errors.New("source body is invalid or too large")
	}
	entry := cachedResponse{URL: canonical, SHA256: Hash(body), FetchedAt: time.Now().UTC(), Body: body}
	encoded, err := json.Marshal(entry)
	if err != nil {
		return entry, err
	}
	if err := os.MkdirAll(c.CacheDir, 0700); err != nil {
		return entry, err
	}
	f, err := os.CreateTemp(c.CacheDir, ".source-*")
	if err != nil {
		return entry, err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(encoded); err != nil {
		f.Close()
		return entry, err
	}
	if err := f.Close(); err != nil {
		return entry, err
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return entry, err
	}
	return entry, nil
}

type link struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}
type localized struct {
	Name     string `json:"name"`
	Language link   `json:"language"`
}
type sourceResource struct {
	ID             int         `json:"id"`
	Name           string      `json:"name"`
	Names          []localized `json:"names"`
	PokedexNumbers []struct {
		EntryNumber int  `json:"entry_number"`
		Pokedex     link `json:"pokedex"`
	} `json:"pokedex_numbers"`
	Varieties []struct {
		IsDefault bool `json:"is_default"`
		Pokemon   link `json:"pokemon"`
	} `json:"varieties"`
	Species link   `json:"species"`
	Pokemon link   `json:"pokemon"`
	Forms   []link `json:"forms"`
}

func linkID(value, resource string) (int, error) {
	prefix := BaseURL + resource + "/"
	if !strings.HasPrefix(value, prefix) || !strings.HasSuffix(value, "/") {
		return 0, errors.New("invalid source link")
	}
	text := strings.TrimSuffix(strings.TrimPrefix(value, prefix), "/")
	id, err := strconv.Atoi(text)
	if err != nil || id < 1 || strconv.Itoa(id) != text {
		return 0, errors.New("invalid source ID")
	}
	return id, nil
}

func (c *Client) read(ctx context.Context, resource string, id int) (sourceResource, cachedResponse, error) {
	entry, err := c.fetch(ctx, resource, id)
	if err != nil {
		return sourceResource{}, entry, err
	}
	var data sourceResource
	if err := json.Unmarshal(entry.Body, &data); err != nil {
		return data, entry, err
	}
	if data.ID != id || !slugPattern.MatchString(data.Name) {
		return data, entry, errors.New("source ID/name mismatch")
	}
	return data, entry, nil
}

func labels(data sourceResource) Facts {
	f := Facts{Slug: data.Name, Name: data.Name, NameStatus: "unknown"}
	chinese := map[string]bool{}
	english := map[string]bool{}
	for _, name := range data.Names {
		if strings.EqualFold(name.Language.Name, "en") && name.Name != "" {
			english[name.Name] = true
		}
		if strings.EqualFold(name.Language.Name, "zh-Hans") && name.Name != "" {
			chinese[name.Name] = true
		}
	}
	if len(english) == 1 {
		for name := range english {
			f.Name = name
		}
	}
	if len(chinese) > 1 {
		f.NameStatus = "conflict"
	} else {
		for name := range chinese {
			value := name
			f.NameZh = &value
			f.NameStatus = "source_provided"
		}
	}
	if len(english) > 1 {
		f.Name = data.Name
		f.NameZh = nil
		f.NameStatus = "conflict"
	}
	return f
}

// Stage performs only source/cache work; it never receives a database handle.
func (c *Client) Stage(ctx context.Context, speciesIDs, versionIDs []int) (Dataset, error) {
	d := Dataset{Format: 1, Source: Source}
	seenForms := map[int]int{}
	for _, id := range speciesIDs {
		species, e, err := c.read(ctx, "pokemon-species", id)
		if err != nil {
			return d, err
		}
		facts := labels(species)
		for _, dex := range species.PokedexNumbers {
			if dex.Pokedex.Name == "national" {
				if facts.NationalDex != 0 && facts.NationalDex != dex.EntryNumber {
					return d, errors.New("conflicting national dex")
				}
				facts.NationalDex = dex.EntryNumber
			}
		}
		d.Resources = append(d.Resources, Resource{Kind: "species", ExternalID: id, SourceURL: e.URL, RawSHA256: e.SHA256, FetchedAt: e.FetchedAt, Facts: facts})
		var defaults []link
		for _, v := range species.Varieties {
			if v.IsDefault {
				defaults = append(defaults, v.Pokemon)
			}
		}
		if len(defaults) != 1 {
			return d, errors.New("expected one default Pokemon variety")
		}
		pokemonID, err := linkID(defaults[0].URL, "pokemon")
		if err != nil {
			return d, err
		}
		pokemon, pe, err := c.read(ctx, "pokemon", pokemonID)
		if err != nil {
			return d, err
		}
		parent, err := linkID(pokemon.Species.URL, "pokemon-species")
		if err != nil || parent != id {
			return d, errors.New("Pokemon species mapping mismatch")
		}
		for _, formLink := range pokemon.Forms {
			formID, err := linkID(formLink.URL, "pokemon-form")
			if err != nil {
				return d, err
			}
			if previous, seen := seenForms[formID]; seen {
				if previous != id {
					return d, errors.New("conflicting form species mapping")
				}
				continue
			}
			seenForms[formID] = id
			form, fe, err := c.read(ctx, "pokemon-form", formID)
			if err != nil {
				return d, err
			}
			formPokemon, err := linkID(form.Pokemon.URL, "pokemon")
			if err != nil || formPokemon != pokemonID {
				return d, errors.New("form Pokemon mapping mismatch")
			}
			ff := labels(form)
			ff.SpeciesExternalID = id
			ff.PokemonURL = pe.URL
			ff.PokemonSHA256 = pe.SHA256
			ff.PokemonFetchedAt = &pe.FetchedAt
			d.Resources = append(d.Resources, Resource{Kind: "form", ExternalID: formID, SourceURL: fe.URL, RawSHA256: fe.SHA256, FetchedAt: fe.FetchedAt, Facts: ff})
		}
	}
	for _, id := range versionIDs {
		game, e, err := c.read(ctx, "version", id)
		if err != nil {
			return d, err
		}
		d.Resources = append(d.Resources, Resource{Kind: "game", ExternalID: id, SourceURL: e.URL, RawSHA256: e.SHA256, FetchedAt: e.FetchedAt, Facts: labels(game)})
	}
	return d, d.Validate()
}
