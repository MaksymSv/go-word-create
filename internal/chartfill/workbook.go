// Package chartfill fills the empty sprint columns of the "AI Assisted Work Items Chart"
// workbook with label-report percentages. It edits the xlsx at the zip level so the chart,
// styles and every other part of the file are preserved untouched.
package chartfill

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"regexp"
	"strconv"
	"strings"
)

const (
	headerRowSprint = 1
	headerRowTeam   = 2
	labelCol        = 2 // column B: "jira label"
	firstDataCol    = 3 // column C
)

// Workbook is an opened xlsx with a parsed view of its first worksheet.
type Workbook struct {
	zipData   []byte
	sheetPath string
	sheetName string
	sheetXML  string
	charts    map[string]string // chart part name -> current XML
	chartsNew map[string]string // chart parts changed since Open
	cells     map[string]string // ref -> text value ("" when the cell is empty)
	maxRow    int
	merges    []mergeRange
}

type mergeRange struct{ c1, r1, c2, r2 int }

type sheetDoc struct {
	Rows []struct {
		Cells []struct {
			R  string  `xml:"r,attr"`
			T  string  `xml:"t,attr"`
			V  *string `xml:"v"`
			IS *struct {
				T string `xml:"t"`
			} `xml:"is"`
		} `xml:"c"`
	} `xml:"sheetData>row"`
	Merges []struct {
		Ref string `xml:"ref,attr"`
	} `xml:"mergeCells>mergeCell"`
}

// Open parses an xlsx file held in memory.
func Open(data []byte) (*Workbook, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("not a valid xlsx file: %w", err)
	}
	files := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		files[f.Name] = f
	}

	w := &Workbook{zipData: data, cells: map[string]string{}, charts: map[string]string{}, chartsNew: map[string]string{}}
	w.sheetPath, w.sheetName = findFirstSheet(files)
	for name := range files {
		if chartPartRe.MatchString(name) {
			b, err := readEntry(files, name)
			if err != nil {
				return nil, err
			}
			w.charts[name] = string(b)
		}
	}
	sheetBytes, err := readEntry(files, w.sheetPath)
	if err != nil {
		return nil, err
	}
	w.sheetXML = string(sheetBytes)

	shared := readSharedStrings(files)

	var doc sheetDoc
	if err := xml.Unmarshal(sheetBytes, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", w.sheetPath, err)
	}
	for _, row := range doc.Rows {
		for _, c := range row.Cells {
			_, rowNum, ok := splitRef(c.R)
			if !ok {
				continue
			}
			if rowNum > w.maxRow {
				w.maxRow = rowNum
			}
			val := ""
			switch {
			case c.T == "s" && c.V != nil:
				if i, err := strconv.Atoi(strings.TrimSpace(*c.V)); err == nil && i >= 0 && i < len(shared) {
					val = shared[i]
				}
			case c.T == "inlineStr" && c.IS != nil:
				val = c.IS.T
			case c.V != nil:
				val = *c.V
			}
			w.cells[c.R] = strings.TrimSpace(val)
		}
	}
	for _, m := range doc.Merges {
		if mr, ok := parseMerge(m.Ref); ok {
			w.merges = append(w.merges, mr)
		}
	}
	return w, nil
}

// Bytes returns the workbook as an xlsx file. Every zip entry other than the
// edited worksheet and the charts changed by SyncCharts is copied as-is.
func (w *Workbook) Bytes() ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(w.zipData), int64(len(w.zipData)))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range zr.File {
		content, replaced := w.chartsNew[f.Name]
		if f.Name == w.sheetPath {
			content, replaced = w.sheetXML, true
		}
		if !replaced {
			if err := zw.Copy(f); err != nil {
				return nil, err
			}
			continue
		}
		hdr := f.FileHeader
		hdr.Method = zip.Deflate
		out, err := zw.CreateHeader(&hdr)
		if err != nil {
			return nil, err
		}
		if _, err := io.WriteString(out, content); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// text returns the value of a cell, resolving merged ranges to their top-left cell.
func (w *Workbook) text(col, row int) string {
	if v := w.cells[cellRef(col, row)]; v != "" {
		return v
	}
	for _, m := range w.merges {
		if col >= m.c1 && col <= m.c2 && row >= m.r1 && row <= m.r2 {
			return w.cells[cellRef(m.c1, m.r1)]
		}
	}
	return ""
}

func (w *Workbook) isEmpty(col, row int) bool {
	return w.cells[cellRef(col, row)] == ""
}

// maxHeaderCol is the right-most column used by the two header rows (including merges).
func (w *Workbook) maxHeaderCol() int {
	maxCol := 0
	for ref := range w.cells {
		col, row, ok := splitRef(ref)
		if ok && (row == headerRowSprint || row == headerRowTeam) && col > maxCol {
			maxCol = col
		}
	}
	for _, m := range w.merges {
		if m.r1 <= headerRowTeam && m.c2 > maxCol {
			maxCol = m.c2
		}
	}
	return maxCol
}

func findFirstSheet(files map[string]*zip.File) (sheetPath, sheetName string) {
	const fallback, fallbackName = "xl/worksheets/sheet1.xml", "Sheet1"
	wbData, err := readEntry(files, "xl/workbook.xml")
	if err != nil {
		return fallback, fallbackName
	}
	var wb struct {
		Sheets []struct {
			Name string `xml:"name,attr"`
			RID  string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
		} `xml:"sheets>sheet"`
	}
	if xml.Unmarshal(wbData, &wb) != nil || len(wb.Sheets) == 0 {
		return fallback, fallbackName
	}
	name := wb.Sheets[0].Name
	relData, err := readEntry(files, "xl/_rels/workbook.xml.rels")
	if err != nil {
		return fallback, name
	}
	var rels struct {
		Rels []struct {
			ID     string `xml:"Id,attr"`
			Target string `xml:"Target,attr"`
		} `xml:"Relationship"`
	}
	if xml.Unmarshal(relData, &rels) != nil {
		return fallback, name
	}
	for _, r := range rels.Rels {
		if r.ID == wb.Sheets[0].RID {
			if strings.HasPrefix(r.Target, "/") {
				return strings.TrimPrefix(r.Target, "/"), name
			}
			return path.Join("xl", r.Target), name
		}
	}
	return fallback, name
}

func readSharedStrings(files map[string]*zip.File) []string {
	data, err := readEntry(files, "xl/sharedStrings.xml")
	if err != nil {
		return nil
	}
	var sst struct {
		SI []struct {
			T string `xml:"t"`
			R []struct {
				T string `xml:"t"`
			} `xml:"r"`
		} `xml:"si"`
	}
	if xml.Unmarshal(data, &sst) != nil {
		return nil
	}
	out := make([]string, len(sst.SI))
	for i, si := range sst.SI {
		s := si.T
		for _, r := range si.R {
			s += r.T
		}
		out[i] = s
	}
	return out
}

func readEntry(files map[string]*zip.File, name string) ([]byte, error) {
	f, ok := files[name]
	if !ok {
		return nil, fmt.Errorf("xlsx entry %q not found", name)
	}
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	return io.ReadAll(rc)
}

var refRe = regexp.MustCompile(`^([A-Z]+)(\d+)$`)

func splitRef(ref string) (col, row int, ok bool) {
	m := refRe.FindStringSubmatch(ref)
	if m == nil {
		return 0, 0, false
	}
	row, _ = strconv.Atoi(m[2])
	return colIndex(m[1]), row, true
}

func colIndex(name string) int {
	n := 0
	for _, r := range name {
		n = n*26 + int(r-'A'+1)
	}
	return n
}

func colName(n int) string {
	s := ""
	for n > 0 {
		n--
		s = string(rune('A'+n%26)) + s
		n /= 26
	}
	return s
}

func cellRef(col, row int) string { return colName(col) + strconv.Itoa(row) }

func parseMerge(ref string) (mergeRange, bool) {
	parts := strings.SplitN(ref, ":", 2)
	if len(parts) != 2 {
		return mergeRange{}, false
	}
	c1, r1, ok1 := splitRef(parts[0])
	c2, r2, ok2 := splitRef(parts[1])
	if !ok1 || !ok2 {
		return mergeRange{}, false
	}
	return mergeRange{c1, r1, c2, r2}, true
}
