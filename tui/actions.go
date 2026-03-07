package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/MysticalDevil/codexsm/audit"
	"github.com/MysticalDevil/codexsm/internal/restoreexec"
	"github.com/MysticalDevil/codexsm/session"
)

func (m *tuiModel) runDryRunPreview() {
	prevDryRun := m.dryRun
	m.dryRun = true
	m.requestDelete()
	m.dryRun = prevDryRun
}

func (m *tuiModel) requestDelete() {
	if m.source == "trash" {
		m.status = "Current source is trash; use r to restore."
		return
	}
	selected, ok := m.selectedSession()
	if !ok {
		m.status = "No session selected."
		return
	}
	if !m.dryRun && !m.confirm {
		m.status = "Real delete requires --confirm."
		return
	}
	if !m.dryRun && !m.yes {
		m.pendingAction = "delete"
		m.pendingID = selected.SessionID
		m.status = fmt.Sprintf("Confirm delete %s: press y to continue, n to cancel.", shortID(selected.SessionID))
		return
	}
	m.runDelete(selected)
}

func (m *tuiModel) requestHostMigrate() {
	if m.source == "trash" {
		m.status = "Current source is trash; host-migrate applies to sessions."
		return
	}
	selected, ok := m.selectedSession()
	if !ok {
		m.status = "No session selected."
		return
	}
	host := strings.TrimSpace(selected.HostDir)
	if host == "" {
		m.status = "Selected session has no host path."
		return
	}
	if !m.sessionHostMissing(selected) {
		m.status = "Selected host path exists; migrate strategy targets missing hosts."
		return
	}
	candidates := m.migrateCandidatesForHost(host)
	if len(candidates) == 0 {
		m.status = "No sessions matched selected host."
		return
	}
	if !m.dryRun && !m.confirm {
		m.status = "Real migrate-host requires --confirm."
		return
	}
	if !m.dryRun && !m.yes {
		m.pendingAction = "migrate-host"
		m.pendingID = selected.SessionID
		m.pendingHost = host
		m.status = fmt.Sprintf("Confirm migrate host %s (sessions=%d): press y to continue, n to cancel.", truncateDisplay(host, 48), len(candidates))
		return
	}
	m.runHostMigrate(host, candidates)
}

func (m *tuiModel) runDelete(selected session.Session) {
	sel := session.Selector{ID: selected.SessionID}
	sum, err := session.DeleteSessions(
		[]session.Session{selected},
		sel,
		session.DeleteOptions{
			DryRun:       m.dryRun,
			Confirm:      m.confirm,
			Yes:          true,
			Hard:         m.hardDelete,
			MaxBatch:     max(1, m.maxBatch),
			SessionsRoot: m.sessionsRoot,
			TrashRoot:    m.trashRoot,
		},
	)
	m.pendingAction = ""
	m.pendingID = ""
	m.pendingHost = ""
	if err != nil {
		m.status = "delete failed: " + err.Error()
		return
	}
	m.persistActionLog(sum.Action, sum.Simulation, sel, []session.Session{selected}, sum.AffectedBytes, sum.Results, sum.ErrorSummary)
	m.status = fmt.Sprintf(
		"delete: action=%s matched=%d affected=%s",
		sum.Action,
		sum.MatchedCount,
		formatBytesIEC(sum.AffectedBytes),
	)
	if !m.dryRun && sum.Succeeded > 0 {
		m.removeSelectedSession()
	}
}

func (m *tuiModel) migrateCandidatesForHost(host string) []session.Session {
	host = strings.TrimSpace(host)
	if host == "" {
		return nil
	}
	out := make([]session.Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		if strings.EqualFold(strings.TrimSpace(s.HostDir), host) {
			out = append(out, s)
		}
	}
	return out
}

func (m *tuiModel) runHostMigrate(host string, candidates []session.Session) {
	sel := session.Selector{HostContains: host}
	sum, err := session.DeleteSessions(
		candidates,
		sel,
		session.DeleteOptions{
			DryRun:       m.dryRun,
			Confirm:      m.confirm,
			Yes:          true,
			Hard:         false,
			MaxBatch:     max(1, m.maxBatch),
			SessionsRoot: m.sessionsRoot,
			TrashRoot:    m.trashRoot,
		},
	)
	m.pendingAction = ""
	m.pendingID = ""
	m.pendingHost = ""
	if err != nil {
		m.status = "migrate-host failed: " + err.Error()
		return
	}
	m.persistActionLog(sum.Action, sum.Simulation, sel, candidates, sum.AffectedBytes, sum.Results, sum.ErrorSummary)
	m.status = fmt.Sprintf(
		"migrate-host: action=%s matched=%d affected=%s",
		sum.Action,
		sum.MatchedCount,
		formatBytesIEC(sum.AffectedBytes),
	)
	if !m.dryRun && sum.Succeeded > 0 {
		m.removeSessionsByID(candidates)
	}
}

func (m *tuiModel) requestRestore() {
	if m.source != "trash" {
		m.status = "Current source is sessions; use d to delete."
		return
	}
	selected, ok := m.selectedSession()
	if !ok {
		m.status = "No session selected."
		return
	}
	if !m.dryRun && !m.confirm {
		m.status = "Real restore requires --confirm."
		return
	}
	if !m.dryRun && !m.yes {
		m.pendingAction = "restore"
		m.pendingID = selected.SessionID
		m.status = fmt.Sprintf("Confirm restore %s: press y to continue, n to cancel.", shortID(selected.SessionID))
		return
	}
	m.runRestore(selected)
}

func (m *tuiModel) runRestore(selected session.Session) {
	sel := session.Selector{ID: selected.SessionID}
	sum, err := restoreexec.Execute(
		[]session.Session{selected},
		sel,
		restoreexec.Options{
			DryRun:            m.dryRun,
			Confirm:           m.confirm,
			Yes:               true,
			MaxBatch:          max(1, m.maxBatch),
			SessionsRoot:      m.sessionsRoot,
			TrashSessionsRoot: filepath.Join(m.trashRoot, "sessions"),
		},
	)
	m.pendingAction = ""
	m.pendingID = ""
	m.pendingHost = ""
	if err != nil {
		m.status = "restore failed: " + err.Error()
		return
	}
	m.persistActionLog(sum.Action, sum.Simulation, sel, []session.Session{selected}, sum.AffectedBytes, sum.Results, sum.ErrorSummary)
	m.status = fmt.Sprintf(
		"restore: action=%s matched=%d affected=%s",
		sum.Action,
		sum.MatchedCount,
		formatBytesIEC(sum.AffectedBytes),
	)
	if !m.dryRun && sum.Succeeded > 0 {
		m.removeSelectedSession()
	}
}

func (m *tuiModel) commitPendingAction() {
	if m.pendingAction == "" {
		return
	}
	selected, ok := m.selectedSession()
	if !ok || selected.SessionID != m.pendingID {
		m.pendingAction = ""
		m.pendingID = ""
		m.status = "Pending action cancelled: selection changed."
		return
	}
	switch m.pendingAction {
	case "delete":
		m.runDelete(selected)
	case "migrate-host":
		host := strings.TrimSpace(m.pendingHost)
		if host == "" {
			m.pendingAction = ""
			m.pendingID = ""
			m.status = "Pending action cancelled: host missing."
			return
		}
		m.runHostMigrate(host, m.migrateCandidatesForHost(host))
	case "restore":
		m.runRestore(selected)
	default:
		m.pendingAction = ""
		m.pendingID = ""
		m.pendingHost = ""
	}
}

func (m *tuiModel) cancelPendingAction() {
	if m.pendingAction == "" {
		return
	}
	m.pendingAction = ""
	m.pendingID = ""
	m.pendingHost = ""
	m.status = "Pending action cancelled."
}

func (m *tuiModel) persistActionLog(action string, simulation bool, sel session.Selector, items []session.Session, affected int64, results []session.DeleteResult, errSummary string) {
	if strings.TrimSpace(m.logFile) == "" {
		return
	}
	rec := audit.ActionRecord{
		Timestamp:     time.Now().UTC(),
		Action:        action,
		Simulation:    simulation,
		Selector:      sel,
		MatchedCount:  len(items),
		AffectedBytes: affected,
		Results:       results,
		ErrorSummary:  errSummary,
		Sessions:      make([]audit.SessionRef, 0, len(items)),
	}
	for _, s := range items {
		rec.Sessions = append(rec.Sessions, audit.SessionRef{SessionID: s.SessionID, Path: s.Path})
	}
	if err := audit.WriteActionLog(m.logFile, rec); err != nil {
		m.status = "action succeeded, log write failed: " + err.Error()
	}
}

func (m *tuiModel) removeSelectedSession() {
	if len(m.tree) == 0 || m.cursor < 0 || m.cursor >= len(m.tree) {
		return
	}
	item := m.tree[m.cursor]
	if item.kind != treeItemSession || item.index < 0 || item.index >= len(m.sessions) {
		return
	}
	m.sessions = append(m.sessions[:item.index], m.sessions[item.index+1:]...)
	m.rebuildTree()
	if len(m.tree) == 0 {
		m.cursor = 0
	}
	m.clampOffset()
}

func (m *tuiModel) removeSessionsByID(items []session.Session) {
	if len(items) == 0 || len(m.sessions) == 0 {
		return
	}
	ids := make(map[string]struct{}, len(items))
	for _, s := range items {
		ids[s.SessionID] = struct{}{}
	}
	filtered := m.sessions[:0]
	for _, s := range m.sessions {
		if _, drop := ids[s.SessionID]; drop {
			continue
		}
		filtered = append(filtered, s)
	}
	m.sessions = filtered
	m.rebuildTree()
	if len(m.tree) == 0 {
		m.cursor = 0
	}
	m.clampOffset()
}
