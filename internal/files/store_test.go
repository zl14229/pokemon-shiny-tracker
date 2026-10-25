package files

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImmutableFilesAndIntegrity(t *testing.T) {
	s := Store{Root: filepath.Join(t.TempDir(), "blobs")}
	a, e := s.Put("匿名.txt", []byte("anonymous"))
	if e != nil {
		t.Fatal(e)
	}
	b, e := s.Put("匿名.txt", []byte("different"))
	if e != nil || a.Key == b.Key {
		t.Fatal(e)
	}
	data, e := s.Read(a.Key, a.SHA256, a.Size)
	if e != nil || string(data) != "anonymous" {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(s.Root, a.Key), []byte("damaged"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Read(a.Key, a.SHA256, a.Size); e == nil {
		t.Fatal("corruption ignored")
	}
	st, _ := os.Stat(filepath.Join(s.Root, b.Key))
	if st.Mode().Perm() != 0600 {
		t.Fatal("unsafe mode")
	}
}
func TestTraversalSymlinksAndTypes(t *testing.T) {
	s := Store{Root: filepath.Join(t.TempDir(), "blobs")}
	for _, name := range []string{"../x.txt", `..\x.txt`, "/x.txt", "CON.txt", "LPT1.txt", "bad?.txt", "bad\n.txt", ".hidden.txt", "wrong.exe", "wrong.svg"} {
		if _, e := s.Put(name, []byte("text")); e == nil {
			t.Fatal("accepted name", name)
		}
	}
	if _, e := s.Put("x.png", []byte("not png")); e == nil {
		t.Fatal("forged type")
	}
	if _, e := s.Put("x.txt", make([]byte, MaxSize+1)); e == nil {
		t.Fatal("oversize")
	}
	a, e := s.Put("x.txt", []byte("anonymous"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Read("../secret", a.SHA256, a.Size); e == nil {
		t.Fatal("traversal read")
	}
	outside := filepath.Join(t.TempDir(), "private.txt")
	_ = os.WriteFile(outside, []byte("private"), 0600)
	_ = os.Remove(filepath.Join(s.Root, a.Key))
	_ = os.Symlink(outside, filepath.Join(s.Root, a.Key))
	if _, e = s.Read(a.Key, a.SHA256, a.Size); e == nil {
		t.Fatal("symlink read")
	}
	link := filepath.Join(t.TempDir(), "root-link")
	_ = os.Symlink(s.Root, link)
	if _, e = (Store{Root: link}).Put("x.txt", []byte("text")); e == nil {
		t.Fatal("root symlink accepted")
	}
	if data, _ := os.ReadFile(outside); string(data) != "private" {
		t.Fatal("outside changed")
	}
}
func TestImageSizeAndHeader(t *testing.T) {
	var b bytes.Buffer
	_ = png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	if media, e := Validate("image.png", b.Bytes()); e != nil || media != "image/png" {
		t.Fatal(e)
	}
	if _, e := Validate("image.jpg", b.Bytes()); e == nil {
		t.Fatal("extension mismatch")
	}
	if _, e := Validate(strings.Repeat("x", 201)+".png", b.Bytes()); e == nil {
		t.Fatal("long name")
	}
}

func TestRejectsOversizedImageAndSharedRoot(t *testing.T) {
	var b bytes.Buffer
	_ = png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 10001, 1)))
	if _, e := Validate("wide.png", b.Bytes()); e == nil {
		t.Fatal("oversized dimensions")
	}
	root := filepath.Join(t.TempDir(), "shared")
	if e := os.Mkdir(root, 0755); e != nil {
		t.Fatal(e)
	}
	if e := os.Chmod(root, 0755); e != nil {
		t.Fatal(e)
	}
	if _, e := (Store{Root: root}).Put("x.txt", []byte("text")); e == nil {
		t.Fatal("nonprivate root accepted")
	}
}
