// data requires explicit source/output paths. It never defaults to a user's live DB.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/zl14229/pokemon-shiny-tracker/internal/backup"
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
	dest := f.String("destination", "", "new restore directory")
	if e := f.Parse(args[1:]); e != nil || f.NArg() != 0 {
		return errors.New("invalid command flags")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	var e error
	switch cmd {
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
