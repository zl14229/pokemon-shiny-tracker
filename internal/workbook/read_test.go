package workbook_test

import (
	"archive/zip"
	"bytes"
	"github.com/zl14229/pokemon-shiny-tracker/internal/testutil"
	"github.com/zl14229/pokemon-shiny-tracker/internal/workbook"
	"strings"
	"testing"
)

func TestReadSparseAndUnicode(t *testing.T) {
	b, e := workbook.Read(testutil.XLSX(t, [][]string{{"编号", "球种"}, {"25", "<匿名&>"}, {"", ""}}))
	if e != nil || len(b.Sheets) != 1 || b.Sheets[0].Rows[1].Cells["B"].Value != "<匿名&>" {
		t.Fatal(b, e)
	}
}
func zipped(t *testing.T, parts map[string]string) []byte {
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for k, v := range parts {
		f, e := w.Create(k)
		if e != nil {
			t.Fatal(e)
		}
		_, _ = f.Write([]byte(v))
	}
	_ = w.Close()
	return b.Bytes()
}
func TestRefusesTraversalMacrosBombAndInvalidZIP(t *testing.T) {
	for _, data := range [][]byte{[]byte("invalid"), make([]byte, workbook.MaxFile+1), zipped(t, map[string]string{"../escape": "bad"}), zipped(t, map[string]string{"xl/vbaProject.bin": "bad"}), zipped(t, map[string]string{"big": strings.Repeat("x", 33<<20)})} {
		if _, e := workbook.Read(data); e == nil {
			t.Fatal("accepted unsafe archive")
		}
	}
}
func TestFormulaCacheAndDateEpoch(t *testing.T) {
	for _, epoch := range []bool{false, true} {
		flag := ""
		expected := "2020-01-01"
		serial := "43831"
		if epoch {
			flag = `<workbookPr date1904="1"/>`
			serial = "42369"
		}
		b := zipped(t, map[string]string{"xl/workbook.xml": `<workbook xmlns:r="urn:r">` + flag + `<sheets><sheet name="x" r:id="r"/></sheets></workbook>`, "xl/_rels/workbook.xml.rels": `<Relationships><Relationship Id="r" Target="worksheets/s.xml"/></Relationships>`, "xl/styles.xml": `<styleSheet><cellXfs><xf numFmtId="14"/></cellXfs></styleSheet>`, "xl/worksheets/s.xml": `<worksheet><sheetData><row r="1"><c r="A1" t="str"><f>EXTERNAL()</f><v>cached</v></c><c r="B1" s="0"><v>` + serial + `</v></c></row></sheetData></worksheet>`})
		book, e := workbook.Read(b)
		if e != nil {
			t.Fatal(e)
		}
		r := book.Sheets[0].Rows[0]
		if !r.Cells["A"].Formula || r.Cells["A"].Value != "cached" || r.Cells["B"].Value != expected {
			t.Fatal(epoch, r)
		}
	}
}
