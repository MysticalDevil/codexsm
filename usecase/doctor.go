package usecase

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/MysticalDevil/codexsm/internal/core"
	"github.com/MysticalDevil/codexsm/session"
)

type DoctorLevel string

const (
	DoctorPass DoctorLevel = "PASS"
	DoctorWarn DoctorLevel = "WARN"
	DoctorFail DoctorLevel = "FAIL"
)

type DoctorCheck struct {
	Name   string
	Level  DoctorLevel
	Detail string
}

type DoctorRiskSample struct {
	Level     session.RiskLevel  `json:"level"`
	Reason    session.RiskReason `json:"reason"`
	Health    session.Health     `json:"health"`
	SessionID string             `json:"session_id"`
	Path      string             `json:"path"`
	Detail    string             `json:"detail,omitempty"`
}

type DoctorRiskReport struct {
	SessionsTotal  int                `json:"sessions_total"`
	RiskTotal      int                `json:"risk_total"`
	RiskRate       float64            `json:"risk_rate"`
	High           int                `json:"high"`
	Medium         int                `json:"medium"`
	IntegrityCheck bool               `json:"integrity_check"`
	SampleLimit    int                `json:"sample_limit"`
	Samples        []DoctorRiskSample `json:"samples"`
}

type DoctorRiskInput struct {
	SessionsRoot   string
	SampleLimit    int
	IntegrityCheck bool
	Repository     core.SessionRepository
	Evaluator      core.RiskEvaluator
}

func DoctorRisk(in DoctorRiskInput) (DoctorRiskReport, error) {
	repo := in.Repository
	if repo == nil {
		repo = core.ScannerRepository{}
	}
	evaluator := in.Evaluator
	if evaluator == nil {
		evaluator = core.SessionRiskEvaluator{}
	}
	limit := in.SampleLimit
	if limit <= 0 {
		limit = 10
	}

	items, err := repo.ScanSessions(in.SessionsRoot)
	if err != nil {
		return DoctorRiskReport{}, err
	}
	type riskyItem struct {
		Session session.Session
		Risk    session.Risk
	}
	var checker session.IntegrityChecker
	if in.IntegrityCheck {
		checker = session.SHA256SidecarChecker
	}

	risky := make([]riskyItem, 0, len(items))
	highCount := 0
	mediumCount := 0
	for _, s := range items {
		r := evaluator.Evaluate(s, checker)
		if r.Level == session.RiskNone {
			continue
		}
		risky = append(risky, riskyItem{Session: s, Risk: r})
		switch r.Level {
		case session.RiskHigh:
			highCount++
		case session.RiskMedium:
			mediumCount++
		}
	}

	sort.SliceStable(risky, func(i, j int) bool {
		ri := risky[i].Risk.Level
		rj := risky[j].Risk.Level
		if ri != rj {
			return riskRank(ri) > riskRank(rj)
		}
		c := risky[j].Session.UpdatedAt.Compare(risky[i].Session.UpdatedAt)
		if c != 0 {
			return c < 0
		}
		return risky[i].Session.SessionID < risky[j].Session.SessionID
	})

	rate := 0.0
	if len(items) > 0 {
		rate = float64(len(risky)) / float64(len(items)) * 100
	}
	report := DoctorRiskReport{
		SessionsTotal:  len(items),
		RiskTotal:      len(risky),
		RiskRate:       rate,
		High:           highCount,
		Medium:         mediumCount,
		IntegrityCheck: in.IntegrityCheck,
		SampleLimit:    limit,
		Samples:        make([]DoctorRiskSample, 0, min(limit, len(risky))),
	}
	for i, item := range risky {
		if i >= limit {
			break
		}
		report.Samples = append(report.Samples, DoctorRiskSample{
			Level:     item.Risk.Level,
			Reason:    item.Risk.Reason,
			Health:    item.Session.Health,
			SessionID: item.Session.SessionID,
			Path:      item.Session.Path,
			Detail:    item.Risk.Detail,
		})
	}
	return report, nil
}

type DoctorHostPathInput struct {
	SessionsRoot string
	SessionsErr  error
	Repository   core.SessionRepository
	CompactPath  func(string, int) string
}

func CheckSessionHostPaths(in DoctorHostPathInput) DoctorCheck {
	if in.SessionsErr != nil {
		return DoctorCheck{Name: "session_host_paths", Level: DoctorWarn, Detail: "skipped: sessions_root unresolved"}
	}
	if _, err := os.Stat(in.SessionsRoot); err != nil {
		if os.IsNotExist(err) {
			return DoctorCheck{Name: "session_host_paths", Level: DoctorWarn, Detail: "skipped: sessions_root missing"}
		}
		return DoctorCheck{Name: "session_host_paths", Level: DoctorFail, Detail: err.Error()}
	}

	repo := in.Repository
	if repo == nil {
		repo = core.ScannerRepository{}
	}
	items, err := repo.ScanSessions(in.SessionsRoot)
	if err != nil {
		return DoctorCheck{Name: "session_host_paths", Level: DoctorFail, Detail: err.Error()}
	}
	if len(items) == 0 {
		return DoctorCheck{Name: "session_host_paths", Level: DoctorPass, Detail: "no sessions found"}
	}

	withHost := 0
	missingCountByHost := make(map[string]int)
	for _, s := range items {
		host := strings.TrimSpace(s.HostDir)
		if host == "" {
			continue
		}
		withHost++
		if _, statErr := os.Stat(host); statErr == nil {
			continue
		} else if os.IsNotExist(statErr) {
			missingCountByHost[host]++
		} else {
			return DoctorCheck{Name: "session_host_paths", Level: DoctorFail, Detail: fmt.Sprintf("stat host %s: %v", host, statErr)}
		}
	}
	if len(missingCountByHost) == 0 {
		return DoctorCheck{
			Name:   "session_host_paths",
			Level:  DoctorPass,
			Detail: fmt.Sprintf("all host paths exist (sessions=%d with_host=%d)", len(items), withHost),
		}
	}

	hosts := make([]string, 0, len(missingCountByHost))
	for host := range missingCountByHost {
		hosts = append(hosts, host)
	}
	sort.Strings(hosts)
	displayHosts := hosts
	if len(displayHosts) > 3 {
		displayHosts = displayHosts[:3]
	}

	compact := in.CompactPath
	if compact == nil {
		compact = func(v string, _ int) string { return v }
	}
	hostLines := make([]string, 0, len(displayHosts))
	for _, host := range displayHosts {
		hostLines = append(hostLines, fmt.Sprintf("- %s (%d)", compact(host, 56), missingCountByHost[host]))
	}

	suggestHost := displayHosts[0]
	return DoctorCheck{
		Name:  "session_host_paths",
		Level: DoctorWarn,
		Detail: fmt.Sprintf(
			"missing_hosts=%d impacted_sessions=%d\nsample_hosts:\n%s\nrecommended_actions:\n1. review: codexsm list --host-contains %s\n2. migrate (soft-delete): codexsm delete --host-contains %s\n3. optional hard-delete: codexsm delete --host-contains %s --dry-run=false --confirm --hard",
			len(missingCountByHost),
			withHost,
			strings.Join(hostLines, "\n"),
			suggestHost,
			suggestHost,
			suggestHost,
		),
	}
}

func riskRank(level session.RiskLevel) int {
	switch level {
	case session.RiskHigh:
		return 2
	case session.RiskMedium:
		return 1
	default:
		return 0
	}
}
