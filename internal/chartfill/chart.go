package chartfill

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var chartPartRe = regexp.MustCompile(`^xl/charts/chart[^/]*\.xml$`)

// ChartRange reports a chart series range that was widened to the last header column.
type ChartRange struct {
	Chart string `json:"chart"`
	From  string `json:"from"`
	To    string `json:"to"`
}

type cellRange struct{ c1, r1, c2, r2 int }

var (
	numRefRe      = regexp.MustCompile(`(?s)<c:numRef><c:f>([^<]*)</c:f>(?:<c:numCache>(.*?)</c:numCache>)?</c:numRef>`)
	strRefRe      = regexp.MustCompile(`(?s)<c:strRef><c:f>([^<]*)</c:f>(?:<c:strCache>(.*?)</c:strCache>)?</c:strRef>`)
	multiLvlRefRe = regexp.MustCompile(`(?s)<c:multiLvlStrRef><c:f>([^<]*)</c:f>(?:<c:multiLvlStrCache>(.*?)</c:multiLvlStrCache>)?</c:multiLvlStrRef>`)
	formatCodeRe  = regexp.MustCompile(`<c:formatCode>[^<]*</c:formatCode>`)
	formulaRe     = regexp.MustCompile(`^((?:'(?:[^']|'')+'|[^!']+)!)\$?([A-Z]+)\$?(\d+)(?::\$?([A-Z]+)\$?(\d+))?$`)
)

// SyncCharts widens every chart series range that starts at the first data column
// to the last header column of the sheet (so new sprint columns are plotted) and
// regenerates the cached values of all series that point at this sheet.
// Ranges are only ever widened. With dryRun the workbook is left unchanged.
func (w *Workbook) SyncCharts(dryRun bool) ([]ChartRange, error) {
	last := w.maxHeaderCol()
	names := make([]string, 0, len(w.charts))
	for name := range w.charts {
		names = append(names, name)
	}
	sort.Strings(names)

	var changes []ChartRange
	for _, name := range names {
		updated := w.charts[name]
		rewrite := func(re *regexp.Regexp, kind string) {
			updated = re.ReplaceAllStringFunc(updated, func(block string) string {
				m := re.FindStringSubmatch(block)
				rng, prefix, ok := w.parseFormula(m[1])
				if !ok {
					return block
				}
				if rng.c1 == firstDataCol && rng.c2 >= firstDataCol && rng.c2 < last {
					oldRef := m[1]
					rng.c2 = last
					changes = append(changes, ChartRange{Chart: name, From: oldRef, To: prefix + formatRange(rng)})
				}
				f := prefix + formatRange(rng)
				switch kind {
				case "num":
					return `<c:numRef><c:f>` + f + `</c:f>` + w.numCache(rng, m[2]) + `</c:numRef>`
				case "str":
					return `<c:strRef><c:f>` + f + `</c:f>` + w.strCache(rng) + `</c:strRef>`
				default:
					return `<c:multiLvlStrRef><c:f>` + f + `</c:f>` + w.multiLvlCache(rng) + `</c:multiLvlStrRef>`
				}
			})
		}
		rewrite(numRefRe, "num")
		rewrite(strRefRe, "str")
		rewrite(multiLvlRefRe, "multi")

		if !dryRun && updated != w.charts[name] {
			w.charts[name] = updated
			w.chartsNew[name] = updated
		}
	}
	return changes, nil
}

// parseFormula parses "Graph!$C$1:$AD$2" and returns the range plus the raw
// "Graph!" prefix. ok is false for ranges on other sheets.
func (w *Workbook) parseFormula(f string) (cellRange, string, bool) {
	m := formulaRe.FindStringSubmatch(f)
	if m == nil {
		return cellRange{}, "", false
	}
	sheet := strings.ReplaceAll(strings.TrimSuffix(m[1], "!"), "&apos;", "'")
	if strings.HasPrefix(sheet, "'") {
		sheet = strings.ReplaceAll(sheet[1:len(sheet)-1], "''", "'")
	}
	if sheet != w.sheetName {
		return cellRange{}, "", false
	}
	r := cellRange{c1: colIndex(m[2]), r1: mustAtoi(m[3])}
	r.c2, r.r2 = r.c1, r.r1
	if m[4] != "" {
		r.c2, r.r2 = colIndex(m[4]), mustAtoi(m[5])
	}
	return r, m[1], true
}

func formatRange(r cellRange) string {
	abs := func(c, row int) string { return "$" + colName(c) + "$" + strconv.Itoa(row) }
	if r.c1 == r.c2 && r.r1 == r.r2 {
		return abs(r.c1, r.r1)
	}
	return abs(r.c1, r.r1) + ":" + abs(r.c2, r.r2)
}

// linear lists the cells of a one-row or one-column range in order (row-major otherwise).
func (r cellRange) linear() [][2]int {
	var out [][2]int
	for row := r.r1; row <= r.r2; row++ {
		for col := r.c1; col <= r.c2; col++ {
			out = append(out, [2]int{col, row})
		}
	}
	return out
}

func (w *Workbook) numCache(r cellRange, old string) string {
	format := formatCodeRe.FindString(old)
	if format == "" {
		format = `<c:formatCode>General</c:formatCode>`
	}
	cells := r.linear()
	var b strings.Builder
	b.WriteString(`<c:numCache>` + format + `<c:ptCount val="` + strconv.Itoa(len(cells)) + `"/>`)
	for i, c := range cells {
		v := w.cells[cellRef(c[0], c[1])]
		if _, err := strconv.ParseFloat(v, 64); err == nil {
			fmt.Fprintf(&b, `<c:pt idx="%d"><c:v>%s</c:v></c:pt>`, i, v)
		}
	}
	b.WriteString(`</c:numCache>`)
	return b.String()
}

func (w *Workbook) strCache(r cellRange) string {
	cells := r.linear()
	var b strings.Builder
	b.WriteString(`<c:strCache><c:ptCount val="` + strconv.Itoa(len(cells)) + `"/>`)
	for i, c := range cells {
		if v := w.cells[cellRef(c[0], c[1])]; v != "" {
			fmt.Fprintf(&b, `<c:pt idx="%d"><c:v>%s</c:v></c:pt>`, i, escapeText(v))
		}
	}
	b.WriteString(`</c:strCache>`)
	return b.String()
}

// multiLvlCache builds the cache of a multi-level category range. Categories run
// along the columns (levels are rows, innermost level = last row) unless the range
// is a single column.
func (w *Workbook) multiLvlCache(r cellRange) string {
	alongCols := r.c1 != r.c2 || r.r1 == r.r2
	var n int
	var levels [][][2]int // each level: the cells that make up one level, in category order
	if alongCols {
		n = r.c2 - r.c1 + 1
		for row := r.r2; row >= r.r1; row-- {
			var lvl [][2]int
			for col := r.c1; col <= r.c2; col++ {
				lvl = append(lvl, [2]int{col, row})
			}
			levels = append(levels, lvl)
		}
	} else {
		n = r.r2 - r.r1 + 1
		for col := r.c2; col >= r.c1; col-- {
			var lvl [][2]int
			for row := r.r1; row <= r.r2; row++ {
				lvl = append(lvl, [2]int{col, row})
			}
			levels = append(levels, lvl)
		}
	}
	var b strings.Builder
	b.WriteString(`<c:multiLvlStrCache><c:ptCount val="` + strconv.Itoa(n) + `"/>`)
	for _, lvl := range levels {
		b.WriteString(`<c:lvl>`)
		for i, c := range lvl {
			if v := w.cells[cellRef(c[0], c[1])]; v != "" {
				fmt.Fprintf(&b, `<c:pt idx="%d"><c:v>%s</c:v></c:pt>`, i, escapeText(v))
			}
		}
		b.WriteString(`</c:lvl>`)
	}
	b.WriteString(`</c:multiLvlStrCache>`)
	return b.String()
}

func escapeText(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}
