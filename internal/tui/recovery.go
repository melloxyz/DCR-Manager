package tui

import (
	"dcr-manager/internal/app"
	"fmt"
	"strings"
)

func (m Model) selectedBackup() (app.Backup, bool) {
	if len(m.data.Backups) == 0 {
		return app.Backup{}, false
	}
	return m.data.Backups[m.backupSelection%len(m.data.Backups)], true
}
func (m Model) recoveryAllowed() bool {
	b, ok := m.selectedBackup()
	s := m.data.Status
	return ok && m.data.RecoveryKnown && b.Status == "ready" && b.Token != "" && (b.Kind == "package" || b.Kind == "cache") && s.RemoteProcesses == 0 && s.OtherDcrProcesses == 0 && (b.Kind != "cache" || s.NpmAvailable && s.CacheKnown)
}
func recoverySize(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1f KiB", float64(n)/1024)
	}
	if n < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MiB", float64(n)/(1024*1024))
	}
	return fmt.Sprintf("%.2f GiB", float64(n)/(1024*1024*1024))
}
func (m Model) recoveryContent(w int) string {
	if m.maintenancePending.Action != "" || m.busy {
		return m.maintenanceOverlay(w)
	}
	if !m.data.RecoveryKnown {
		return emptyCard(w, "Recovery inventory unavailable", "Refresh before attempting recovery.")
	}
	var total int64
	unknown, review, attention := 0, 0, 0
	for _, b := range m.data.Backups {
		if b.SizeKnown {
			total += b.Bytes
		} else {
			unknown++
		}
		if b.Review {
			review++
		}
		if b.Status == "rollback-incomplete" || b.Status == "switching" || b.Status == "switch-failed" || b.Status == "unclassified-stage" {
			attention++
		}
	}
	size := recoverySize(total)
	if unknown > 0 {
		size += " + unmeasured"
	}
	out := "\n" + panel(w, line(w-4, bold("Recovery inventory"), label(fmt.Sprint(len(m.data.Backups))+" items"))+"\n"+line(w-4, label("Logical file size"), size)+"\n"+line(w-4, label("Retention review (30 days)"), fmt.Sprint(review))+"\n"+line(w-4, label("Stages requiring review"), fmt.Sprint(attention)))
	if b, ok := m.selectedBackup(); ok {
		state := "Unavailable"
		if m.maintenanceAllowed(app.RestoreBackup) {
			state = "Available"
		}
		sz := "Unknown"
		if b.SizeKnown {
			sz = recoverySize(b.Bytes)
		}
		out += section(fmt.Sprintf("Selected item %d / %d", m.backupSelection%len(m.data.Backups)+1, len(m.data.Backups))) + panel(w, bold(safe(b.Kind)+" / "+versionLabel(b.Version))+"\n"+label(safe(b.Name))+"\n"+line(w-4, label("Validation"), safe(b.Status))+"\n"+line(w-4, label(fmt.Sprintf("%d days old", b.AgeDays)), sz)+"\n"+line(w-4, bold("[V] Restore selected"), label(state))+"\n"+label("[B] Next item"))
		if b.Kind == "stage" {
			out += "\n" + label("Inspect transaction state; never auto-restore staging folders.")
		}
	} else {
		out += emptyCard(w, "No backups or stages", "Maintenance artifacts will appear here.")
	}
	if attention > 0 {
		out += "\n" + panel(w, colorText("Review transaction state", "#E3B341")+"\n"+label("Preserve stage and backup. See docs/RECOVERY.md."))
	}
	if m.maintenanceNotice != "" {
		out += "\n\n" + panel(w, safe(m.maintenanceNotice))
	}
	return out + "\n\n" + label("Retention is advisory; nothing is automatically deleted.")
}
func (m Model) recoveryConfirmation(w int) string {
	req := m.maintenancePending
	b, _ := m.selectedBackup()
	desc := "Restore " + versionLabel(b.Version) + " (current " + versionLabel(req.CurrentVersion) + ")."
	if b.Kind == "cache" {
		desc = "Restore cache to its original empty npm location."
	}
	return "\n" + panel(w, colorText("Restore backup?", "#E3B341")+"\n"+label(desc)+"\n"+label(safe(req.CacheID))+"\n"+bold("[Y] Confirm   [N / Esc] Cancel"))
}
func recoveryRequestMatches(r app.MaintenanceRequest, b app.Backup) bool {
	return r.CacheID == b.Name && r.Expected == b.Token && strings.TrimSpace(b.Token) != ""
}
