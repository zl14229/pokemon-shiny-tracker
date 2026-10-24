package reference

import (
	_ "embed"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"gorm.io/gorm"
)

// The normalized subset was collected using the cached, paced Stage client.
// It contains necessary structured facts/evidence, not source prose or artwork.
//
//go:embed seed.json
var bundled []byte

func Bundled() (Dataset, error) { return Decode(bundled) }

// EnsureSeed never contacts a source or overwrites pre-existing reference facts.
func EnsureSeed(db *gorm.DB) error {
	for _, entity := range []any{&model.Species{}, &model.Form{}, &model.Game{}} {
		var count int64
		if err := db.Model(entity).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
	}
	d, err := Bundled()
	if err != nil {
		return err
	}
	_, err = Import(db, d, true)
	return err
}
