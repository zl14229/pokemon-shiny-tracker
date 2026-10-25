package api

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"github.com/zl14229/pokemon-shiny-tracker/internal/testutil"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
)

func attachmentRequest(t *testing.T, r *gin.Engine, name string, data []byte) *httptest.ResponseRecorder {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	f, e := w.CreateFormFile("file", name)
	if e != nil {
		t.Fatal(e)
	}
	_, _ = f.Write(data)
	_ = w.Close()
	q := httptest.NewRequest("POST", "/api/v1/shiny-records/1/attachments", &b)
	q.Header.Set("Content-Type", w.FormDataContentType())
	out := httptest.NewRecorder()
	r.ServeHTTP(out, q)
	return out
}
func TestAttachmentHTTP(t *testing.T) {
	db := testutil.DB(t)
	testutil.Seed(t, db)
	_ = db.Create(&model.ShinyRecord{CurrentSpeciesID: 1}).Error
	r := NewRouter(db)
	w := attachmentRequest(t, r, "匿名.txt", []byte("anonymous"))
	if w.Code != 201 || strings.Contains(w.Body.String(), "storage_key") {
		t.Fatal(w.Code, w.Body.String())
	}
	var body struct{ Data model.Attachment }
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	w = request(r, "GET", "/api/v1/attachments/1/content")
	if w.Code != 200 || w.Body.String() != "anonymous" || w.Header().Get("X-Content-Type-Options") != "nosniff" || !strings.HasPrefix(w.Header().Get("Content-Disposition"), "attachment") {
		t.Fatal(w.Code, w.Header())
	}
	if w = request(r, "GET", "/api/v1/shiny-records/1/attachments"); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w = jsonRequest(r, "POST", "/api/v1/shiny-records/1/links", `{"name":"proof","url":"https://example.com/proof"}`); w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = request(r, "DELETE", "/api/v1/attachments/1"); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w = request(r, "GET", "/api/v1/attachments/1/content"); w.Code != 404 {
		t.Fatal("deleted file still served", w.Code)
	}
	for _, name := range []string{"../../escape.txt", `..\escape.txt`, "bad.exe", "NUL.txt"} {
		w = attachmentRequest(t, r, name, []byte("anonymous"))
		if w.Code != 400 {
			t.Fatal(name, w.Code)
		}
	}
	w = attachmentRequest(t, r, "false.png", []byte("anonymous"))
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
}
