// Command reference keeps network acquisition separate from local data application.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/zl14229/pokemon-shiny-tracker/internal/config"
	"github.com/zl14229/pokemon-shiny-tracker/internal/database"
	"github.com/zl14229/pokemon-shiny-tracker/internal/reference"
)

func ids(value string) ([]int, error) {
	if value == "" {
		return nil, nil
	}
	parts := strings.Split(value, ",")
	if len(parts) > 50 {
		return nil, errors.New("at most 50 IDs per selection")
	}
	seen := map[int]bool{}
	result := []int{}
	for _, part := range parts {
		id, err := strconv.Atoi(part)
		if err != nil || id < 1 || seen[id] {
			return nil, errors.New("IDs must be positive unique integers")
		}
		seen[id] = true
		result = append(result, id)
	}
	return result, nil
}

func run() error {
	mode := flag.String("mode", "preview", "stage, preview or apply")
	species := flag.String("species", "25,26,133", "comma-separated PokéAPI species IDs")
	versions := flag.String("versions", "1,2,40,41", "comma-separated PokéAPI version IDs")
	input := flag.String("input", "", "normalized staged dataset path")
	output := flag.String("output", "", "stage output path")
	cache := flag.String("cache", "", "absolute cache directory outside Git")
	offline := flag.Bool("offline", false, "read source cache only")
	refresh := flag.Bool("refresh", false, "explicitly refresh source cache, without applying data")
	flag.Parse()
	if flag.NArg() != 0 {
		return errors.New("unexpected arguments")
	}
	if *mode == "stage" {
		if *output == "" {
			return errors.New("stage requires -output")
		}
		s, err := ids(*species)
		if err != nil {
			return err
		}
		v, err := ids(*versions)
		if err != nil {
			return err
		}
		if *cache == "" {
			dir, err := reference.DefaultCacheDir()
			if err != nil {
				return err
			}
			*cache = dir
		}
		client, err := reference.NewClient(*cache)
		if err != nil {
			return err
		}
		client.Offline = *offline
		client.Refresh = *refresh
		dataset, err := client.Stage(context.Background(), s, v)
		if err != nil {
			return err
		}
		file, err := os.OpenFile(*output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		if _, err := file.Write(append(dataset.Canonical(), '\n')); err != nil {
			file.Close()
			return err
		}
		if err := file.Close(); err != nil {
			return err
		}
		fmt.Printf("staged %d validated resources; no database opened\n", len(dataset.Resources))
		return nil
	}
	if (*mode != "preview" && *mode != "apply") || *input == "" {
		return errors.New("use stage, or preview/apply with -input")
	}
	data, err := os.ReadFile(*input)
	if err != nil {
		return err
	}
	dataset, err := reference.Decode(data)
	if err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	db, err := database.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	conn, _ := db.DB()
	defer conn.Close()
	report, err := reference.Import(db, dataset, *mode == "apply")
	if encoded, encodeErr := json.MarshalIndent(report, "", "  "); encodeErr == nil {
		fmt.Println(string(encoded))
	}
	return err
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
