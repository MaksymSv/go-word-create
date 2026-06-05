package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"

	"go-word-create/internal/config"
	"go-word-create/internal/jiraservice"
	"go-word-create/internal/labelreport"
	"go-word-create/internal/word"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	sprintName := flag.String("sprint", "", "Sprint name (required)")
	outputFile := flag.String("output", cfg.OutputFile, "Output .docx file path")
	format := flag.String("format", "short", "Report format: short or full")
	debug := flag.Bool("debug", false, "Print report to stdout; do not write Word document")
	flag.Parse()

	if *sprintName == "" {
		fmt.Fprintln(os.Stderr, "Error: -sprint flag is required")
		flag.Usage()
		os.Exit(1)
	}
	if *format != "short" && *format != "full" {
		fmt.Fprintf(os.Stderr, "Error: -format must be \"short\" or \"full\", got %q\n", *format)
		os.Exit(1)
	}

	jiraService, err := jiraservice.NewJiraService(
		cfg.JiraURL, cfg.JiraUsername, cfg.JiraAPIToken, cfg.JiraEpicField, cfg.JiraSPField,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create Jira service: %v\n", err)
		os.Exit(1)
	}

	issues, err := jiraService.GetSprintIssues(cfg.ProjectKey, cfg.BoardName, *sprintName, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: sprint %q on board %q: %v\n", *sprintName, cfg.BoardName, err)
		os.Exit(1)
	}

	reports := labelreport.Aggregate(issues, cfg.ReportLabels)

	if *debug {
		if *format == "full" {
			printFullFormatConsole(reports)
		} else {
			printShortFormatConsole(reports)
		}
		return
	}

	doc := word.NewDocument()
	if *format == "full" {
		renderFullFormatDoc(doc, reports)
	} else {
		renderShortFormatDoc(doc, reports)
	}

	if err := doc.SaveDocumentToFile(outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to save document %q: %v\n", *outputFile, err)
		os.Exit(1)
	}
	fmt.Printf("Created document %q (%d component(s))\n", *outputFile, len(reports))
}

// --- Word document renderers ---

func formatPct(v float64) string {
	if math.Trunc(v) == v {
		return strconv.Itoa(int(v)) + "%"
	}
	return strconv.FormatFloat(v, 'f', 1, 64) + "%"
}

func shortFormatConfig() word.TableConfig {
	cfg := word.DefaultConfig()
	cfg.ColumnHeaders = []string{"Label", "Count", "Count,%", "Total SP", "Total SP,%"}
	cfg.ColumnWidths = []float64{25, 10, 10, 10, 10}
	cfg.CenterColumns = []int{1, 2, 3, 4}
	return cfg
}

func fullFormatConfig() word.TableConfig {
	cfg := word.DefaultConfig()
	cfg.ColumnHeaders = []string{"Label", "Count", "Count,%", "Total SP", "Total SP,%", "Key", "Summary", "SP"}
	cfg.ColumnWidths = []float64{13, 6, 6, 6, 6, 10, 47, 6}
	cfg.CenterColumns = []int{1, 2, 3, 4, 7}
	return cfg
}

func unlabeledConfig() word.TableConfig {
	cfg := word.DefaultConfig()
	cfg.ColumnHeaders = []string{"Key", "Summary", "SP"}
	cfg.ColumnWidths = []float64{13, 78, 9}
	cfg.CenterColumns = []int{0, 2}
	return cfg
}

func renderShortFormatDoc(doc *word.Doc, reports []labelreport.ComponentReport) {
	for _, r := range reports {
		doc.AddHeading(1, r.ComponentName)
		t := word.NewTable(&doc.WordDocument, shortFormatConfig())
		t.AddHeaderRow()
		for _, g := range r.LabelGroups {
			t.AddDataRow([]string{
				g.LabelName,
				strconv.Itoa(g.Count),
				formatPct(g.CountPct),
				strconv.FormatFloat(g.TotalSP, 'f', 1, 64),
				formatPct(g.TotalSPPct),
			})
		}
		t.AddDataRow([]string{"Total", "", formatPct(r.TotalLabeledCountPct), "", formatPct(r.TotalLabeledSPPct)})
		appendUnlabeledTable(doc, r.UnlabeledIssues)
	}
}

func renderFullFormatDoc(doc *word.Doc, reports []labelreport.ComponentReport) {
	for _, r := range reports {
		doc.AddHeading(1, r.ComponentName)
		t := word.NewTable(&doc.WordDocument, fullFormatConfig())
		t.AddHeaderRow()
		for _, g := range r.LabelGroups {
			if len(g.Issues) == 0 {
				t.AddDataRow([]string{g.LabelName, "0", formatPct(g.CountPct), "0.0", formatPct(g.TotalSPPct), "", "", ""})
				continue
			}
			for _, iss := range g.Issues {
				t.AddDataRow([]string{
					g.LabelName,
					strconv.Itoa(g.Count),
					formatPct(g.CountPct),
					strconv.FormatFloat(g.TotalSP, 'f', 1, 64),
					formatPct(g.TotalSPPct),
					iss.Key,
					iss.Summary,
					strconv.FormatFloat(iss.StoryPoints, 'f', 1, 64),
				})
			}
		}
		t.AddDataRow([]string{"Total", "", formatPct(r.TotalLabeledCountPct), "", formatPct(r.TotalLabeledSPPct), "", "", ""})
		appendUnlabeledTable(doc, r.UnlabeledIssues)
	}
}

func appendUnlabeledTable(doc *word.Doc, issues []jiraservice.Issue) {
	if len(issues) == 0 {
		return
	}
	doc.AddHeading(2, "Unlabeled Issues")
	t := word.NewTable(&doc.WordDocument, unlabeledConfig())
	t.AddHeaderRow()
	for _, iss := range issues {
		t.AddDataRow([]string{
			iss.Key,
			iss.Summary,
			strconv.FormatFloat(iss.StoryPoints, 'f', 1, 64),
		})
	}
}

// --- Console renderers ---

func printShortFormatConsole(reports []labelreport.ComponentReport) {
	for _, r := range reports {
		fmt.Printf("\nComponent: %s\n", r.ComponentName)
		fmt.Printf("  %-35s | %5s | %8s | %8s | %10s\n", "Label", "Count", "Count,%", "Total SP", "Total SP,%")
		fmt.Println("  " + repeat("-", 80))
		for _, g := range r.LabelGroups {
			fmt.Printf("  %-35s | %5d | %8s | %8.1f | %10s\n",
				g.LabelName, g.Count, formatPct(g.CountPct), g.TotalSP, formatPct(g.TotalSPPct))
		}
		fmt.Println("  " + repeat("-", 80))
		fmt.Printf("  %-35s | %5s | %8s | %8s | %10s\n",
			"Total", "", formatPct(r.TotalLabeledCountPct), "", formatPct(r.TotalLabeledSPPct))
		printUnlabeledConsole(r.UnlabeledIssues)
	}
}

func printFullFormatConsole(reports []labelreport.ComponentReport) {
	for _, r := range reports {
		fmt.Printf("\nComponent: %s\n", r.ComponentName)
		fmt.Printf("  %-30s | %5s | %8s | %8s | %10s | %-12s | %-50s | %4s\n",
			"Label", "Count", "Count,%", "Total SP", "Total SP,%", "Key", "Summary", "SP")
		fmt.Println("  " + repeat("-", 140))
		for _, g := range r.LabelGroups {
			if len(g.Issues) == 0 {
				fmt.Printf("  %-30s | %5d | %8s | %8.1f | %10s | %-12s | %-50s | %4s\n",
					g.LabelName, g.Count, formatPct(g.CountPct), g.TotalSP, formatPct(g.TotalSPPct), "", "", "")
				continue
			}
			for _, iss := range g.Issues {
				fmt.Printf("  %-30s | %5d | %8s | %8.1f | %10s | %-12s | %-50s | %4.1f\n",
					g.LabelName, g.Count, formatPct(g.CountPct), g.TotalSP, formatPct(g.TotalSPPct),
					iss.Key, truncate(iss.Summary, 50), iss.StoryPoints)
			}
		}
		fmt.Println("  " + repeat("-", 140))
		fmt.Printf("  %-30s | %5s | %8s | %8s | %10s | %-12s | %-50s | %4s\n",
			"Total", "", formatPct(r.TotalLabeledCountPct), "", formatPct(r.TotalLabeledSPPct), "", "", "")
		printUnlabeledConsole(r.UnlabeledIssues)
	}
}

func printUnlabeledConsole(issues []jiraservice.Issue) {
	if len(issues) == 0 {
		return
	}
	fmt.Printf("\n  Unlabeled Issues:\n")
	fmt.Printf("  %-12s | %-60s | %4s\n", "Key", "Summary", "SP")
	fmt.Println("  " + repeat("-", 83))
	for _, iss := range issues {
		fmt.Printf("  %-12s | %-60s | %4.1f\n", iss.Key, truncate(iss.Summary, 60), iss.StoryPoints)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}
