package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"go-word-create/internal/config"
	"go-word-create/internal/jiraservice"
	"go-word-create/internal/mcpserver"
)

func main() {
	workdir := flag.String("workdir", ".", "directory containing the .env file (the process changes into it before loading config)")
	debug := flag.Bool("debug", false, "enable verbose logging (stderr)")
	flag.Parse()

	// stdout carries the MCP protocol; all logging must go to stderr.
	log.SetOutput(os.Stderr)
	if *debug {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	if err := os.Chdir(*workdir); err != nil {
		log.Fatalf("failed to change into -workdir %q: %v", *workdir, err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	jiraSvc, err := jiraservice.NewJiraService(
		cfg.JiraURL, cfg.JiraUsername, cfg.JiraAPIToken, cfg.JiraEpicField, cfg.JiraSPField,
	)
	if err != nil {
		log.Fatalf("failed to create Jira client: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	log.Printf("jira-reports MCP server starting on stdio (teams=%d, labels=%d)", len(cfg.Teams), len(cfg.ReportLabels))
	if err := mcpserver.New(cfg, jiraSvc).Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatalf("mcp server stopped: %v", err)
	}
}
