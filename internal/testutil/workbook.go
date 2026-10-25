package testutil

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"testing"
)

// XLSX generates anonymous, runtime-only fixtures; never load personal data in CI.
func XLSX(t testing.TB, rows [][]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	if _, err := z.Create("xl/"); err != nil {
		t.Fatal(err)
	}
	parts := map[string]string{"xl/workbook.xml": `<workbook xmlns:r="urn:r"><sheets><sheet name="匿名样本" r:id="rId1"/></sheets></workbook>`, "xl/_rels/workbook.xml.rels": `<Relationships><Relationship Id="rId1" Target="worksheets/sheet1.xml"/></Relationships>`}
	var sheet bytes.Buffer
	sheet.WriteString(`<worksheet><sheetData>`)
	for i, row := range rows {
		fmt.Fprintf(&sheet, `<row r="%d">`, i+1)
		for j, v := range row {
			fmt.Fprintf(&sheet, `<c r="%c%d" t="inlineStr"><is><t>`, 'A'+j, i+1)
			_ = xml.EscapeText(&sheet, []byte(v))
			sheet.WriteString(`</t></is></c>`)
		}
		sheet.WriteString(`</row>`)
	}
	sheet.WriteString(`</sheetData></worksheet>`)
	parts["xl/worksheets/sheet1.xml"] = sheet.String()
	for name, v := range parts {
		f, e := z.Create(name)
		if e != nil {
			t.Fatal(e)
		}
		_, _ = f.Write([]byte(v))
	}
	if e := z.Close(); e != nil {
		t.Fatal(e)
	}
	return buf.Bytes()
}
