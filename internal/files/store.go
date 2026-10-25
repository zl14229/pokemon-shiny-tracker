// Package files stores bounded immutable blobs, independent of user filenames.
package files

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/zl14229/pokemon-shiny-tracker/internal/database"
)

const MaxSize = 10 << 20

var keyPattern = regexp.MustCompile(`^[0-9a-f]{32}\.blob$`)

type Store struct{ Root string }
type Info struct {
	Key       string
	SHA256    string
	MediaType string
	Size      int64
}

func ValidName(name string) bool {
	if name == "" || len(name) > 200 || strings.TrimSpace(name) != name || strings.HasPrefix(name, ".") || strings.ContainsAny(name, "/\\:<>\"|?*") {
		return false
	}
	for _, c := range name {
		if unicode.IsControl(c) {
			return false
		}
	}
	stem := strings.TrimRight(strings.ToUpper(strings.Split(name, ".")[0]), " .")
	if stem == "CON" || stem == "PRN" || stem == "AUX" || stem == "NUL" || regexp.MustCompile(`^(COM|LPT)[1-9]$`).MatchString(stem) {
		return false
	}
	return utf8.ValidString(name)
}
func Validate(name string, data []byte) (string, error) {
	if !ValidName(name) || len(data) == 0 || len(data) > MaxSize {
		return "", errors.New("附件名称或大小无效")
	}
	media := http.DetectContentType(data)
	ext := strings.ToLower(filepath.Ext(name))
	allowed := map[string]string{".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".gif": "image/gif", ".pdf": "application/pdf", ".txt": "text/plain; charset=utf-8"}
	if allowed[ext] != media {
		return "", errors.New("仅支持匹配内容的PNG/JPEG/GIF/PDF/UTF8 TXT")
	}
	if strings.HasPrefix(media, "image/") {
		c, _, e := image.DecodeConfig(bytes.NewReader(data))
		if e != nil || c.Width < 1 || c.Height < 1 || c.Width > 10000 || c.Height > 10000 || int64(c.Width)*int64(c.Height) > 25000000 {
			return "", errors.New("图片格式或像素尺寸无效")
		}
	}
	if ext == ".txt" && !utf8.Valid(data) {
		return "", errors.New("文本须为UTF8")
	}
	return media, nil
}
func (s Store) open(create bool) (*os.Root, error) {
	if !filepath.IsAbs(s.Root) || filepath.Clean(s.Root) != s.Root {
		return nil, errors.New("invalid root")
	}
	if err := database.ValidatePath(filepath.Join(s.Root, "path-check")); err != nil {
		return nil, err
	}
	st, e := os.Lstat(s.Root)
	if os.IsNotExist(e) && create {
		if e = os.Mkdir(s.Root, 0700); e != nil && !os.IsExist(e) {
			return nil, e
		}
		st, e = os.Lstat(s.Root)
	}
	if e != nil {
		return nil, e
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 || st.Mode().Perm()&0077 != 0 {
		return nil, errors.New("附件根目录必须为私有真实目录")
	}
	return os.OpenRoot(s.Root)
}
func (s Store) Put(name string, data []byte) (Info, error) {
	media, e := Validate(name, data)
	if e != nil {
		return Info{}, e
	}
	root, e := s.open(true)
	if e != nil {
		return Info{}, e
	}
	defer root.Close()
	nonce := make([]byte, 16)
	if _, e = rand.Read(nonce); e != nil {
		return Info{}, e
	}
	key := hex.EncodeToString(nonce) + ".blob"
	f, e := root.OpenFile(key, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return Info{}, e
	}
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = root.Remove(key)
		}
	}()
	if _, e = f.Write(data); e != nil {
		return Info{}, e
	}
	if e = f.Sync(); e != nil {
		return Info{}, e
	}
	if e = f.Close(); e != nil {
		return Info{}, e
	}
	dir, e := root.Open(".")
	if e != nil {
		return Info{}, e
	}
	e = dir.Sync()
	_ = dir.Close()
	if e != nil {
		return Info{}, e
	}
	hash := sha256.Sum256(data)
	ok = true
	return Info{Key: key, SHA256: hex.EncodeToString(hash[:]), MediaType: media, Size: int64(len(data))}, nil
}
func (s Store) Read(key, sha string, size int64) ([]byte, error) {
	if !keyPattern.MatchString(key) || len(sha) != 64 || size < 1 || size > MaxSize {
		return nil, errors.New("无效的附件元数据")
	}
	root, e := s.open(false)
	if e != nil {
		return nil, e
	}
	defer root.Close()
	st, e := root.Lstat(key)
	if e != nil || !st.Mode().IsRegular() {
		return nil, errors.New("附件不是正常文件")
	}
	f, e := root.Open(key)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	data, e := io.ReadAll(io.LimitReader(f, MaxSize+1))
	if e != nil {
		return nil, e
	}
	hash := sha256.Sum256(data)
	if int64(len(data)) != size || hex.EncodeToString(hash[:]) != sha {
		return nil, errors.New("附件完整性检查失败")
	}
	return data, nil
}

// RemoveNew only cleans up a blob created by a failed upload. Normal deletion
// retains immutable files so a concurrent SQLite snapshot can still copy them.
func (s Store) RemoveNew(key string) error {
	if !keyPattern.MatchString(key) {
		return errors.New("invalid key")
	}
	root, e := s.open(false)
	if e != nil {
		return e
	}
	defer root.Close()
	return root.Remove(key)
}
func ValidKey(key string) bool { return keyPattern.MatchString(key) }
