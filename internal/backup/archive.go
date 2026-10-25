package backup

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/zl14229/pokemon-shiny-tracker/internal/database"
	"github.com/zl14229/pokemon-shiny-tracker/internal/files"
)

const format = "pokemon-shiny-tracker-backup-v1"

type Entry struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}
type Manifest struct {
	Format        string    `json:"format"`
	SchemaVersion int       `json:"schema_version"`
	CreatedAt     time.Time `json:"created_at"`
	Database      Entry     `json:"database"`
	Files         []Entry   `json:"files"`
}

func digestFile(path, name string) (Entry, error) {
	f, e := os.Open(path)
	if e != nil {
		return Entry{}, e
	}
	defer f.Close()
	h := sha256.New()
	n, e := io.Copy(h, io.LimitReader(f, MaxDatabase+1))
	if e != nil || n > MaxDatabase {
		return Entry{}, errors.New("文件读取或大小检查失败")
	}
	return Entry{Path: name, SHA256: hex.EncodeToString(h.Sum(nil)), Size: n}, nil
}
func attachmentEntries(ctx context.Context, path string) ([]Entry, error) {
	db, e := readDB(path)
	if e != nil {
		return nil, e
	}
	defer db.Close()
	rows, e := db.QueryContext(ctx, "SELECT kind,name,media_type,storage_key,sha256,size_bytes,url FROM attachments ORDER BY id")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	result := []Entry{}
	for rows.Next() {
		var kind, name, media string
		var key, sha, link *string
		var size *int64
		if e = rows.Scan(&kind, &name, &media, &key, &sha, &size, &link); e != nil {
			return nil, e
		}
		if kind == "file" {
			if key == nil || sha == nil || size == nil || !files.ValidKey(*key) || !files.ValidName(name) || len(*sha) != 64 || *size < 1 || *size > files.MaxSize || link != nil {
				return nil, errors.New("附件元数据无效")
			}
			if media != "image/png" && media != "image/jpeg" && media != "image/gif" && media != "application/pdf" && media != "text/plain; charset=utf-8" {
				return nil, errors.New("附件媒体类型无效")
			}
			result = append(result, Entry{Path: "files/" + *key, SHA256: *sha, Size: *size})
		} else if kind != "link" || key != nil || sha != nil || size != nil || link == nil || !validLink(*link) {
			return nil, errors.New("链接元数据无效")
		}
		if len(result) > 100000 {
			return nil, errors.New("附件数量超限")
		}
	}
	return result, rows.Err()
}
func validLink(value string) bool {
	u, e := url.Parse(value)
	return e == nil && u.Hostname() != "" && u.User == nil && (u.Scheme == "http" || u.Scheme == "https") && len(value) <= 2048
}
func checkBlobContent(ctx context.Context, dbPath, root string) error {
	db, e := readDB(dbPath)
	if e != nil {
		return e
	}
	defer db.Close()
	rows, e := db.QueryContext(ctx, "SELECT name,media_type,storage_key,sha256,size_bytes FROM attachments WHERE kind='file'")
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var name, media, key, hash string
		var size int64
		if e = rows.Scan(&name, &media, &key, &hash, &size); e != nil {
			return e
		}
		data, e := (files.Store{Root: root}).Read(key, hash, size)
		if e != nil {
			return e
		}
		actual, e := files.Validate(name, data)
		if e != nil || actual != media {
			return errors.New("附件内容与元数据类型不符")
		}
	}
	return rows.Err()
}
func privateOutput(path string) (*os.File, error) {
	if e := database.ValidatePath(path); e != nil {
		return nil, e
	}
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return nil, e
	}
	return os.CreateTemp(filepath.Dir(path), ".pst-export-*")
}

// Publish a completed private file without ever replacing an existing path.
func publish(f *os.File, path string) error {
	if e := f.Sync(); e != nil {
		return e
	}
	if e := f.Close(); e != nil {
		return e
	}
	if e := os.Link(f.Name(), path); e != nil {
		return errors.New("输出已存在或无法独占发布，请选择新文件")
	}
	return syncDirectory(filepath.Dir(path))
}

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
func Backup(ctx context.Context, source, output string) (Manifest, error) {
	var m Manifest
	work, e := os.MkdirTemp("", "pst-snapshot-")
	if e != nil {
		return m, e
	}
	defer os.RemoveAll(work)
	dbpath := filepath.Join(work, "database.sqlite3")
	if e = snapshot(ctx, source, dbpath); e != nil {
		return m, e
	}
	entry, e := digestFile(dbpath, "database.sqlite3")
	if e != nil {
		return m, e
	}
	blobs, e := attachmentEntries(ctx, dbpath)
	if e != nil {
		return m, e
	}
	m = Manifest{Format: format, SchemaVersion: database.CurrentVersion, CreatedAt: time.Now().UTC(), Database: entry, Files: blobs}
	f, e := privateOutput(output)
	if e != nil {
		return m, e
	}
	defer os.Remove(f.Name())
	defer f.Close()
	z := zip.NewWriter(f)
	ok := false
	defer func() {
		if !ok {
			_ = z.Close()
		}
	}()
	w, e := z.Create("database.sqlite3")
	if e != nil {
		return m, e
	}
	sourceDB, e := os.Open(dbpath)
	if e != nil {
		return m, e
	}
	_, e = io.Copy(w, sourceDB)
	_ = sourceDB.Close()
	if e != nil {
		return m, e
	}
	total := entry.Size
	for _, b := range blobs {
		if e = ctx.Err(); e != nil {
			return m, e
		}
		total += b.Size
		if total > MaxArchive {
			return m, errors.New("备份超过2GiB解压总量")
		}
		data, e := (files.Store{Root: source + ".files"}).Read(strings.TrimPrefix(b.Path, "files/"), b.SHA256, b.Size)
		if e != nil {
			return m, errors.New("数据库引用的附件缺失或完整性失败，备份取消")
		}
		w, e = z.Create(b.Path)
		if e != nil {
			return m, e
		}
		if _, e = w.Write(data); e != nil {
			return m, e
		}
	}
	w, e = z.Create("manifest.json")
	if e != nil {
		return m, e
	}
	if e = json.NewEncoder(w).Encode(m); e != nil {
		return m, e
	}
	if e = z.Close(); e != nil {
		return m, e
	}
	ok = true
	if _, e = Verify(ctx, f.Name()); e != nil {
		return m, e
	}
	if e = publish(f, output); e != nil {
		return m, e
	}
	return m, nil
}
func extract(ctx context.Context, archive, stage string) (Manifest, error) {
	var m Manifest
	if e := database.ValidatePath(archive); e != nil {
		return m, e
	}
	st, e := os.Stat(archive)
	if e != nil || st.Size() > MaxArchive+(16<<20) {
		return m, errors.New("备份文件过大或不可读取")
	}
	z, e := zip.OpenReader(archive)
	if e != nil {
		return m, e
	}
	defer z.Close()
	if len(z.File) > 100002 {
		return m, errors.New("备份条目过多")
	}
	parts := map[string]*zip.File{}
	var total uint64
	for _, f := range z.File {
		if !f.Mode().IsRegular() || parts[f.Name] != nil || f.UncompressedSize64 > uint64(MaxDatabase) {
			return m, errors.New("重复、非法或超限的备份条目")
		}
		total += f.UncompressedSize64
		if total > uint64(MaxArchive) {
			return m, errors.New("备份解压大小超限")
		}
		parts[f.Name] = f
	}
	mf := parts["manifest.json"]
	if mf == nil || mf.UncompressedSize64 > 16<<20 {
		return m, errors.New("缺少或超限的备份清单")
	}
	r, e := mf.Open()
	if e != nil {
		return m, e
	}
	decoder := json.NewDecoder(io.LimitReader(r, 16<<20+1))
	decoder.DisallowUnknownFields()
	e = decoder.Decode(&m)
	var trailing any
	if e == nil && decoder.Decode(&trailing) != io.EOF {
		e = errors.New("extra manifest data")
	}
	_ = r.Close()
	if e != nil || m.Format != format || m.SchemaVersion != database.CurrentVersion || m.Database.Path != "database.sqlite3" {
		return m, errors.New("备份格式或版本无效")
	}
	expected := append([]Entry{m.Database}, m.Files...)
	allow := map[string]bool{"manifest.json": true}
	root, e := os.OpenRoot(stage)
	if e != nil {
		return m, e
	}
	defer root.Close()
	if len(m.Files) > 0 {
		if e = root.Mkdir("files", 0700); e != nil {
			return m, e
		}
	}
	for i, b := range expected {
		if e = ctx.Err(); e != nil {
			return m, e
		}
		if allow[b.Path] || len(b.SHA256) != 64 || b.Size < 1 || (i > 0 && (!strings.HasPrefix(b.Path, "files/") || !files.ValidKey(strings.TrimPrefix(b.Path, "files/")) || b.Size > files.MaxSize)) || (i == 0 && b.Size > MaxDatabase) {
			return m, errors.New("备份路径、数量或元数据无效")
		}
		allow[b.Path] = true
		part := parts[b.Path]
		if part == nil || part.UncompressedSize64 != uint64(b.Size) {
			return m, errors.New("备份文件缺失或大小不符")
		}
		in, e := part.Open()
		if e != nil {
			return m, e
		}
		out, e := root.OpenFile(b.Path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			_ = in.Close()
			return m, e
		}
		h := sha256.New()
		n, err := io.Copy(io.MultiWriter(out, h), io.LimitReader(in, b.Size+1))
		closeIn := in.Close()
		syncErr := out.Sync()
		closeOut := out.Close()
		if err != nil || closeIn != nil || syncErr != nil || closeOut != nil || n != b.Size || hex.EncodeToString(h.Sum(nil)) != b.SHA256 {
			return m, errors.New("备份内容或SHA256校验失败")
		}
	}
	if len(parts) != len(allow) {
		return m, errors.New("备份包含未声明条目")
	}
	db, e := readDB(filepath.Join(stage, "database.sqlite3"))
	if e != nil {
		return m, e
	}
	e = databaseCheck(ctx, db)
	_ = db.Close()
	if e != nil {
		return m, e
	}
	refs, e := attachmentEntries(ctx, filepath.Join(stage, "database.sqlite3"))
	if e != nil {
		return m, e
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].Path < refs[j].Path })
	listed := append([]Entry{}, m.Files...)
	sort.Slice(listed, func(i, j int) bool { return listed[i].Path < listed[j].Path })
	a, _ := json.Marshal(refs)
	b, _ := json.Marshal(listed)
	if string(a) != string(b) {
		return m, errors.New("附件清单与数据库不一致")
	}
	if e = checkBlobContent(ctx, filepath.Join(stage, "database.sqlite3"), filepath.Join(stage, "files")); e != nil {
		return m, e
	}
	return m, nil
}
func Verify(ctx context.Context, archive string) (Manifest, error) {
	stage, e := os.MkdirTemp("", "pst-verify-")
	if e != nil {
		return Manifest{}, e
	}
	defer os.RemoveAll(stage)
	return extract(ctx, archive, stage)
}
func Restore(ctx context.Context, archive, dest string) (Manifest, error) {
	var m Manifest
	if !filepath.IsAbs(dest) || filepath.Clean(dest) != dest {
		return m, errors.New("恢复目录必须为干净绝对路径")
	}
	if _, e := os.Lstat(dest); !os.IsNotExist(e) {
		return m, errors.New("恢复目录已存在或不可访问，禁止覆盖")
	}
	if e := database.ValidatePath(filepath.Join(dest, "database.sqlite3")); e != nil {
		return m, e
	}
	if e := os.MkdirAll(filepath.Dir(dest), 0700); e != nil {
		return m, e
	}
	stage, e := os.MkdirTemp(filepath.Dir(dest), ".pst-restore-")
	if e != nil {
		return m, e
	}
	defer os.RemoveAll(stage)
	m, e = extract(ctx, archive, stage)
	if e != nil {
		return m, e
	}
	if len(m.Files) > 0 {
		if e = os.Rename(filepath.Join(stage, "files"), filepath.Join(stage, "database.sqlite3.files")); e != nil {
			return m, e
		}
		if e = syncDirectory(filepath.Join(stage, "database.sqlite3.files")); e != nil {
			return m, e
		}
	}
	if e = syncDirectory(stage); e != nil {
		return m, e
	}
	if e = publishDirectory(stage, dest); e != nil {
		return m, e
	}
	return m, syncDirectory(filepath.Dir(dest))
}
