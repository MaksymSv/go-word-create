package main

import (
	"flag"
	"fmt"
	"os"

	"go-word-create/internal/chartfill"
	"go-word-create/internal/config"
	"go-word-create/internal/jiraservice"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	file := flag.String("file", chartfill.DefaultFile, "Workbook to fill")
	output := flag.String("output", "", "Write the filled workbook here instead of updating -file in place")
	dryRun := flag.Bool("dry-run", false, "Show what would be filled without writing anything")
	flag.Parse()

	data, err := os.ReadFile(*file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	wb, err := chartfill.Open(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s: %v\n", *file, err)
		os.Exit(1)
	}

	jiraService, err := jiraservice.NewJiraService(
		cfg.JiraURL, cfg.JiraUsername, cfg.JiraAPIToken, cfg.JiraEpicField, cfg.JiraSPField,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create Jira service: %v\n", err)
		os.Exit(1)
	}

	res, err := wb.Fill(chartfill.NewJiraFetcher(jiraService, cfg), chartfill.DefaultPrefixes, *dryRun)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	for _, f := range res.Filled {
		fmt.Printf("%s  %s (%s)\n", f.Column, f.Sprint, f.Component)
		for _, c := range f.Cells {
			fmt.Printf("    %-5s %-30s %d\n", c.Cell, c.Title, c.Value)
		}
	}
	for _, c := range res.Charts {
		fmt.Printf("chart %s: %s -> %s\n", c.Chart, c.From, c.To)
	}
	for _, s := range res.Skipped {
		fmt.Printf("%s  %s skipped: %s\n", s.Column, s.Header, s.Reason)
	}
	if !res.Changed() {
		fmt.Println("Nothing to fill.")
		return
	}
	if *dryRun {
		fmt.Println("Dry run: workbook not changed.")
		return
	}

	out, err := wb.Bytes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	target := *file
	if *output != "" {
		target = *output
	}
	if err := chartfill.WriteFileAtomic(target, out); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Workbook saved: %s\n", target)
}
