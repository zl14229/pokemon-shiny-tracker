package backup

import (
	"context"
	"encoding/json"
	"github.com/zl14229/pokemon-shiny-tracker/internal/database"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Export JSON is a portable data export, not a substitute for the ZIP restore format.
type stickyWriter struct {
	io.Writer
	err error
}

func (w *stickyWriter) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	n, e := w.Writer.Write(p)
	if e == nil && n < len(p) {
		e = io.ErrShortWrite
	}
	w.err = e
	return n, e
}

func Export(ctx context.Context, source, output string) error {
	work, e := os.MkdirTemp("", "pst-json-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(work)
	path := filepath.Join(work, "database.sqlite3")
	if e = snapshot(ctx, source, path); e != nil {
		return e
	}
	db, e := readDB(path)
	if e != nil {
		return e
	}
	defer db.Close()
	out, e := privateOutput(output)
	if e != nil {
		return e
	}
	defer os.Remove(out.Name())
	defer out.Close()
	w := &stickyWriter{Writer: out}
	_, e = io.WriteString(w, `{"format":"pokemon-shiny-tracker-export-v1","schema_version":`)
	if e != nil {
		return e
	}
	if e = json.NewEncoder(w).Encode(database.CurrentVersion); e != nil {
		return e
	}
	_, _ = io.WriteString(w, `,"created_at":`)
	_ = json.NewEncoder(w).Encode(time.Now().UTC())
	_, _ = io.WriteString(w, `,"tables":{`)
	for index, name := range tables {
		if e = ctx.Err(); e != nil {
			return e
		}
		if index > 0 {
			_, _ = io.WriteString(w, ",")
		}
		_ = json.NewEncoder(w).Encode(name)
		_, _ = io.WriteString(w, ":[")
		rows, e := db.QueryContext(ctx, `SELECT * FROM "`+name+`" ORDER BY id`)
		if e != nil {
			return e
		}
		columns, e := rows.Columns()
		if e != nil {
			_ = rows.Close()
			return e
		}
		first := true
		for rows.Next() {
			values := make([]any, len(columns))
			ptrs := make([]any, len(columns))
			for i := range values {
				ptrs[i] = &values[i]
			}
			if e = rows.Scan(ptrs...); e != nil {
				_ = rows.Close()
				return e
			}
			row := map[string]any{}
			for i, key := range columns {
				row[key] = values[i]
			}
			if !first {
				_, _ = io.WriteString(w, ",")
			}
			first = false
			if e = json.NewEncoder(w).Encode(row); e != nil {
				_ = rows.Close()
				return e
			}
		}
		e = rows.Err()
		_ = rows.Close()
		if e != nil {
			return e
		}
		_, _ = io.WriteString(w, "]")
	}
	_, e = io.WriteString(w, "}}\n")
	if e != nil {
		return e
	}
	return publish(out, output)
}
