package database

import (
	"errors"
	"fmt"

	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"gorm.io/gorm"
)

type migration struct {
	version int
	apply   func(*gorm.DB) error
}

var migrations = []migration{{1, func(tx *gorm.DB) error {
	if err := tx.AutoMigrate(&model.Game{}, &model.Species{}, &model.Form{}); err != nil {
		return err
	}
	if err := tx.Exec("CREATE UNIQUE INDEX uidx_form_species ON forms(id,species_id)").Error; err != nil {
		return err
	}
	return tx.AutoMigrate(&model.Hunt{}, &model.ShinyRecord{}, &model.ShinyHistory{})
}}, {2, func(tx *gorm.DB) error {
	return tx.Migrator().CreateTable(&model.ReferenceBatch{}, &model.ExternalReference{})
}}}

func Migrate(db *gorm.DB) error {
	return migrate(db, migrations)
}

func migrate(db *gorm.DB, steps []migration) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var version int
		if err := tx.Raw("PRAGMA user_version").Scan(&version).Error; err != nil {
			return err
		}
		if len(steps) == 0 || version > steps[len(steps)-1].version {
			return errors.New("unsupported database schema version")
		}
		if version == 0 {
			var count int64
			if err := tx.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&count).Error; err != nil {
				return err
			}
			if count != 0 {
				return errors.New("unrecognized existing database")
			}
		}
		for _, step := range steps {
			if step.version <= version {
				continue
			}
			if step.version != version+1 {
				return errors.New("non-sequential database migration")
			}
			if err := step.apply(tx); err != nil {
				return err
			}
			if err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", step.version)).Error; err != nil {
				return err
			}
			version = step.version
		}
		return nil
	})
}
