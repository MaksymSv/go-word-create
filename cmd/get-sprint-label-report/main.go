package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"go-word-create/internal/config"
	"go-word-create/internal/jiraservice"
	"go-word-create/internal/labelreport"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	sprintName := flag.String("sprint", "", "Sprint name (required)")
	format := flag.String("format", "short", "Report format: short or full")
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

	sprintNames, err := parseSprintNames(*sprintName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to parse -sprint: %v\n", err)
		os.Exit(1)
	}

	jiraService, err := jiraservice.NewJiraService(
		cfg.JiraURL, cfg.JiraUsername, cfg.JiraAPIToken, cfg.JiraEpicField, cfg.JiraSPField,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create Jira service: %v\n", err)
		os.Exit(1)
	}

	for _, sprint := range sprintNames {
		issues, err := jiraService.GetSprintIssues(cfg.ProjectKey, sprint, cfg.GetAllBoardNames(), cfg.JiraIssueTypes)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: sprint %q: %v\n", sprint, err)
			os.Exit(1)
		}

		reports := labelreport.Aggregate(issues, cfg.ReportLabels)

		if len(sprintNames) > 1 {
			fmt.Printf("\n=== Sprint: %s ===\n", sprint)
		}
		if *format == "full" {
			printFullFormatConsole(reports)
		} else {
			printShortFormatConsole(reports)
		}
	}
}

func parseSprintNames(raw string) ([]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("sprint flag is empty")
	}

	parts := strings.Split(trimmed, ",")
	sprintNames := make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			return nil, fmt.Errorf("sprint list contains an empty sprint name")
		}
		sprintNames = append(sprintNames, name)
	}

	return sprintNames, nil
}

// --- Console renderers ---

func formatPct(v float64) string {
	if math.Trunc(v) == v {
		return strconv.Itoa(int(v)) + " %"
	}
	return strconv.FormatFloat(v, 'f', 0, 64) + " %"
}

// --- Console renderers ---

func printShortFormatConsole(reports []labelreport.ComponentReport) {
	for _, r := range reports {
		fmt.Printf("\nComponent: %s\n", r.ComponentName)
		fmt.Printf("  %-35s | %5s | %8s | %8s | %10s\n", "Label", "Count", "Count,%", "Total SP", "Total SP,%")
		fmt.Println("  " + repeat("-", 80))
		for _, g := range r.LabelGroups {
			fmt.Printf("  %-35s | %5d | %8s | %8.0f | %10s\n",
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
		fmt.Printf("  %-30s | %5s | %8s | %8s | %10s | %-13s | %-50s | %4s\n",
			"Label", "Count", "Count,%", "Total SP", "Total SP,%", "Key", "Summary", "SP")
		fmt.Println("  " + repeat("-", 150))
		for _, g := range r.LabelGroups {
			if len(g.Issues) == 0 {
				fmt.Printf("  %-30s | %5d | %8s | %8.1f | %10s | %-13s | %-50s | %4s\n",
					g.LabelName, g.Count, formatPct(g.CountPct), g.TotalSP, formatPct(g.TotalSPPct), "", "", "")
				continue
			}
			for _, iss := range g.Issues {
				fmt.Printf("  %-30s | %5d | %8s | %8.1f | %10s | %-13s | %-50s | %4.1f\n",
					g.LabelName, g.Count, formatPct(g.CountPct), g.TotalSP, formatPct(g.TotalSPPct),
					iss.Key, truncate(iss.Summary, 50), iss.StoryPoints)
			}
		}
		fmt.Println("  " + repeat("-", 150))
		fmt.Printf("  %-30s | %5s | %8s | %8s | %10s | %-13s | %-50s | %4s\n",
			"Total", "", formatPct(r.TotalLabeledCountPct), "", formatPct(r.TotalLabeledSPPct), "", "", "")
		printUnlabeledConsole(r.UnlabeledIssues)
	}
}

func printUnlabeledConsole(issues []jiraservice.Issue) {
	if len(issues) == 0 {
		return
	}
	fmt.Printf("\n  Unlabeled Issues:\n")
	fmt.Printf("  %-13s | %-60s | %4s\n", "Key", "Summary", "SP")
	fmt.Println("  " + repeat("-", 84))
	for _, iss := range issues {
		fmt.Printf("  %-13s | %-60s | %4.1f\n", iss.Key, truncate(iss.Summary, 60), iss.StoryPoints)
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
