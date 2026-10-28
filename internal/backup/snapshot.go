// Package backup creates portable snapshots and restores only into new locations.
package backup

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/zl14229/pokemon-shiny-tracker/internal/database"
)

const MaxDatabase int64 = 512 << 20
const MaxArchive int64 = 2 << 30

var tables = []string{"hunt_plans", "hunt_plan_targets", "hunt_data_batches", "hunt_facts", "pokemon_hunt_methods", "shiny_availabilities", "catalog_entries", "attachments", "external_references", "forms", "games", "hunts", "import_batches", "import_rows", "reference_batches", "shiny_histories", "shiny_records", "species"}

func fileDSN(path, mode string) string {
	u := url.URL{Scheme: "file", Path: path}
	q := url.Values{"mode": {mode}, "_foreign_keys": {"on"}, "_busy_timeout": {"5000"}}
	u.RawQuery = q.Encode()
	return u.String()
}
func readDB(path string) (*sql.DB, error) {
	if e := database.ValidatePath(path); e != nil {
		return nil, e
	}
	st, e := os.Lstat(path)
	if e != nil || !st.Mode().IsRegular() || st.Size() > MaxDatabase {
		return nil, errors.New("数据库不存在、类型不符或超出512MiB")
	}
	db, e := sql.Open("sqlite3", fileDSN(path, "ro"))
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, e = db.Exec("PRAGMA query_only=ON; PRAGMA trusted_schema=OFF"); e != nil {
		_ = db.Close()
		return nil, e
	}
	return db, nil
}
func databaseCheck(ctx context.Context, db *sql.DB) error {
	var version int
	if e := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); e != nil {
		return e
	}
	if version != database.CurrentVersion {
		return errors.New("备份数据库版本不匹配；不能自动迁移或覆盖")
	}
	for _, name := range tables {
		var n int
		if e := db.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", name).Scan(&n); e != nil || n != 1 {
			return errors.New("应用数据库结构不完整")
		}
	}
	var result string
	if e := db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&result); e != nil || result != "ok" {
		return errors.New("数据库完整性检查失败")
	}
	rows, e := db.QueryContext(ctx, "PRAGMA foreign_key_check")
	if e != nil {
		return e
	}
	defer rows.Close()
	if rows.Next() {
		return errors.New("数据库存在外键问题")
	}
	return rows.Err()
}
func snapshot(ctx context.Context, source, dest string) error {
	src, e := readDB(source)
	if e != nil {
		return e
	}
	defer src.Close()
	var pages, size, version int64
	if e = src.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); e != nil || version != database.CurrentVersion {
		return errors.New("源数据库版本不匹配")
	}
	if e = src.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pages); e != nil {
		return e
	}
	if e = src.QueryRowContext(ctx, "PRAGMA page_size").Scan(&size); e != nil || pages < 1 || size < 1 || pages > MaxDatabase/size {
		return errors.New("数据库快照超过512MiB")
	}
	f, e := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if e != nil {
		return e
	}
	_ = f.Close()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(dest)
		}
	}()
	dst, e := sql.Open("sqlite3", fileDSN(dest, "rw"))
	if e != nil {
		return e
	}
	defer dst.Close()
	dst.SetMaxOpenConns(1)
	dst.SetMaxIdleConns(1)
	sc, e := src.Conn(ctx)
	if e != nil {
		return e
	}
	defer sc.Close()
	dc, e := dst.Conn(ctx)
	if e != nil {
		return e
	}
	e = sc.Raw(func(s any) error {
		return dc.Raw(func(d any) error {
			ss, yes := s.(*sqlite3.SQLiteConn)
			dd, other := d.(*sqlite3.SQLiteConn)
			if !yes || !other {
				return errors.New("unsupported SQLite driver")
			}
			b, err := dd.Backup("main", ss, "main")
			if err != nil {
				return err
			}
			for {
				if err = ctx.Err(); err != nil {
					_ = b.Finish()
					return err
				}
				done, step := b.Step(128)
				if step != nil {
					_ = b.Finish()
					return step
				}
				if int64(b.PageCount()) > MaxDatabase/size {
					_ = b.Finish()
					return errors.New("数据库快照超过512MiB")
				}
				if done {
					return b.Finish()
				}
				select {
				case <-ctx.Done():
					_ = b.Finish()
					return ctx.Err()
				case <-time.After(time.Millisecond * 5):
				}
			}
		})
	})
	_ = dc.Close()
	if e != nil {
		return e
	}
	if _, e = dst.ExecContext(ctx, "PRAGMA journal_mode=DELETE"); e != nil {
		return e
	}
	if e = dst.Close(); e != nil {
		return e
	}
	check, e := readDB(dest)
	if e != nil {
		return e
	}
	e = databaseCheck(ctx, check)
	_ = check.Close()
	if e != nil {
		return fmt.Errorf("快照检查失败: %w", e)
	}
	ok = true
	return nil
}
