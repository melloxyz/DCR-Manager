package tui

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"dcr-manager/internal/app"
	"fmt"
	"strings"
)

type actionResult struct {
	action app.RemoteAction
	err    error
}

func (m Model) WithOperator(op app.RemoteOperator) Model { m.operator = op; m.sync(); return m }
func (m Model) remotePIDs() []app.ProcessIdentity {
	var ids []app.ProcessIdentity
	for _, p := range m.data.Processes {
		if p.Role == "Remote" {
			ids = append(ids, app.ProcessIdentity{PID: p.PID, Created: p.Created})
		}
	}
	return ids
}
func (m Model) canOperate(a app.RemoteAction) bool {
	if m.width < 50 || m.height < 14 || m.operator == nil || !m.data.ProcessControlReady || !m.ready || m.loading || m.busy || m.err != "" {
		return false
	}
	s := m.data.Status
	if a == app.Start {
		return s.RemoteProcesses == 0 && s.Installed && s.NodeAvailable
	}
	if a == app.Stop || a == app.Restart {
		return s.RemoteProcesses > 0 && len(m.remotePIDs()) == s.RemoteProcesses && (a == app.Stop || s.Installed && s.NodeAvailable)
	}
	return false
}
func (m Model) actionCmd(a app.RemoteAction) tea.Cmd {
	ids := m.remotePIDs()
	op := m.operator
	return func() tea.Msg { return actionResult{a, op(context.Background(), a, ids)} }
}

// Handle confirmation before global navigation. I/O occurs only in tea.Cmd.
func (m Model) actionKey(k string) (Model, tea.Cmd, bool) {
	if m.busy {
		return m, nil, true
	}
	if m.pending != "" {
		switch k {
		case "n", "esc", "q", "ctrl+c":
			m.pending = ""
			m.actionNotice = "Cancelled. No changes made."
			m.sync()
		case "y":
			a := m.pending
			m.pending = ""
			if !m.canOperate(a) {
				m.actionNotice = "State changed. Refresh first."
				m.sync()
				return m, nil, true
			}
			m.busy = true
			m.actionNotice = ""
			m.sync()
			return m, m.actionCmd(a), true
		}
		return m, nil, true
	}
	if next, cmd, handled := m.maintenanceKey(k); handled {
		return next, cmd, true
	}
	if m.page != 1 {
		return m, nil, false
	}
	var a app.RemoteAction
	switch k {
	case "s":
		a = app.Start
	case "x":
		a = app.Stop
	case "t":
		a = app.Restart
	default:
		return m, nil, false
	}
	if !m.canOperate(a) {
		m.actionNotice = "Action unavailable. Check installation or refresh."
		m.sync()
		return m, nil, true
	}
	m.actionNotice = ""
	if a == app.Start {
		m.busy = true
		m.sync()
		return m, m.actionCmd(a), true
	}
	m.pending = a
	m.vp.GotoTop()
	m.sync()
	return m, nil, true
}
func (m Model) actionPanel(w int) string {
	if m.operator == nil {
		return legacyNotice(w, "Start, stop and restart Remote.")
	}
	if m.pending != "" {
		return "\n" + panel(w, colorText(strings.ToUpper(string(m.pending))+" REMOTE?", "#E3B341")+"\n"+label(fmt.Sprintf("This will disconnect active Remote tasks (%d processes).", m.data.Status.RemoteProcesses))+"\n"+bold("[Y] Confirm   [N / Esc] Cancel")) + "\n"
	}
	if m.busy {
		return "\n" + panel(w, colorText("Remote operation in progress...", "#E3B341")+"\n"+label("Wait for verification. Do not close this window.")) + "\n"
	}
	out := section("Remote actions")
	rows := []string{}
	for _, item := range []struct {
		a          app.RemoteAction
		key, title string
	}{{app.Start, "S", "Start Remote"}, {app.Stop, "X", "Stop Remote"}, {app.Restart, "T", "Restart Remote"}} {
		if m.canOperate(item.a) {
			rows = append(rows, line(w-4, bold("["+item.key+"] "+item.title), label("Available")))
		} else {
			rows = append(rows, line(w-4, label("["+item.key+"] "+item.title), label("Unavailable")))
		}
	}
	out += panel(w, strings.Join(rows, "\n"))
	if m.ready && !m.data.ProcessControlReady {
		out += "\n\n" + panel(w, colorText("Controls locked", "#E3B341")+"\n"+label(safe(m.data.ProcessControlIssue)))
	}
	if m.actionNotice != "" {
		out += "\n\n" + panel(w, safe(m.actionNotice))
	}
	return out
}
