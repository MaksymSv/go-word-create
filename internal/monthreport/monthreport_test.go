package monthreport

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"go-word-create/internal/jiraservice"
)

func TestSplit(t *testing.T) {
	ti := Split("Alpha", []jiraservice.Issue{
		{Key: "PRJ-1", Status: "Closed"},
		{Key: "PRJ-2", Status: "In Progress"},
		{Key: "PRJ-3", Status: "closed"},
	})
	if ti.Component != "Alpha" || len(ti.Closed) != 1 || len(ti.Open) != 2 {
		t.Errorf("Split = %+v", ti)
	}
}

func TestFileName(t *testing.T) {
	m := time.Date(2025, time.October, 1, 0, 0, 0, 0, time.UTC)
	for in, want := range map[string]string{
		"report.docx": "report - 2025-10.docx",
		"report":      "report - 2025-10.docx",
	} {
		if got := FileName(in, m); got != want {
			t.Errorf("FileName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBytesIsValidDocxWithHeadingsAndIssues(t *testing.T) {
	m := time.Date(2025, time.October, 1, 0, 0, 0, 0, time.UTC)
	data, err := Bytes(m, []TeamIssues{
		{Component: "Alpha", Closed: []jiraservice.Issue{{Type: "Story", Key: "PRJ-1", Summary: "Done thing", StoryPoints: 3}}},
		{Component: "Beta", Open: []jiraservice.Issue{{Type: "Bug", Key: "PRJ-2", Summary: "Open thing", StoryPoints: 1}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("not a zip/docx: %v", err)
	}
	var xml string
	for _, f := range zr.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(rc)
		_ = rc.Close()
		xml = string(b)
	}
	for _, want := range []string{
		"Closed Issues During October 2025 (Alpha)",
		"Issues were in work but not Closed during October 2025 (Beta)",
		"PRJ-1", "Done thing", "PRJ-2", "Open thing",
	} {
		if !strings.Contains(xml, want) {
			t.Errorf("document.xml missing %q", want)
		}
	}
}
