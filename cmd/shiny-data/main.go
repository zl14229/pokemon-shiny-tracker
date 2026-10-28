// data requires explicit source/output paths. It never defaults to a user's live DB.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/zl14229/pokemon-shiny-tracker/internal/backup"
	"github.com/zl14229/pokemon-shiny-tracker/internal/database"
	"github.com/zl14229/pokemon-shiny-tracker/internal/huntdata"
	"io"
	"os"
	"time"
)

func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: shiny-data backup|export|verify|restore --database ABS --output NEW_FILE --archive ABS --destination NEW_DIR")
	}
	cmd := args[0]
	f := flag.NewFlagSet(cmd, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	db := f.String("database", "", "source database")
	output := f.String("output", "", "new file")
	archive := f.String("archive", "", "backup ZIP")
	snapshot := f.String("snapshot", "", "staged hunt JSON; empty uses bundled snapshot")
	reviewed := f.String("reviewed-sha", "", "checksum from reviewed preview")
	version := f.String("version", "", "batch to retract")
	dest := f.String("destination", "", "new restore directory")
	if e := f.Parse(args[1:]); e != nil || f.NArg() != 0 {
		return errors.New("invalid command flags")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	var e error
	switch cmd {
	case "hunt-preview", "hunt-apply", "hunt-rollback":
		if *db == "" {
			return errors.New("hunt reference operation requires --database")
		}
		store, err := database.Open(*db)
		if err != nil {
			return err
		}
		conn, _ := store.DB()
		defer conn.Close()
		if cmd == "hunt-rollback" {
			if *version == "" || *reviewed == "" {
				return errors.New("rollback requires --version and --reviewed-sha")
			}
			e = huntdata.Rollback(store, *version, *reviewed)
			break
		}
		d, err := huntdata.Bundled()
		if *snapshot != "" {
			input, openErr := os.Open(*snapshot)
			if openErr != nil {
				return openErr
			}
			d, err = huntdata.Decode(input)
			_ = input.Close()
		}
		if err != nil {
			return err
		}
		report, err := huntdata.Import(store, d, cmd == "hunt-apply", *reviewed)
		if err != nil {
			return err
		}
		return json.NewEncoder(out).Encode(report)
	case "backup":
		if *db == "" || *output == "" {
			return errors.New("backup requires --database and --output")
		}
		_, e = backup.Backup(ctx, *db, *output)
	case "export":
		if *db == "" || *output == "" {
			return errors.New("export requires --database and --output")
		}
		e = backup.Export(ctx, *db, *output)
	case "verify":
		if *archive == "" {
			return errors.New("verify requires --archive")
		}
		_, e = backup.Verify(ctx, *archive)
	case "restore":
		if *archive == "" || *dest == "" {
			return errors.New("restore requires --archive and --destination (must not exist)")
		}
		_, e = backup.Restore(ctx, *archive, *dest)
	default:
		return errors.New("unknown data command")
	}
	if e != nil {
		return e
	}
	_, e = fmt.Fprintln(out, "PASS:", cmd, "completed; no existing destination was overwritten")
	return e
}
func main() {
	if e := run(os.Args[1:], os.Stdout); e != nil {
		fmt.Fprintln(os.Stderr, "data operation failed:", e)
		os.Exit(1)
	}
}
