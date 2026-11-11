package chartfill

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"go-word-create/internal/labelreport"
)

// Shared strings used by the fixture.
const fixtureSST = `<?xml version="1.0" encoding="UTF-8"?><sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="14" uniqueCount="14">` +
	`<si><t>Sprint</t></si><si><t>jira label</t></si><si><t>Team</t></si>` + // 0 1 2
	`<si><t>S26.19</t></si><si><t>S26.20</t></si><si><t>S26.21</t></si>` + // 3 4 5
	`<si><t>ST</t></si><si><t>PC</t></si>` + // 6 7
	`<si><t>BA Assisted WI count, %</t></si><si><t>total assisted WI count, %</t></si>` + // 8 9
	`<si><t>BA Assisted SP, %</t></si><si><t>total assisted SP, %</t></si>` + // 10 11
	`<si><t>ai-assisted-ba</t></si><si><t>ZZ</t></si></sst>` // 12 13

// Columns: C,D = S26.19 (filled), E,F = S26.20 (styled empty cells), G,H = S26.21 (cells missing entirely).
const fixtureSheet = `<?xml version="1.0" encoding="UTF-8"?>` +
	`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><dimension ref="A1:F12"/><sheetData>` +
	`<row r="1" spans="1:8"><c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>1</v></c><c r="C1" t="s"><v>3</v></c><c r="D1"/><c r="E1" t="s"><v>4</v></c><c r="F1"/><c r="G1" t="s"><v>5</v></c><c r="H1"/></row>` +
	`<row r="2"><c r="A2" t="s"><v>2</v></c><c r="B2"/><c r="C2" t="s"><v>6</v></c><c r="D2" t="s"><v>7</v></c><c r="E2" t="s"><v>6</v></c><c r="F2" t="s"><v>7</v></c><c r="G2" t="s"><v>6</v></c><c r="H2" t="s"><v>7</v></c></row>` +
	`<row r="3"/>` +
	`<row r="4" spans="1:8"><c r="A4" t="s"><v>8</v></c><c r="B4" t="s"><v>12</v></c><c r="C4" s="1"><v>10</v></c><c r="D4" s="1"><v>20</v></c><c r="E4" s="2"/><c r="F4" s="2"/></row>` +
	`<row r="5"><c r="A5" t="s"><v>9</v></c><c r="B5"/><c r="C5" s="5"><v>30</v></c><c r="D5" s="5"><v>40</v></c><c r="E5" s="6"/><c r="F5" s="6"/></row>` +
	`<row r="9"><c r="A9" t="s"><v>10</v></c><c r="B9" t="s"><v>12</v></c><c r="C9" s="1"><v>50</v></c><c r="D9" s="1"><v>60</v></c><c r="E9" s="2"/><c r="F9" s="2"/></row>` +
	`<row r="12"><c r="A12" t="s"><v>11</v></c><c r="B12"/><c r="C12" s="5"><v>70</v></c><c r="D12" s="5"><v>80</v></c><c r="E12" s="6"/><c r="F12" s="6"/></row>` +
	`</sheetData><mergeCells count="3"><mergeCell ref="C1:D1"/><mergeCell ref="E1:F1"/><mergeCell ref="G1:H1"/></mergeCells></worksheet>`

func buildXLSX(t *testing.T, sheet string) []byte {
	t.Helper()
	return buildXLSXWithChart(t, sheet, `<chart>keep me</chart>`)
}

func buildXLSXWithChart(t *testing.T, sheet, chart string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	entries := []struct{ name, body string }{
		{"[Content_Types].xml", `<Types/>`},
		{"xl/workbook.xml", `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Graph" sheetId="1" r:id="rId9"/></sheets></workbook>`},
		{"xl/_rels/workbook.xml.rels", `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId9" Target="worksheets/sheet1.xml"/></Relationships>`},
		{"xl/sharedStrings.xml", fixtureSST},
		{"xl/worksheets/sheet1.xml", sheet},
		{"xl/charts/chart1.xml", chart},
	}
	for _, e := range entries {
		w, err := zw.Create(e.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(w, e.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func entry(t *testing.T, data []byte, name string) string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range zr.File {
		if f.Name == name {
			rc, _ := f.Open()
			defer func() { _ = rc.Close() }()
			b, _ := io.ReadAll(rc)
			return string(b)
		}
	}
	t.Fatalf("entry %s not found", name)
	return ""
}

func fakeReport(ba, total float64) labelreport.ComponentReport {
	return labelreport.ComponentReport{
		LabelGroups: []labelreport.LabelGroup{
			{LabelName: "ai-assisted-ba", CountPct: ba, TotalSPPct: ba + 0.4},
		},
		TotalLabeledCountPct: total,
		TotalLabeledSPPct:    total + 0.4,
	}
}

func TestRowsAreMappedByJiraLabelColumn(t *testing.T) {
	w, err := Open(buildXLSX(t, fixtureSheet))
	if err != nil {
		t.Fatal(err)
	}
	rows := w.Rows()
	want := []struct {
		row   int
		label string
		total bool
		m     Metric
	}{
		{4, "ai-assisted-ba", false, MetricCount},
		{5, "", true, MetricCount},
		{9, "ai-assisted-ba", false, MetricSP},
		{12, "", true, MetricSP},
	}
	if len(rows) != len(want) {
		t.Fatalf("rows = %+v", rows)
	}
	for i, wr := range want {
		got := rows[i]
		if got.Row != wr.row || got.Label != wr.label || got.Total != wr.total || got.Metric != wr.m {
			t.Errorf("row %d = %+v, want %+v", i, got, wr)
		}
	}
}

func TestPendingBuildsSprintNamesFromMergedHeaderAndTeamRow(t *testing.T) {
	w, _ := Open(buildXLSX(t, fixtureSheet))
	cols := w.Pending(DefaultPrefixes)
	var got []string
	for _, c := range cols {
		got = append(got, fmt.Sprintf("%s=%s/%s/%d", c.Ref, c.Sprint, c.Component, len(c.EmptyRows)))
	}
	want := "E=ST26.20/STABLETEK/4 F=PC26.20/PROCESSING/4 G=ST26.21/STABLETEK/4 H=PC26.21/PROCESSING/4"
	if strings.Join(got, " ") != want {
		t.Errorf("pending = %v\nwant     %s", got, want)
	}
}

func TestUnknownPrefixIsSkipped(t *testing.T) {
	w, _ := Open(buildXLSX(t, fixtureSheet))
	res, err := w.Fill(func(string, string) (labelreport.ComponentReport, error) { return fakeReport(1, 1), nil },
		map[string]string{"ST": "STABLETEK"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Skipped) != 2 || !strings.Contains(res.Skipped[0].Reason, "unknown team prefix") {
		t.Errorf("skipped = %+v", res.Skipped)
	}
}

func TestFillWritesRoundedValuesAndPreservesEverythingElse(t *testing.T) {
	orig := buildXLSX(t, fixtureSheet)
	w, _ := Open(orig)
	var asked []string
	res, err := w.Fill(func(sprint, component string) (labelreport.ComponentReport, error) {
		asked = append(asked, sprint+"/"+component)
		if sprint == "PC26.21" {
			return labelreport.ComponentReport{}, fmt.Errorf("sprint not found")
		}
		return fakeReport(41.6, 87.5), nil
	}, DefaultPrefixes, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Filled) != 3 || len(res.Skipped) != 1 || res.Skipped[0].Column != "H" {
		t.Fatalf("result = %+v", res)
	}

	out, err := w.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	sheet := entry(t, out, "xl/worksheets/sheet1.xml")
	for _, want := range []string{
		`<c r="E4" s="2"><v>42</v></c>`,  // count % rounded
		`<c r="E5" s="6"><v>88</v></c>`,  // total count %
		`<c r="E9" s="2"><v>42</v></c>`,  // SP % (41.6+0.4 = 42)
		`<c r="E12" s="6"><v>88</v></c>`, // total SP % (87.9)
		`<c r="C4" s="1"><v>10</v></c>`,  // existing value untouched
		`<c r="G4" s="2"><v>42</v></c>`,  // inserted cell borrows left neighbour style
		`<c r="G12" s="6"><v>88</v></c>`,
		`<dimension ref="A1:G12"/>`,
	} {
		if !strings.Contains(sheet, want) {
			t.Errorf("sheet missing %s\n%s", want, sheet)
		}
	}
	if strings.Contains(sheet, `r="H4"`) {
		t.Error("skipped column H must not be written")
	}
	if got := entry(t, out, "xl/charts/chart1.xml"); got != `<chart>keep me</chart>` {
		t.Errorf("chart changed: %q", got)
	}

	// Result reopens cleanly and is idempotent: only the skipped column is still pending.
	w2, err := Open(out)
	if err != nil {
		t.Fatal(err)
	}
	pending := w2.Pending(DefaultPrefixes)
	if len(pending) != 1 || pending[0].Ref != "H" {
		t.Errorf("pending after fill = %+v", pending)
	}
}

func TestFillDryRunDoesNotChangeSheet(t *testing.T) {
	w, _ := Open(buildXLSX(t, fixtureSheet))
	before := w.sheetXML
	res, err := w.Fill(func(string, string) (labelreport.ComponentReport, error) { return fakeReport(5, 5), nil }, DefaultPrefixes, true)
	if err != nil || len(res.Filled) != 4 {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if w.sheetXML != before {
		t.Error("dry run modified the sheet")
	}
}

func TestUntrackedLabelAborts(t *testing.T) {
	w, _ := Open(buildXLSX(t, fixtureSheet))
	_, err := w.Fill(func(string, string) (labelreport.ComponentReport, error) {
		return labelreport.ComponentReport{}, nil
	}, DefaultPrefixes, true)
	if err == nil || !strings.Contains(err.Error(), "ai-assisted-ba") {
		t.Errorf("err = %v", err)
	}
}

func TestColumnNames(t *testing.T) {
	for n, name := range map[int]string{1: "A", 26: "Z", 27: "AA", 30: "AD", 52: "AZ", 53: "BA"} {
		if colName(n) != name || colIndex(name) != n {
			t.Errorf("%d <-> %s", n, name)
		}
	}
}

// Series in the shape Excel writes: categories C1:F2 and values C5:F5 end at F, the
// fixture has headers up to H. The second series points at another sheet.
const fixtureChart = `<c:chartSpace xmlns:c="http://schemas.openxmlformats.org/drawingml/2006/chart"><c:chart><c:plotArea><c:barChart>` +
	`<c:ser><c:tx><c:strRef><c:f>Graph!$A$5</c:f><c:strCache><c:ptCount val="1"/><c:pt idx="0"><c:v>stale</c:v></c:pt></c:strCache></c:strRef></c:tx>` +
	`<c:cat><c:multiLvlStrRef><c:f>Graph!$C$1:$F$2</c:f><c:multiLvlStrCache><c:ptCount val="4"/></c:multiLvlStrCache></c:multiLvlStrRef></c:cat>` +
	`<c:val><c:numRef><c:f>Graph!$C$5:$F$5</c:f><c:numCache><c:formatCode>0%</c:formatCode><c:ptCount val="4"/></c:numCache></c:numRef></c:val></c:ser>` +
	`<c:ser><c:val><c:numRef><c:f>Other!$C$5:$F$5</c:f><c:numCache><c:formatCode>General</c:formatCode><c:ptCount val="4"/></c:numCache></c:numRef></c:val></c:ser>` +
	`</c:barChart></c:plotArea></c:chart></c:chartSpace>`

func TestFillExtendsChartRangesAndRefreshesCaches(t *testing.T) {
	w, err := Open(buildXLSXWithChart(t, fixtureSheet, fixtureChart))
	if err != nil {
		t.Fatal(err)
	}
	res, err := w.Fill(func(sprint, _ string) (labelreport.ComponentReport, error) {
		if sprint == "PC26.21" {
			return labelreport.ComponentReport{}, fmt.Errorf("sprint not found")
		}
		return fakeReport(41.6, 87.5), nil
	}, DefaultPrefixes, false)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, c := range res.Charts {
		got[c.From] = c.To
	}
	if len(got) != 2 || got["Graph!$C$1:$F$2"] != "Graph!$C$1:$H$2" || got["Graph!$C$5:$F$5"] != "Graph!$C$5:$H$5" {
		t.Fatalf("chart changes = %+v", res.Charts)
	}
	if !res.Changed() {
		t.Error("Changed() should be true")
	}

	out, _ := w.Bytes()
	chart := entry(t, out, "xl/charts/chart1.xml")
	for _, want := range []string{
		`<c:f>Graph!$C$1:$H$2</c:f>`,
		`<c:f>Graph!$C$5:$H$5</c:f>`,
		`<c:f>Graph!$A$5</c:f>`,      // single cell stays
		`<c:f>Other!$C$5:$F$5</c:f>`, // other sheet untouched
		`<c:multiLvlStrCache><c:ptCount val="6"/><c:lvl><c:pt idx="0"><c:v>ST</c:v></c:pt><c:pt idx="1"><c:v>PC</c:v></c:pt>`,
		`</c:lvl><c:lvl><c:pt idx="0"><c:v>S26.19</c:v></c:pt><c:pt idx="2"><c:v>S26.20</c:v></c:pt><c:pt idx="4"><c:v>S26.21</c:v></c:pt></c:lvl>`,
		`<c:formatCode>0%</c:formatCode><c:ptCount val="6"/><c:pt idx="0"><c:v>30</c:v></c:pt><c:pt idx="1"><c:v>40</c:v></c:pt><c:pt idx="2"><c:v>88</c:v></c:pt>`,
		`<c:v>total assisted WI count, %</c:v>`, // series name cache refreshed (stale text replaced)
	} {
		if !strings.Contains(chart, want) {
			t.Errorf("chart missing %s\n%s", want, chart)
		}
	}
	if strings.Contains(chart, "stale") {
		t.Error("series name cache was not refreshed")
	}
	if strings.Contains(chart, `Other!$C$5:$H$5`) {
		t.Error("range on another sheet must not be extended")
	}

	// Idempotent: reopening finds the chart already covering the headers.
	w2, _ := Open(out)
	res2, _ := w2.Fill(func(string, string) (labelreport.ComponentReport, error) {
		return labelreport.ComponentReport{}, fmt.Errorf("sprint not found")
	}, DefaultPrefixes, false)
	if len(res2.Charts) != 0 || res2.Changed() {
		t.Errorf("second run: %+v", res2)
	}
}

func TestSyncChartsDryRunAndNeverShrinks(t *testing.T) {
	w, _ := Open(buildXLSXWithChart(t, fixtureSheet, fixtureChart))
	changes, err := w.SyncCharts(true)
	if err != nil || len(changes) != 2 {
		t.Fatalf("changes=%+v err=%v", changes, err)
	}
	if len(w.chartsNew) != 0 {
		t.Error("dry run must not modify charts")
	}

	wide := strings.ReplaceAll(fixtureChart, "$F$", "$Z$")
	w, _ = Open(buildXLSXWithChart(t, fixtureSheet, wide))
	if changes, _ := w.SyncCharts(false); len(changes) != 0 {
		t.Errorf("ranges wider than the headers must stay: %+v", changes)
	}
	if !strings.Contains(w.charts["xl/charts/chart1.xml"], `Graph!$C$1:$Z$2`) {
		t.Error("range was shrunk")
	}
}
