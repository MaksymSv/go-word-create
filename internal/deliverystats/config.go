package deliverystats

import "go-word-create/internal/config"

// OptionsFromConfig maps the .env-driven configuration to service options.
func OptionsFromConfig(cfg *config.Config) Options {
	return Options{
		ProjectKey:           cfg.ProjectKey,
		IssueTypes:           cfg.JiraIssueTypes,
		CompletedResolutions: cfg.DeliveryCompletedResolutions,
		SprintCount:          cfg.DeliverySprintCount,
		BugDays:              cfg.DeliveryBugDays,
		CacheDir:             cfg.DeliveryCacheDir,
		CacheTTL:             cfg.DeliveryCacheTTL,
	}
}

// TeamsFromConfig lists the configured teams in .env order.
func TeamsFromConfig(cfg *config.Config) []Team {
	teams := make([]Team, len(cfg.Teams))
	for i, t := range cfg.Teams {
		teams[i] = Team{Component: t.ComponentName, Board: t.BoardName}
	}
	return teams
}
