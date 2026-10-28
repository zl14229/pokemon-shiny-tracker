package huntdata

import (
	"bytes"
	_ "embed"
	"github.com/zl14229/pokemon-shiny-tracker/internal/reference"
	"gorm.io/gorm"
)

//go:embed bundled.json
var bundled []byte

func Bundled() (Dataset, error) { return Decode(bytes.NewReader(bundled)) }
func EnsureBundled(db *gorm.DB) error {
	// Reference initialization is local; no network reads on startup.
	if e := reference.EnsureSeed(db); e != nil {
		return e
	}
	d, e := Bundled()
	if e != nil {
		return e
	}
	_, e = Import(db, d, true, Digest(d))
	return e
}
