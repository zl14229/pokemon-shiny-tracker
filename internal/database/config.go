// Package database owns SQLite connections and explicit, transactional migrations.
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
		TranslateError: true,
		NowFunc:        func() time.Time { return time.Now().UTC() },
		Logger: logger.New(log.New(os.Stderr, "", log.LstdFlags), logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			ParameterizedQueries:      true,
			IgnoreRecordNotFoundError: true,
		}),
	}
}
