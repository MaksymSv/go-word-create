package mcpserver

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"go-word-create/internal/chartfill"
)

// --- fill_sprint_chart ---

type fillSprintChartIn struct {
	File  string `json:"file,omitempty" jsonschema:"workbook path relative to the server working directory, must end in .xlsx; default proj-docs/AI Assisted Work Items Chart.xlsx"`
	Write bool   `json:"write,omitempty" jsonschema:"set true to save the filled workbook back to the file; without it nothing is written and the filled workbook is only returned as base64. Set it only after the user asked to update the file"`
}

type fillSprintChartOut struct {
	File     string                    `json:"file"`
	Written  bool                      `json:"written"`
	Filled   []chartfill.FilledColumn  `json:"filled"`
	Charts   []chartfill.ChartRange    `json:"chartRanges"`
	Skipped  []chartfill.SkippedColumn `json:"skipped"`
	Message  string                    `json:"message"`
	FileName string                    `json:"fileName,omitempty"`
	XLSX     string                    `json:"xlsx,omitempty" jsonschema:"base64-encoded filled .xlsx; present only when write is false and something was filled"`
}

func (h *handler) fillSprintChart(_ context.Context, _ *mcp.CallToolRequest, in fillSprintChartIn) (*mcp.CallToolResult, fillSprintChartOut, error) {
	rel := strings.TrimSpace(in.File)
	if rel == "" {
		rel = chartfill.DefaultFile
	}
	path, err := resolveWorkbookPath(rel)
	if err != nil {
		return nil, fillSprintChartOut{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fillSprintChartOut{}, fmt.Errorf("read %s: %w", rel, err)
	}
	wb, err := chartfill.Open(data)
	if err != nil {
		return nil, fillSprintChartOut{}, fmt.Errorf("%s: %w", rel, err)
	}

	res, err := wb.Fill(chartfill.NewJiraFetcher(h.jira, h.cfg), chartfill.DefaultPrefixes, false)
	if err != nil {
		return nil, fillSprintChartOut{}, err
	}
	out := fillSprintChartOut{File: rel, Filled: res.Filled, Charts: res.Charts, Skipped: res.Skipped}

	if !res.Changed() {
		out.Message = "Nothing to fill: no sprint column with headers has empty cells that could be calculated, and the chart already covers all columns."
		return nil, out, nil
	}
	filled, err := wb.Bytes()
	if err != nil {
		return nil, fillSprintChartOut{}, err
	}

	if in.Write {
		if err := chartfill.WriteFileAtomic(path, filled); err != nil {
			return nil, fillSprintChartOut{}, fmt.Errorf("write %s: %w", rel, err)
		}
		out.Written = true
		out.Message = fmt.Sprintf("Saved %d column(s) and %d chart range(s) into %s.", len(res.Filled), len(res.Charts), rel)
		return nil, out, nil
	}
	out.FileName = filepath.Base(rel)
	out.XLSX = base64.StdEncoding.EncodeToString(filled)
	out.Message = fmt.Sprintf("PREVIEW ONLY, %s was not changed. Call again with write=true to save %d column(s) and %d chart range(s) into it.", rel, len(res.Filled), len(res.Charts))
	return nil, out, nil
}

// resolveWorkbookPath confines the workbook to the server working directory.
func resolveWorkbookPath(rel string) (string, error) {
	if !strings.EqualFold(filepath.Ext(rel), ".xlsx") {
		return "", fmt.Errorf("file %q must be an .xlsx workbook", rel)
	}
	if !filepath.IsLocal(rel) {
		return "", fmt.Errorf("file %q must be a path inside the server working directory", rel)
	}
	root, err := filepath.EvalSymlinks(".")
	if err != nil {
		return "", err
	}
	abs := filepath.Join(root, rel)
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		if r, err := filepath.Rel(root, resolved); err != nil || !filepath.IsLocal(r) {
			return "", fmt.Errorf("file %q resolves outside the server working directory", rel)
		}
	}
	return abs, nil
}
