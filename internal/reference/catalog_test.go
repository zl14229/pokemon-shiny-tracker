package reference

import (
	"encoding/json"
	"testing"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"github.com/zl14229/pokemon-shiny-tracker/internal/testutil"
)

func TestCatalogOfflineImportStablePersonalIdentity(t *testing.T) {
	db := testutil.DB(t)
	if e := EnsureSeed(db); e != nil {
		t.Fatal(e)
	}
	var s model.Species
	db.Where("national_dex = ?", 25).First(&s)
	personal := model.ShinyRecord{CurrentSpeciesID: s.ID, HoldingStatus: "held"}
	if e := db.Create(&personal).Error; e != nil {
		t.Fatal(e)
	}
	before, _ := json.Marshal(personal)
	snapshot, e := CatalogBundled()
	if e != nil {
		t.Fatal(e)
	}
	preview, e := ImportCatalog(db, snapshot, false)
	if e != nil || preview.Added != 2604 || preview.Applied {
		t.Fatal(preview, e)
	}
	var n int64
	db.Model(&model.CatalogEntry{}).Count(&n)
	if n != 0 {
		t.Fatal("preview persisted")
	}
	result, e := ImportCatalog(db, snapshot, true)
	if e != nil || result.Added != 2604 {
		t.Fatal(result, e)
	}
	again, e := ImportCatalog(db, snapshot, true)
	if e != nil || again.Unchanged != 2604 || again.Added != 0 {
		t.Fatal(again, e)
	}
	var got model.ShinyRecord
	db.First(&got, personal.ID)
	after, _ := json.Marshal(got)
	if string(before) != string(after) {
		t.Fatal("personal record overwritten")
	}
	db.Model(&model.Species{}).Count(&n)
	if n != 1025 {
		t.Fatal(n)
	}
	db.Model(&model.Form{}).Count(&n)
	if n != 1579 {
		t.Fatal(n)
	}
	var p model.CatalogEntry
	db.Where("kind = ? AND external_id = ?", "species", 25).First(&p)
	if p.SpeciesID != s.ID {
		t.Fatal("species identity changed")
	}
	var regional model.CatalogEntry
	db.Joins("JOIN forms ON forms.id = catalog_entries.form_id").Where("forms.slug = ?", "raichu-alola").First(&regional)
	if regional.ID == 0 || regional.Inclusion != "included" {
		t.Fatal(regional)
	}
	var mega model.CatalogEntry
	db.Joins("JOIN forms ON forms.id = catalog_entries.form_id").Where("forms.slug = ?", "charizard-mega-x").First(&mega)
	if mega.Inclusion != "excluded" {
		t.Fatal(mega)
	}
	snapshot.Revision = "1111111111111111111111111111111111111111"
	if _, e := ImportCatalog(db, snapshot, true); e == nil {
		t.Fatal("changed provenance accepted")
	}
}
func TestCatalogValidationAndLanguageFallback(t *testing.T) {
	cases := []struct {
		names              map[string]string
		status, slug, want string
	}{
		{map[string]string{"zh-hans": "皮卡丘", "en": "Pikachu"}, "source_provided", "pikachu", "皮卡丘"},
		{map[string]string{"zh-hant": "皮卡丘", "en": "Pikachu"}, "unknown", "pikachu", "Pikachu"},
		{map[string]string{"zh-hans": "wrong", "en": "Pikachu"}, "conflict", "pikachu", "Pikachu"},
		{map[string]string{}, "unknown", "raichu-alola", "raichu-alola"},
	}
	for _, c := range cases {
		if got := DisplayCatalogName(c.names, c.status, c.slug); got != c.want {
			t.Fatal(got, c.want)
		}
	}
	d, e := CatalogBundled()
	if e != nil {
		t.Fatal(e)
	}
	d.Records = append(d.Records, d.Records[0])
	if d.Validate() == nil {
		t.Fatal("duplicate accepted")
	}
}
func TestCatalogConflictRollsBack(t *testing.T) {
	db := testutil.DB(t)
	s := model.Species{Labels: model.Labels{Slug: "local-bulbasaur", Name: "Private"}, NationalDex: 1}
	db.Create(&s)
	p := model.ShinyRecord{CurrentSpeciesID: s.ID}
	db.Create(&p)
	d, _ := CatalogBundled()
	if _, e := ImportCatalog(db, d, true); e == nil {
		t.Fatal("local conflict ignored")
	}
	var n int64
	db.Model(&model.CatalogEntry{}).Count(&n)
	if n != 0 {
		t.Fatal(n)
	}
	db.Model(&model.ShinyRecord{}).Count(&n)
	if n != 1 {
		t.Fatal(n)
	}
}

func TestCatalogConflictingChineseValueFallsBackWithoutOverwrite(t *testing.T) {
	db := testutil.DB(t)
	wrong := "本地待核对"
	s := model.Species{Labels: model.Labels{Slug: "bulbasaur", Name: "Bulbasaur", NameZh: &wrong, NameStatus: "source_provided"}, NationalDex: 1}
	if err := db.Create(&s).Error; err != nil {
		t.Fatal(err)
	}
	d, _ := CatalogBundled()
	d.Records = d.Records[:1]
	report, err := ImportCatalog(db, d, true)
	if err != nil || len(report.Differences) != 1 {
		t.Fatal(report, err)
	}
	var entry model.CatalogEntry
	db.First(&entry)
	if entry.NameStatus != "conflict" {
		t.Fatal(entry.NameStatus)
	}
	var names map[string]string
	json.Unmarshal([]byte(entry.NamesJSON), &names)
	if DisplayCatalogName(names, entry.NameStatus, "bulbasaur") != "Bulbasaur" {
		t.Fatal("conflict did not fall back")
	}
	var got model.Species
	db.First(&got, s.ID)
	if got.NameZh == nil || *got.NameZh != wrong {
		t.Fatal("original reference overwritten")
	}
}
