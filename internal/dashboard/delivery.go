package dashboard

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"go-word-create/internal/deliverystats"
)

// DeliverySettings tells the delivery page which options it may offer.
type DeliverySettings struct {
	SprintCount int `json:"sprintCount"`
	MaxSprints  int `json:"maxSprints"`
	BugDays     int `json:"bugDays"`
}

func (h *DashboardHandler) serveDelivery(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, h.webFS, "delivery.html")
}

func (h *DashboardHandler) getDeliverySettings(w http.ResponseWriter, r *http.Request) {
	opt := deliverystats.OptionsFromConfig(h.cfg)
	writeJSON(w, DeliverySettings{
		SprintCount: h.delivery.DefaultSprintCount(),
		MaxSprints:  deliverystats.MaxSprints,
		BugDays:     opt.BugDays,
	})
}

func (h *DashboardHandler) getDeliveryReport(w http.ResponseWriter, r *http.Request) {
	component := r.PathValue("component")
	team := h.cfg.GetTeamByComponent(component)
	if team.ComponentName == "" {
		writeError(w, http.StatusNotFound, fmt.Sprintf("team %q not found", component))
		return
	}
	sprints := 0
	if v := r.URL.Query().Get("sprints"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > deliverystats.MaxSprints {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("sprints must be an integer between 1 and %d", deliverystats.MaxSprints))
			return
		}
		sprints = n
	}
	rep, err := h.delivery.Report(deliverystats.Team{Component: team.ComponentName, Board: team.BoardName}, sprints, r.URL.Query().Get("refresh") == "1")
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("failed to build delivery report for team %q: %v", component, err))
		return
	}
	writeJSON(w, rep)
}

// WarmDelivery builds the default delivery report of every team in the
// background so the first page view does not wait for the Jira scan.
func (h *DashboardHandler) WarmDelivery() {
	go func() {
		for _, t := range deliverystats.TeamsFromConfig(h.cfg) {
			if _, err := h.delivery.Report(t, 0, false); err != nil {
				log.Printf("dashboard: delivery warm-up for team %q failed: %v", t.Component, err)
			}
		}
		log.Printf("dashboard: delivery reports warmed")
	}()
}
