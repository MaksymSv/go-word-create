package mcpserver

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go-word-create/internal/chartfill"
	"go-word-create/internal/jiraservice"
)

func inlineCell(ref, text string) string {
	return `<c r="` + ref + `" t="inlineStr"><is><t>` + text + `</t></is></c>`
}

func chartFixture(t *testing.T) []byte {
	t.Helper()
	sheet := `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><dimension ref="A1:D5"/><sheetData>` +
		`<row r="1">` + inlineCell("A1", "Sprint") + inlineCell("B1", "jira label") + inlineCell("C1", "S26.20") + `<c r="D1"/></row>` +
		`<row r="2">` + inlineCell("A2", "Team") + inlineCell("C2", "ST") + inlineCell("D2", "PC") + `</row>` +
		`<row r="4">` + inlineCell("A4", "DEV Assisted WI count, %") + inlineCell("B4", "ai-assisted-dev") + `<c r="C4" s="2"/><c r="D4" s="2"/></row>` +
		`<row r="5">` + inlineCell("A5", "total assisted SP, %") + `<c r="C5" s="6"/><c r="D5" s="6"/></row>` +
		`</sheetData><mergeCells count="1"><mergeCell ref="C1:D1"/></mergeCells></worksheet>`
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"xl/workbook.xml":          `<workbook/>`,
		"xl/worksheets/sheet1.xml": sheet,
		"xl/charts/chart1.xml":     `<chart/>`,
	} {
		w, _ := zw.Create(name)
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func chartFake() *fakeJira {
	return &fakeJira{namedIssues: map[string][]jiraservice.Issue{
		"ST26.20": {
			{Key: "PRJ-1", StoryPoints: 3, Components: []string{"STABLETEK"}, Labels: []string{"ai-assisted-dev"}},
			{Key: "PRJ-2", StoryPoints: 1, Components: []string{"STABLETEK"}},
		},
		// PC26.20 is unknown to the fake -> skipped
	}}
}

func setupWorkdir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.MkdirAll("proj-docs", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chartfill.DefaultFile, chartFixture(t), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestFillSprintChartPreviewDoesNotWrite(t *testing.T) {
	setupWorkdir(t)
	before, _ := os.ReadFile(chartfill.DefaultFile)

	_, out, err := newHandler(chartFake()).fillSprintChart(context.Background(), nil, fillSprintChartIn{})
	if err != nil {
		t.Fatal(err)
	}
	if out.Written || out.XLSX == "" || len(out.Filled) != 1 || len(out.Skipped) != 1 {
		t.Fatalf("out = %+v", out)
	}
	if out.Filled[0].Sprint != "ST26.20" || out.Filled[0].Component != "STABLETEK" {
		t.Errorf("filled = %+v", out.Filled[0])
	}
	if got := out.Filled[0].Cells; got[0].Value != 50 || got[1].Value != 75 {
		t.Errorf("cells = %+v (want 50 count %%, 75 SP %%)", got)
	}
	if !strings.Contains(out.Skipped[0].Reason, "sprint not found") || out.Skipped[0].Sprint != "PC26.20" {
		t.Errorf("skipped = %+v", out.Skipped)
	}
	after, _ := os.ReadFile(chartfill.DefaultFile)
	if !bytes.Equal(before, after) {
		t.Error("preview must not modify the file")
	}
	raw, err := base64.StdEncoding.DecodeString(out.XLSX)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw))); err != nil {
		t.Errorf("returned xlsx is not a zip: %v", err)
	}
}

func TestFillSprintChartWriteUpdatesFile(t *testing.T) {
	setupWorkdir(t)
	_, out, err := newHandler(chartFake()).fillSprintChart(context.Background(), nil, fillSprintChartIn{Write: true})
	if err != nil {
		t.Fatal(err)
	}
	if !out.Written || out.XLSX != "" {
		t.Fatalf("out = %+v", out)
	}
	data, _ := os.ReadFile(chartfill.DefaultFile)
	wb, err := chartfill.Open(data)
	if err != nil {
		t.Fatal(err)
	}
	pending := wb.Pending(chartfill.DefaultPrefixes)
	if len(pending) != 1 || pending[0].Sprint != "PC26.20" {
		t.Errorf("pending after write = %+v", pending)
	}
	if entries, _ := filepath.Glob("proj-docs/.chartfill-*"); len(entries) != 0 {
		t.Errorf("temp files left behind: %v", entries)
	}
}

func TestFillSprintChartRejectsBadPaths(t *testing.T) {
	setupWorkdir(t)
	for _, file := range []string{"../escape.xlsx", "/etc/passwd.xlsx", "proj-docs/chart.docx", "proj-docs/../../x.xlsx"} {
		if _, _, err := newHandler(chartFake()).fillSprintChart(context.Background(), nil, fillSprintChartIn{File: file, Write: true}); err == nil {
			t.Errorf("file %q should be rejected", file)
		}
	}
}
