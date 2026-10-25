// Package workbook reads a deliberately bounded subset of XLSX. It never evaluates
// formulas, opens hyperlinks/external resources, writes files, or runs macros.
package workbook

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const MaxFile = 8 << 20
const MaxRows = 10000
const MaxColumns = 64

type Cell struct {
	Value   string `json:"value"`
	Formula bool   `json:"formula"`
	Issue   string `json:"issue,omitempty"`
}
type Row struct {
	Number int             `json:"number"`
	Cells  map[string]Cell `json:"cells"`
}
type Sheet struct {
	Name string `json:"name"`
	Rows []Row  `json:"rows"`
}
type Book struct {
	Sheets   []Sheet `json:"sheets"`
	Date1904 bool    `json:"date_1904"`
}
type rich struct {
	Text string `xml:"t"`
	Runs []struct {
		Text string `xml:"t"`
	} `xml:"r"`
}

func (r rich) text() string {
	s := r.Text
	for _, x := range r.Runs {
		s += x.Text
	}
	return s
}
func Read(data []byte) (Book, error) {
	fail := func() (Book, error) {
		return Book{}, errors.New("不支持或超限的XLSX：仅接受标准无宏文件，最多32表/10000行/64列/8MiB，解压总量32MiB")
	}
	if len(data) > MaxFile {
		return fail()
	}
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fail()
	}
	if len(z.File) > 512 {
		return fail()
	}
	files := map[string]*zip.File{}
	var total uint64
	for _, f := range z.File {
		if f.FileInfo().IsDir() && path.Clean(strings.TrimSuffix(f.Name, "/")) == strings.TrimSuffix(f.Name, "/") && !strings.HasPrefix(f.Name, "/") && !strings.HasPrefix(f.Name, "../") {
			continue
		}
		total += f.UncompressedSize64
		if total > 32<<20 || f.UncompressedSize64 > 16<<20 {
			return fail()
		}
		if files[f.Name] != nil || path.Clean(f.Name) != f.Name || strings.Contains(f.Name, "\\") || strings.HasPrefix(f.Name, "/") || strings.HasPrefix(f.Name, "../") || strings.Contains(strings.ToLower(f.Name), "vbaproject") {
			return fail()
		}
		files[f.Name] = f
	}
	read := func(name string, target any) error {
		f := files[name]
		if f == nil {
			return errors.New("missing part")
		}
		r, e := f.Open()
		if e != nil {
			return e
		}
		defer r.Close()
		b, e := io.ReadAll(io.LimitReader(r, 16<<20+1))
		if e != nil || len(b) > 16<<20 {
			return errors.New("part limit")
		}
		return xml.Unmarshal(b, target)
	}
	var wb struct {
		Props struct {
			Date1904 bool `xml:"date1904,attr"`
		} `xml:"workbookPr"`
		Sheets []struct {
			Name string `xml:"name,attr"`
			ID   string `xml:"id,attr"`
		} `xml:"sheets>sheet"`
	}
	var rel struct {
		Items []struct {
			ID     string `xml:"Id,attr"`
			Target string `xml:"Target,attr"`
			Mode   string `xml:"TargetMode,attr"`
		} `xml:"Relationship"`
	}
	if read("xl/workbook.xml", &wb) != nil || read("xl/_rels/workbook.xml.rels", &rel) != nil || len(wb.Sheets) > 32 {
		return fail()
	}
	targets := map[string]string{}
	for _, r := range rel.Items {
		if r.Mode != "External" {
			targets[r.ID] = r.Target
		}
	}
	var shared struct {
		Items []rich `xml:"si"`
	}
	if files["xl/sharedStrings.xml"] != nil && read("xl/sharedStrings.xml", &shared) != nil {
		return fail()
	}
	var styles struct {
		Formats []struct {
			ID   int    `xml:"numFmtId,attr"`
			Code string `xml:"formatCode,attr"`
		} `xml:"numFmts>numFmt"`
		XFs []struct {
			ID int `xml:"numFmtId,attr"`
		} `xml:"cellXfs>xf"`
	}
	if files["xl/styles.xml"] != nil && read("xl/styles.xml", &styles) != nil {
		return fail()
	}
	custom := map[int]string{}
	for _, f := range styles.Formats {
		custom[f.ID] = f.Code
	}
	book := Book{Date1904: wb.Props.Date1904, Sheets: []Sheet{}}
	count := 0
	coord := regexp.MustCompile(`^([A-Z]{1,2})([1-9][0-9]*)$`)
	for _, s := range wb.Sheets {
		target := targets[s.ID]
		if strings.HasPrefix(target, "/") {
			target = strings.TrimPrefix(target, "/")
		} else {
			target = path.Join("xl", target)
		}
		if !strings.HasPrefix(target, "xl/worksheets/") {
			return fail()
		}
		var doc struct {
			Rows []struct {
				Number int `xml:"r,attr"`
				Cells  []struct {
					Ref     string  `xml:"r,attr"`
					Type    string  `xml:"t,attr"`
					Style   int     `xml:"s,attr"`
					Value   string  `xml:"v"`
					Formula *string `xml:"f"`
					Inline  rich    `xml:"is"`
				} `xml:"c"`
			} `xml:"sheetData>row"`
		}
		if read(target, &doc) != nil {
			return fail()
		}
		sheet := Sheet{Name: s.Name, Rows: []Row{}}
		seen := map[int]bool{}
		for _, r := range doc.Rows {
			count++
			if count > MaxRows || r.Number < 1 || r.Number > 100000 || seen[r.Number] {
				return fail()
			}
			seen[r.Number] = true
			row := Row{Number: r.Number, Cells: map[string]Cell{}}
			for _, c := range r.Cells {
				m := coord.FindStringSubmatch(c.Ref)
				if m == nil {
					return fail()
				}
				n, _ := strconv.Atoi(m[2])
				col := 0
				for _, x := range m[1] {
					col = col*26 + int(x-'A'+1)
				}
				if n != r.Number || col > MaxColumns {
					return fail()
				}
				if _, exists := row.Cells[m[1]]; exists {
					return fail()
				}
				v := c.Value
				issue := ""
				switch c.Type {
				case "s":
					i, e := strconv.Atoi(v)
					if e != nil || i < 0 || i >= len(shared.Items) {
						return fail()
					}
					v = shared.Items[i].text()
				case "inlineStr":
					v = c.Inline.text()
				case "e":
					issue = "Excel单元格错误"
				case "", "n":
					if v != "" && c.Style >= 0 && c.Style < len(styles.XFs) {
						f := styles.XFs[c.Style].ID
						code := strings.ToLower(custom[f])
						code = regexp.MustCompile(`"[^"]*"|\[[^]]*\]|\\.`).ReplaceAllString(code, "")
						if ((f >= 14 && f <= 17) || f == 22) || (strings.Contains(code, "y") && strings.Contains(code, "d")) {
							v, issue = excelDate(v, book.Date1904)
						}
					}
				case "str", "b":
				default:
					issue = "不支持的单元格类型"
				}
				if len(v) > 10000 {
					return fail()
				}
				row.Cells[m[1]] = Cell{Value: v, Formula: c.Formula != nil, Issue: issue}
			}
			sheet.Rows = append(sheet.Rows, row)
		}
		book.Sheets = append(book.Sheets, sheet)
	}
	return book, nil
}
func excelDate(v string, epoch1904 bool) (string, string) {
	n, e := strconv.ParseFloat(v, 64)
	if e != nil || n < 0 || n > 2950000 || n != float64(int(n)) {
		return v, "日期含时间或无效序号，须人工改为ISO日期"
	}
	days := int(n)
	base := time.Date(1899, 12, 31, 0, 0, 0, 0, time.UTC)
	if epoch1904 {
		base = time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)
	} else {
		if days == 60 {
			return v, "Excel虚构的1900-02-29"
		}
		if days > 60 {
			days--
		}
	}
	d := base.AddDate(0, 0, days)
	return fmt.Sprintf("%04d-%02d-%02d", d.Year(), d.Month(), d.Day()), ""
}
