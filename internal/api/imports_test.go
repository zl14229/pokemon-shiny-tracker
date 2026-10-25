package api

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/zl14229/pokemon-shiny-tracker/internal/service"
	"github.com/zl14229/pokemon-shiny-tracker/internal/testutil"
	"mime/multipart"
	"net/http/httptest"
	"testing"
)

func workbookRequest(t *testing.T, r *gin.Engine, route string, file []byte, mapping string) *httptest.ResponseRecorder {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	f, e := w.CreateFormFile("file", "anonymous.xlsx")
	if e != nil {
		t.Fatal(e)
	}
	_, _ = f.Write(file)
	if mapping != "" {
		_ = w.WriteField("mapping", mapping)
	}
	_ = w.Close()
	req := httptest.NewRequest("POST", route, &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	out := httptest.NewRecorder()
	r.ServeHTTP(out, req)
	return out
}
func TestImportHTTPRequiresPreviewAndConfirmation(t *testing.T) {
	db := testutil.DB(t)
	testutil.Seed(t, db)
	r := NewRouter(db)
	f := testutil.XLSX(t, [][]string{{"编号", "日期"}, {"25", "2022"}})
	if w := workbookRequest(t, r, "/api/v1/imports/inspect", f, ""); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	m := `{"sheet":"匿名样本","header_row":1,"columns":{"A":"current_national_dex","B":"obtained_date"}}`
	w := workbookRequest(t, r, "/api/v1/imports/preview", f, m)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var body struct {
		Data service.ImportPreview `json:"data"`
	}
	if e := json.Unmarshal(w.Body.Bytes(), &body); e != nil {
		t.Fatal(e)
	}
	payload, _ := json.Marshal(service.ImportConfirm{Token: body.Data.Token, Rows: []int{2}, AcknowledgeWarnings: true})
	for i := 0; i < 2; i++ {
		w = jsonRequest(r, "POST", "/api/v1/imports/confirm", string(payload))
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	if w = request(r, "GET", "/api/v1/imports/batches/1"); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w = request(r, "GET", "/api/v1/imports/batches"); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w = workbookRequest(t, r, "/api/v1/imports/inspect", []byte("not a workbook"), ""); w.Code != 400 {
		t.Fatal(w.Code)
	}
	if w = jsonRequest(r, "POST", "/api/v1/imports/confirm", `{"token":"no-preview","rows":[2]}`); w.Code != 400 {
		t.Fatal(w.Code)
	}
}

func TestImportRejectsCrossSiteWrites(t *testing.T) {
	r := NewRouter(testutil.DB(t))
	req := httptest.NewRequest("POST", "/api/v1/imports/confirm", bytes.NewBufferString(`{"token":"x","rows":[2]}`))
	req.Header.Set("Origin", "https://unrelated.example")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatal(w.Code)
	}
}
