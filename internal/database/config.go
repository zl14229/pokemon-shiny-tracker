// Package database holds GORM configuration for the later T005 connection layer.
// T001 does not install a driver, open a database, or create any models.
package database

import (
	"log"
	"os"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewConfig prepares conservative ORM options without connecting to storage.
// Parameterized logging keeps bound personal values out of future SQL logs.
func NewConfig() *gorm.Config {
	return &gorm.Config{
		Logger: logger.New(log.New(os.Stderr, "", log.LstdFlags), logger.Config{
			SlowThreshold:        200 * time.Millisecond,
			LogLevel:             logger.Warn,
			ParameterizedQueries: true,
		}),
	}
}
