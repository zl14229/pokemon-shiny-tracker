// Package config loads the small, local-only T001 runtime configuration.
package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
)

type Config struct {
	Port int
}

// Load reads process environment. It does not read .env or any credential file.
func Load() (Config, error) {
	value, exists := os.LookupEnv("SHINY_HTTP_PORT")
	if !exists {
		return Config{Port: 8080}, nil
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
	return Config{Port: port}, nil
}

// Address always binds to IPv4 loopback for local single-user operation.
func (c Config) Address() string {
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(c.Port))
}
