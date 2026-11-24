package tui

import (
	tea "charm.land/bubbletea/v2"
	"dcr-manager/internal/app"
	"fmt"
	"strings"
)

var logGroups = []string{"all", "locks", "network"}

func (m Model) diagnosticKey(k string) (Model, tea.Cmd, bool) {
	if k == "h" {
		m.troubleshooting = true
		m.findingSelection = 0
		m.selectPage(7)
		return m, nil, true
	}
	if m.page != 7 {
		return m, nil, false
	}
	if k == "l" {
		m.troubleshooting = false
		m.vp.GotoTop()
		m.sync()
		return m, nil, true
	}
	if m.troubleshooting {
		findings := app.Diagnose(m.data)
		if k == "d" && len(findings) > 0 {
			m.findingSelection = (m.findingSelection + 1) % len(findings)
			m.vp.GotoTop()
			m.sync()
			return m, nil, true
		}
		if k == "g" && m.ready && !m.loading && m.err == "" && len(findings) > 0 {
			p := findings[m.findingSelection%len(findings)].Page
			m.troubleshooting = false
			m.selectPage(p)
			return m, nil, true
		}
	} else if k == "f" {
		m.logFilter = (m.logFilter + 1) % len(logGroups)
		m.vp.GotoTop()
		m.sync()
		return m, nil, true
	}
	return m, nil, false
}
func (m Model) troubleshootingContent(w int) string {
	if m.loading || m.err != "" {
		return emptyCard(w, "Snapshot not current", "Refresh successfully before using recommendations.")
	}
	items := app.Diagnose(m.data)
	out := "\n" + label("[D] Next finding  [G] Open suggested page  [L] Logs") + "\n"
	if len(items) == 0 {
		return out + emptyCard(w, "No findings in assessed checks", "Local checks cannot verify cloud connectivity.")
	}
	selected := items[m.findingSelection%len(items)]
	c := "#E3B341"
	if selected.Severity == "critical" {
		c = "#E57373"
	} else if selected.Severity == "info" {
		c = "#A0A0A0"
	}
	out += section(fmt.Sprintf("Finding %d / %d", m.findingSelection%len(items)+1, len(items))) + panel(w, colorText(strings.ToUpper(selected.Severity), c)+"\n"+bold(selected.Title)+"\n"+label(selected.Detail)+"\n\n"+bold("Suggested next step")+"\n"+label(selected.Next))
	out += "\n\n" + label("Navigation only. No repair or process action runs automatically.")
	return out
}
func (m Model) logsContent(w int) string {
	if m.troubleshooting {
		return m.troubleshootingContent(w)
	}
	group := logGroups[m.logFilter%len(logGroups)]
	rows := app.FilterLogs(m.data.Logs, group)
	out := "\n" + panel(w, line(w-4, bold("npm history"), fmt.Sprintf("%d / %d", len(rows), len(m.data.Logs)))+"\n"+label("[F] Filter: "+group+"   [H] Troubleshooting"))
	if !m.data.LogsKnown {
		out += "\n" + label("Scan unavailable or partial.")
	}
	if len(rows) == 0 {
		return out + emptyCard(w, "No matching retained entries", "No conclusion about current errors can be drawn.")
	}
	for _, l := range rows {
		var codes []string
		for _, c := range l.Codes {
			if app.CodeGroup(c) != "" {
				codes = append(codes, c)
			}
		}
		v := label("No recognized error code")
		if len(codes) > 0 {
			v = colorText(strings.Join(codes, ", "), "#E3B341")
		}
		out += "\n\n" + panel(w, label(safe(l.Name))+"\n"+label(safe(l.ModifiedUTC))+"\n"+v)
	}
	return out + "\n\n" + label("Historical npm events only. Up to 100 recent files / 20 matches; raw logs hidden.")
}
