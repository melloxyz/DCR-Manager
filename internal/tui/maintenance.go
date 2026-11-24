package tui

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"dcr-manager/internal/app"
	"strings"
)

type maintenanceDone struct{ err error }

func (m Model) WithMaintainer(op app.Maintainer) Model { m.maintainer = op; m.sync(); return m }
func (m Model) maintenanceAllowed(a app.MaintenanceAction) bool {
	if m.width < 50 || m.height < 14 || m.maintainer == nil || !m.ready || m.loading || m.busy || m.err != "" || !m.data.ProcessControlReady {
		return false
	}
	s := m.data.Status
	if a == app.QuarantineCache {
		return s.NpmAvailable && s.CacheKnown && len(m.data.Caches) > 0 && len(m.data.Caches) == s.CacheCandidates
	}
	if a == app.RepairPackage && s.Version == "unknown" {
		return false
	}
	return s.NodeAvailable && s.NpmAvailable && s.RemoteProcesses == 0 && s.OtherDcrProcesses == 0
}
func (m Model) maintenanceKey(k string) (Model, tea.Cmd, bool) {
	if m.busy {
		return m, nil, false
	}
	if m.maintenancePending.Action != "" {
		switch k {
		case "n", "esc", "q", "ctrl+c":
			m.maintenancePending = app.MaintenanceRequest{}
			m.maintenanceNotice = "Cancelled. No changes made."
			m.sync()
		case "y":
			req := m.maintenancePending
			m.maintenancePending = app.MaintenanceRequest{}
			if !m.maintenanceAllowed(req.Action) {
				m.maintenanceNotice = "Action unavailable; refresh or resize."
				m.sync()
				return m, nil, true
			}
			m.busy = true
			m.maintenanceNotice = ""
			m.sync()
			op := m.maintainer
			return m, func() tea.Msg { return maintenanceDone{op(context.Background(), req)} }, true
		}
		return m, nil, true
	}
	if m.page == 4 && k == "c" && len(m.data.Caches) > 0 {
		m.cacheSelection = (m.cacheSelection + 1) % len(m.data.Caches)
		m.vp.GotoTop()
		m.sync()
		return m, nil, true
	}
	var a app.MaintenanceAction
	if m.page == 2 {
		switch k {
		case "p":
			a = app.RepairPackage
		case "u":
			a = app.UpdatePackage
		}
	}
	if m.page == 4 && k == "z" {
		a = app.QuarantineCache
	}
	if a == "" {
		return m, nil, false
	}
	if !m.maintenanceAllowed(a) {
		m.maintenanceNotice = "Action unavailable. Close DCR clients for package changes; refresh diagnostics."
		m.vp.GotoTop()
		m.sync()
		return m, nil, true
	}
	req := app.MaintenanceRequest{Action: a, Expected: m.data.Status.Version}
	if a == app.QuarantineCache {
		c := m.data.Caches[m.cacheSelection%len(m.data.Caches)]
		req.CacheID = c.ID
		req.Expected = c.Token
	}
	if a == app.RepairPackage && req.Expected == "unknown" {
		m.maintenanceNotice = "Version unknown. Use explicit update instead."
		m.sync()
		return m, nil, true
	}
	m.maintenancePending = req
	m.maintenanceNotice = ""
	m.vp.GotoTop()
	m.sync()
	return m, nil, true
}
func (m Model) maintenanceOverlay(w int) string {
	if m.busy {
		return "\n" + panel(w, colorText("Maintenance in progress...", "#E3B341")+"\n"+label("Staging or verifying files. Do not close this window."))
	}
	req := m.maintenancePending
	title, desc := "Repair package", "Reinstall the confirmed version: "+safe(req.Expected)
	switch req.Action {
	case app.RepairPackage:
		if req.Expected == "absent" {
			title, desc = "Install package", "Install latest DCR in the dedicated directory."
		}
	case app.UpdatePackage:
		title, desc = "Update package", "Install latest from npm; current: "+safe(req.Expected)
	case app.QuarantineCache:
		title, desc = "Quarantine cache", "Move only cache "+safe(req.CacheID)+" to a backup."
	}
	return "\n" + panel(w, colorText(title+"?", "#E3B341")+"\n"+label(desc)+"\n"+label("No global cache wipe. Backups are preserved.")+"\n"+bold("[Y] Confirm   [N / Esc] Cancel"))
}
func (m Model) maintenancePanel(w int) string {
	if m.maintainer == nil {
		return legacyNotice(w, "Package and cache maintenance.")
	}
	rows := []string{}
	if m.page == 2 {
		title := "Repair current version"
		if m.data.Status.Version == "absent" {
			title = "Install package"
		}
		for _, x := range []struct {
			key, title string
			a          app.MaintenanceAction
		}{{"P", title, app.RepairPackage}, {"U", "Update to latest", app.UpdatePackage}} {
			state := "Unavailable"
			if m.maintenanceAllowed(x.a) {
				state = "Available"
			}
			rows = append(rows, line(w-4, bold("["+x.key+"] "+x.title), label(state)))
		}
		if m.data.Status.RemoteProcesses+m.data.Status.OtherDcrProcesses > 0 {
			rows = append(rows, label("Stop Remote and DCR MCP clients before package changes."))
		}
	} else {
		for i, c := range m.data.Caches {
			prefix := "  "
			if i == m.cacheSelection%len(m.data.Caches) {
				prefix = "> "
			}
			rows = append(rows, line(w-4, bold(prefix+safe(c.ID)), label(versionLabel(c.Version))))
		}
		state := "Unavailable"
		if m.maintenanceAllowed(app.QuarantineCache) {
			state = "Available"
		}
		rows = append(rows, label("[C] Next candidate"), line(w-4, bold("[Z] Quarantine selected"), label(state)))
	}
	out := section("Maintenance") + panel(w, strings.Join(rows, "\n"))
	if m.maintenanceNotice != "" {
		out += "\n\n" + panel(w, safe(m.maintenanceNotice))
	}
	return out
}
