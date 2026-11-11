package chartfill

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	typeAttrRe  = regexp.MustCompile(`\s+t="[^"]*"`)
	spansAttrRe = regexp.MustCompile(`\s+spans="[^"]*"`)
	styleAttrRe = regexp.MustCompile(`\ss="(\d+)"`)
	cellTagRe   = regexp.MustCompile(`<c r="([A-Z]+)\d+"([^>]*?)(/?)>`)
	dimensionRe = regexp.MustCompile(`<dimension ref="([A-Z]+)(\d+)(?::([A-Z]+)(\d+))?"\s*/>`)
	rowOpenRe   = regexp.MustCompile(`<row r="(\d+)"`)
)

// setNumber writes an integer into a cell, creating the cell (and row) when the
// template does not have it yet. Existing styling is kept.
func (w *Workbook) setNumber(col, row, value int) error {
	ref := cellRef(col, row)
	num := strconv.Itoa(value)

	selfClosing := regexp.MustCompile(`<c r="` + ref + `"([^>]*?)/>`)
	if loc := selfClosing.FindStringSubmatchIndex(w.sheetXML); loc != nil {
		attrs := typeAttrRe.ReplaceAllString(w.sheetXML[loc[2]:loc[3]], "")
		w.splice(loc[0], loc[1], `<c r="`+ref+`"`+attrs+`><v>`+num+`</v></c>`)
		w.cells[ref] = num
		return nil
	}
	open := regexp.MustCompile(`(?s)<c r="` + ref + `"([^>]*)>(.*?)</c>`)
	if loc := open.FindStringSubmatchIndex(w.sheetXML); loc != nil {
		attrs := typeAttrRe.ReplaceAllString(w.sheetXML[loc[2]:loc[3]], "")
		w.splice(loc[0], loc[1], `<c r="`+ref+`"`+attrs+`><v>`+num+`</v></c>`)
		w.cells[ref] = num
		return nil
	}
	if err := w.insertCell(col, row, num); err != nil {
		return err
	}
	w.cells[ref] = num
	if row > w.maxRow {
		w.maxRow = row
	}
	w.growDimension(col, row)
	return nil
}

func (w *Workbook) splice(start, end int, repl string) {
	w.sheetXML = w.sheetXML[:start] + repl + w.sheetXML[end:]
}

// insertCell adds a brand-new cell in column order, reusing the style of the
// nearest cell on its left so the new value looks like its neighbours.
func (w *Workbook) insertCell(col, row int, num string) error {
	ref := cellRef(col, row)
	rowRe := regexp.MustCompile(`(?s)<row r="` + strconv.Itoa(row) + `"([^>]*?)(/>|>(.*?)</row>)`)
	loc := rowRe.FindStringSubmatchIndex(w.sheetXML)

	if loc == nil {
		newRow := `<row r="` + strconv.Itoa(row) + `"><c r="` + ref + `"><v>` + num + `</v></c></row>`
		pos := strings.Index(w.sheetXML, "</sheetData>")
		for _, m := range rowOpenRe.FindAllStringSubmatchIndex(w.sheetXML, -1) {
			if n, _ := strconv.Atoi(w.sheetXML[m[2]:m[3]]); n > row {
				pos = m[0]
				break
			}
		}
		if pos < 0 {
			return fmt.Errorf("worksheet has no <sheetData> section")
		}
		w.splice(pos, pos, newRow)
		return nil
	}

	attrs := spansAttrRe.ReplaceAllString(w.sheetXML[loc[2]:loc[3]], "")
	inner := ""
	if loc[6] >= 0 {
		inner = w.sheetXML[loc[6]:loc[7]]
	}

	insertAt, style := len(inner), ""
	for _, m := range cellTagRe.FindAllStringSubmatchIndex(inner, -1) {
		c := colIndex(inner[m[2]:m[3]])
		if c > col {
			insertAt = m[0]
			break
		}
		if sm := styleAttrRe.FindStringSubmatch(inner[m[4]:m[5]]); sm != nil {
			style = ` s="` + sm[1] + `"`
		}
	}
	newCell := `<c r="` + ref + `"` + style + `><v>` + num + `</v></c>`
	inner = inner[:insertAt] + newCell + inner[insertAt:]
	w.splice(loc[0], loc[1], `<row r="`+strconv.Itoa(row)+`"`+attrs+`>`+inner+`</row>`)
	return nil
}

func (w *Workbook) growDimension(col, row int) {
	loc := dimensionRe.FindStringSubmatchIndex(w.sheetXML)
	if loc == nil {
		return
	}
	m := dimensionRe.FindStringSubmatch(w.sheetXML)
	endCol, endRow := colIndex(m[1]), mustAtoi(m[2])
	if m[3] != "" {
		endCol, endRow = colIndex(m[3]), mustAtoi(m[4])
	}
	if col <= endCol && row <= endRow {
		return
	}
	endCol, endRow = max(endCol, col), max(endRow, row)
	w.splice(loc[0], loc[1], `<dimension ref="`+m[1]+m[2]+`:`+cellRef(endCol, endRow)+`"/>`)
}

func mustAtoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
