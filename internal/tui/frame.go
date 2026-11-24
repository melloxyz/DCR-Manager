package tui

import (
	tea "charm.land/bubbletea/v2"
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"strings"
)

func (m Model) View() tea.View {
	w, h := max(1, m.width), max(1, m.height)
	if w < 42 || h < 10 {
		v := tea.NewView(strings.Join(paintBlock("Resize terminal / Q quit", w, h, "#212121"), "\n"))
		v.AltScreen = true
		v.BackgroundColor = bg
		v.ForegroundColor = fg
		v.WindowTitle = "DCR Manager"
		return v
	}
	sw := m.sidebarWidth()
	bodyH := h - 3
	status := "No snapshot"
	if m.ready {
		status = "Local snapshot " + m.updated.Format("15:04:05")
	}
	if m.loading {
		status = "Refreshing diagnostics..."
	}
	if m.err != "" {
		status = "Last refresh failed"
	}
	head := paintRow(line(w, " "+bold("DCR Manager")+"  "+label(Version), label("Windows  /  Preview ")), w, "#171717")
	sub := paintRow(line(w, " "+label(status), label("Local diagnostics / controls ")), w, "#212121")
	body := paintBlock(m.vp.View(), m.paneWidth(), bodyH, "#212121")
	nav := []string{bold("  WORKSPACE"), label("  Community toolkit"), "", label("  NAVIGATION")}
	for i, p := range pages {
		text := fmt.Sprintf("  %s  %s", glyphs[i], p)
		if i == m.page {
			text = " ▎" + glyphs[i] + "  " + p
			nav = append(nav, paintRow(bold(text), sw, "#303030"))
		} else {
			nav = append(nav, paintRow(label(text), sw, "#171717"))
		}
	}
	nav = append(nav, "", label("  1-9  Quick switch"))
	navRows := paintBlock(strings.Join(nav, "\n"), sw, bodyH, "#171717")
	rows := []string{head, sub}
	for i := 0; i < bodyH; i++ {
		side := ""
		if sw > 0 {
			side = navRows[i]
		}
		rows = append(rows, side+paintRow("", 2, "#212121")+body[i]+paintRow("", 2, "#212121"))
	}
	focus := "Navigation"
	if m.focus {
		focus = "Content"
	}
	hint := " Up/Down Navigate   Tab Focus   PgUp/PgDn Scroll   R Refresh   Q Quit"
	if w < 90 {
		hint = " 1-9 Pages  Tab Focus  R Refresh  Q Quit"
	}
	if m.pending != "" {
		hint = " Y Confirm   N/Esc Cancel"
	} else if m.busy {
		hint = " Remote operation in progress..."
	} else if m.page == 1 && m.operator != nil {
		hint = " S Start   X Stop   T Restart   R Refresh   Q Quit"
	}
	footer := paintRow(label(ansi.Truncate(hint, max(1, w-len(focus)-3), "…"))+"  "+label(focus), w, "#171717")
	rows = append(rows, footer)
	v := tea.NewView(strings.Join(rows, "\n"))
	v.AltScreen = true
	v.BackgroundColor = bg
	v.ForegroundColor = fg
	v.WindowTitle = "DCR Manager"
	v.MouseMode = tea.MouseModeCellMotion
	return v
}
func (m Model) packagesContent(w int) string {
	out := "\n" + bold("Detected installations") + "\n"
	for _, p := range m.data.Installations {
		status := stateText("Installed", p.EntryExists)
		if !p.EntryExists {
			status = colorText("Incomplete", "#E57373")
		} else if p.Kind == "npx" {
			status = colorText("Legacy", "#E3B341")
		}
		requirement := safe(p.NodeRange)
		if requirement == "" {
			requirement = "Not declared"
		}
		out += "\n" + panel(w, line(w-4, bold(packageName(p.Kind)), status)+"\n"+bold(versionLabel(p.Version))+"\n"+line(w-4, label("Node.js requirement"), requirement)) + "\n"
	}
	if len(m.data.Installations) == 0 {
		out += "\n" + panel(w, label("No identified installations.")) + "\n"
	}
	return out + section("Maintenance") + panel(w, label("Repair and update are available in the legacy manager.")+"\n"+label("Launch.cmd  /  package operations"))
}
