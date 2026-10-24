package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoadDefault(t *testing.T) {
	// Setenv registers restoration before Unsetenv, preserving the caller's env.
	t.Setenv("SHINY_HTTP_PORT", "")
	if err := os.Unsetenv("SHINY_HTTP_PORT"); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil || cfg.Address() != "127.0.0.1:8080" {
		t.Fatalf("Load() = %+v, %v", cfg, err)
	}
}

func TestLoadPort(t *testing.T) {
	for _, value := range []string{"1", "8081", "65535"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("SHINY_HTTP_PORT", value)
			cfg, err := Load()
			if err != nil || cfg.Address() != "127.0.0.1:"+value {
				t.Fatalf("Load() = %+v, %v", cfg, err)
			}
		})
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	for _, value := range []string{"", "0", "-1", "65536", "999999999999999999999999", "8080.5", " 8080", "+8080", "localhost:8080", "private-invalid-input"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("SHINY_HTTP_PORT", value)
			_, err := Load()
			if err == nil {
				t.Fatal("invalid port was accepted")
			}
			if strings.Contains(err.Error(), "private-invalid-input") {
				t.Fatal("configuration error echoed raw input")
			}
		})
	}
}
