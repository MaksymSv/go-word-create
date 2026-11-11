package chartfill

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"go-word-create/internal/config"
	"go-word-create/internal/jiraservice"
	"go-word-create/internal/labelreport"
)

// DefaultFile is the workbook the CLI and MCP tool fill when no file is given.
const DefaultFile = "proj-docs/AI Assisted Work Items Chart.xlsx"

// DefaultPrefixes maps the team prefix in the workbook's "Team" row (and in Jira
// sprint names such as ST26.20) to the Jira component the numbers come from.
var DefaultPrefixes = map[string]string{
	"ST": "STABLETEK",
	"PC": "PROCESSING",
}

// Metric is the kind of percentage a row holds.
type Metric string

const (
	MetricCount Metric = "count" // work items, % of issues
	MetricSP    Metric = "sp"    // story points, % of total SP
)

// Row describes one data row of the sheet.
type Row struct {
	Row    int    `json:"row"`
	Title  string `json:"title"`           // column A
	Label  string `json:"label,omitempty"` // column B (jira label); empty for total rows
	Total  bool   `json:"total,omitempty"`
	Metric Metric `json:"metric"`
}

// Column is a sprint column that has headers.
type Column struct {
	Ref       string // column letters, e.g. "Y"
	Col       int
	Header    string // row 1 text, e.g. "S26.20"
	Prefix    string // row 2 text, e.g. "ST"
	Sprint    string // Jira sprint name, e.g. "ST26.20"
	Component string
	EmptyRows []Row
	Problem   string // non-empty when the column cannot be filled
}

var (
	spRe        = regexp.MustCompile(`(?i)\bSP\b|story\s*point`)
	countRe     = regexp.MustCompile(`(?i)\bWI\b|count|work\s*item`)
	sprintNumRe = regexp.MustCompile(`^[A-Za-z]+`)
)

// Rows returns the metric rows, identified by the "jira label" column (B).
// A row with an empty label whose title starts with "total" is a total row.
func (w *Workbook) Rows() []Row {
	var rows []Row
	for r := headerRowTeam + 1; r <= w.maxRow; r++ {
		title := w.cells[cellRef(1, r)]
		label := w.cells[cellRef(labelCol, r)]
		total := label == "" && strings.HasPrefix(strings.ToLower(title), "total")
		if label == "" && !total {
			continue
		}
		var m Metric
		switch {
		case spRe.MatchString(title):
			m = MetricSP
		case countRe.MatchString(title):
			m = MetricCount
		default:
			continue
		}
		rows = append(rows, Row{Row: r, Title: title, Label: label, Total: total, Metric: m})
	}
	return rows
}

// Pending lists the columns that have a sprint header and team prefix but at
// least one empty data cell.
func (w *Workbook) Pending(prefixes map[string]string) []Column {
	rows := w.Rows()
	var out []Column
	for col := firstDataCol; col <= w.maxHeaderCol(); col++ {
		header, prefix := w.text(col, headerRowSprint), w.text(col, headerRowTeam)
		if header == "" || prefix == "" {
			continue
		}
		var empty []Row
		for _, r := range rows {
			if w.isEmpty(col, r.Row) {
				empty = append(empty, r)
			}
		}
		if len(empty) == 0 {
			continue
		}
		c := Column{Ref: colName(col), Col: col, Header: header, Prefix: prefix, EmptyRows: empty}
		number := sprintNumRe.ReplaceAllString(header, "")
		component, known := prefixes[strings.ToUpper(prefix)]
		switch {
		case number == "":
			c.Problem = fmt.Sprintf("cannot derive a sprint number from header %q", header)
		case !known:
			c.Problem = fmt.Sprintf("unknown team prefix %q", prefix)
		default:
			c.Sprint = strings.ToUpper(prefix) + number
			c.Component = component
		}
		out = append(out, c)
	}
	return out
}

// Fetcher returns the label report of one component in one sprint.
type Fetcher func(sprint, component string) (labelreport.ComponentReport, error)

// FilledCell is one value written into the sheet.
type FilledCell struct {
	Cell   string `json:"cell"`
	Title  string `json:"title"`
	Value  int    `json:"value"`
	Metric Metric `json:"metric"`
}

type FilledColumn struct {
	Column    string       `json:"column"`
	Sprint    string       `json:"sprint"`
	Component string       `json:"component"`
	Cells     []FilledCell `json:"cells"`
}

type SkippedColumn struct {
	Column string `json:"column"`
	Header string `json:"header"`
	Sprint string `json:"sprint,omitempty"`
	Reason string `json:"reason"`
}

type Result struct {
	Filled  []FilledColumn  `json:"filled"`
	Skipped []SkippedColumn `json:"skipped"`
	// Charts lists chart series ranges widened to cover new sprint columns.
	Charts []ChartRange `json:"chartRanges"`
}

// Changed reports whether saving the workbook would change anything.
func (r Result) Changed() bool { return len(r.Filled) > 0 || len(r.Charts) > 0 }

// Fill writes the percentages of every pending column into the workbook.
// Only empty cells are written. A column whose sprint cannot be fetched is
// reported in Result.Skipped; a label from column B that is not a tracked report
// label aborts with an error. Chart ranges are widened to the last header column
// (see SyncCharts). With dryRun the workbook is left unchanged.
func (w *Workbook) Fill(fetch Fetcher, prefixes map[string]string, dryRun bool) (Result, error) {
	res := Result{Filled: []FilledColumn{}, Skipped: []SkippedColumn{}, Charts: []ChartRange{}}
	for _, col := range w.Pending(prefixes) {
		if col.Problem != "" {
			res.Skipped = append(res.Skipped, SkippedColumn{Column: col.Ref, Header: col.Header, Sprint: col.Sprint, Reason: col.Problem})
			continue
		}
		report, err := fetch(col.Sprint, col.Component)
		if err != nil {
			res.Skipped = append(res.Skipped, SkippedColumn{Column: col.Ref, Header: col.Header, Sprint: col.Sprint, Reason: err.Error()})
			continue
		}
		filled := FilledColumn{Column: col.Ref, Sprint: col.Sprint, Component: col.Component}
		for _, row := range col.EmptyRows {
			v, err := valueFor(report, row)
			if err != nil {
				return res, fmt.Errorf("column %s (%s): %w", col.Ref, col.Sprint, err)
			}
			if !dryRun {
				if err := w.setNumber(col.Col, row.Row, v); err != nil {
					return res, err
				}
			}
			filled.Cells = append(filled.Cells, FilledCell{Cell: cellRef(col.Col, row.Row), Title: row.Title, Value: v, Metric: row.Metric})
		}
		res.Filled = append(res.Filled, filled)
	}
	charts, err := w.SyncCharts(dryRun)
	if err != nil {
		return res, err
	}
	res.Charts = append(res.Charts, charts...)
	return res, nil
}

func valueFor(rep labelreport.ComponentReport, row Row) (int, error) {
	if row.Total {
		if row.Metric == MetricSP {
			return roundPct(rep.TotalLabeledSPPct), nil
		}
		return roundPct(rep.TotalLabeledCountPct), nil
	}
	for _, g := range rep.LabelGroups {
		if g.LabelName == row.Label {
			if row.Metric == MetricSP {
				return roundPct(g.TotalSPPct), nil
			}
			return roundPct(g.CountPct), nil
		}
	}
	return 0, fmt.Errorf("row %d: label %q is not a tracked report label (REPORT_LABELS)", row.Row, row.Label)
}

func roundPct(v float64) int { return int(math.Round(v)) }

// SprintSource is the part of the Jira service the fetcher needs.
type SprintSource interface {
	GetSprintIssues(projectKey, sprintName string, boardNames, issueTypes []string) ([]jiraservice.Issue, error)
}

// NewJiraFetcher builds a Fetcher on top of the same data path as get-sprint-label-report.
func NewJiraFetcher(src SprintSource, cfg *config.Config) Fetcher {
	return func(sprint, component string) (labelreport.ComponentReport, error) {
		issues, err := src.GetSprintIssues(cfg.ProjectKey, sprint, cfg.GetAllBoardNames(), cfg.JiraIssueTypes)
		if err != nil {
			return labelreport.ComponentReport{}, err
		}
		for _, r := range labelreport.Aggregate(issues, cfg.ReportLabels) {
			if r.ComponentName == component {
				return r, nil
			}
		}
		return labelreport.ComponentReport{}, fmt.Errorf("no %s issues in sprint %s", component, sprint)
	}
}
