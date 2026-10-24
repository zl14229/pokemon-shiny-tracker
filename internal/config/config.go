// Package config loads the small, local-only T001 runtime configuration.
package config

import (
	"fmt"
	"github.com/zl14229/pokemon-shiny-tracker/internal/database"
	"net"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	Port   int
	DBPath string
}

// Load reads process environment. It does not read .env or any credential file.
func Load() (Config, error) {
	dbPath, exists := os.LookupEnv("SHINY_DB_PATH")
	if !exists {
		dataDir := os.Getenv("XDG_DATA_HOME")
		if dataDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return Config{}, err
			}
			dataDir = filepath.Join(home, ".local", "share")
		}
		dbPath = filepath.Join(dataDir, "pokemon-shiny-tracker", "tracker.sqlite3")
	}
	if err := database.ValidatePath(dbPath); err != nil {
		return Config{}, err
	}
	value, exists := os.LookupEnv("SHINY_HTTP_PORT")
	if !exists {
		return Config{Port: 8080, DBPath: dbPath}, nil
	}
	// Only ASCII decimal digits are accepted, without signs or whitespace.
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return Config{}, fmt.Errorf("SHINY_HTTP_PORT must be an integer from 1 to 65535")
		}
	}
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("SHINY_HTTP_PORT must be an integer from 1 to 65535")
	}
	return Config{Port: port, DBPath: dbPath}, nil
}

// Address always binds to IPv4 loopback for local single-user operation.
func (c Config) Address() string {
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(c.Port))
}
