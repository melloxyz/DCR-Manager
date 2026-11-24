package tui

import (
	"fmt"
	"strings"
)

func emptyCard(w int, title, desc string) string {
	return "\n" + panel(w, bold(title)+"\n"+label(desc)) + "\n"
}
func legacyNotice(w int, task string) string {
	return section("Maintenance") + panel(w, label(task)+"\n"+label("Available in Launch.cmd (legacy manager)."))
}
func (m Model) connectionContent(w int) string {
	if m.pending != "" || m.busy {
		return m.actionPanel(w)
	}
	s := m.data.Status
	state, c := "Stopped", "#E3B341"
	if s.RemoteProcesses > 0 {
		state, c = "Running locally", "#66BB80"
	}
	return "\n" + panel(w, line(w-4, bold("Remote service"), colorText(state, c))+"\n"+line(w-4, label("Local sessions"), fmt.Sprint(s.RemoteRoots))+"\n"+line(w-4, label("Remote processes"), fmt.Sprint(s.RemoteProcesses))) + "\n\n" + panel(w, bold("Connectivity")+"\n"+label("Local activity does not confirm the server connection.")) + m.actionPanel(w)
}
func (m Model) processesContent(w int) string {
	if len(m.data.Processes) == 0 {
		return emptyCard(w, "No DCR processes", "Refresh when Desktop Commander is running.")
	}
	rows := []string{line(w-4, bold("PID / role"), bold("Parent / source"))}
	for _, p := range m.data.Processes {
		rows = append(rows, line(w-4, fmt.Sprintf("%d  %s", p.PID, safe(p.Role)), fmt.Sprintf("%d  %s", p.Parent, safe(p.Location))))
	}
	out := "\n" + panel(w, strings.Join(rows, "\n"))
	if len(m.data.Health) > 0 {
		rows = []string{}
		for _, p := range m.data.Health {
			value := stateText("Parent present", true)
			if p.Assessment != "parent-present" {
				value = colorText("Review required", "#E3B341")
			}
			rows = append(rows, line(w-4, fmt.Sprintf("PID %d", p.PID), value))
		}
		out += section("Process health") + panel(w, strings.Join(rows, "\n"))
	}
	return out
}
func (m Model) cacheContent(w int) string {
	s := m.data.Status
	state, c := "No candidates", "#66BB80"
	if !s.CacheKnown {
		state, c = "Not assessed", "#E3B341"
	} else if s.CacheCandidates > 0 {
		state, c = "Review available", "#E3B341"
	}
	out := "\n" + panel(w, line(w-4, bold("npm cache"), colorText(state, c))+"\n"+line(w-4, label("Exclusive candidates"), fmt.Sprint(s.CacheCandidates)))
	for _, p := range m.data.Installations {
		if p.Kind == "npx" {
			out += "\n\n" + panel(w, line(w-4, bold("Legacy npm cache"), colorText("Legacy", "#E3B341"))+"\n"+bold(versionLabel(p.Version))+"\n"+label("Dedicated DCR cache candidate"))
		}
	}
	return out + legacyNotice(w, "Inspect, quarantine or restore an eligible cache.")
}
func (m Model) recoveryContent(w int) string {
	out := ""
	if len(m.data.Backups) == 0 {
		out += emptyCard(w, "No backups detected", "Package and cache backups will appear here.")
	}
	for _, p := range m.data.Backups {
		review := label("Retained")
		if p.Review {
			review = colorText("Review", "#E3B341")
		}
		out += "\n" + panel(w, line(w-4, bold("Backup"), review)+"\n"+safe(p.Name)+"\n"+label(fmt.Sprintf("%d days old", p.AgeDays))) + "\n"
	}
	return out + legacyNotice(w, "Restore validated backups and review rollback options.")
}
func (m Model) clientsContent(w int) string {
	if len(m.data.Clients) == 0 {
		return emptyCard(w, "No known client configurations", "Custom locations are not automatically scanned.")
	}
	rows := []string{}
	for _, p := range m.data.Clients {
		status := stateText("Checked", true)
		if p.Status != "checked" {
			status = colorText("Not assessed", "#E3B341")
		}
		rows = append(rows, line(w-4, bold(safe(p.Client)), status), label(fmt.Sprintf("%d direct reference(s) / %d floating version(s)", p.Entries, p.Floating)))
		rows = append(rows, "")
	}
	return "\n" + panel(w, strings.Join(rows[:len(rows)-1], "\n")) + "\n\n" + label("Read-only audit. Plugins and custom locations may not be detected.")
}
func (m Model) logsContent(w int) string {
	if len(m.data.Logs) == 0 {
		return emptyCard(w, "No matching npm logs", "Only retained, matching logs are displayed.")
	}
	rows := []string{}
	for _, p := range m.data.Logs {
		codes := strings.Join(p.Codes, ", ")
		v := label("No known error code")
		if codes != "" {
			v = colorText(safe(codes), "#E57373")
		}
		rows = append(rows, label(safe(p.Name)), v, "")
	}
	return "\n" + panel(w, strings.Join(rows[:len(rows)-1], "\n"))
}
