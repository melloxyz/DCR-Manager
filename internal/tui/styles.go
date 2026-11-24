package tui

import (
	lg "charm.land/lipgloss/v2"
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"strconv"
	"strings"
)

// Explicitly paint every terminal cell. Nested SGR resets restore the local
// surface instead of the terminal's potentially translucent default.
func backgroundSequence(hex string) string {
	v, _ := strconv.ParseUint(strings.TrimPrefix(hex, "#"), 16, 32)
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", (v>>16)&255, (v>>8)&255, v&255)
}
func paintRow(s string, w int, hex string) string {
	w = max(0, w)
	s = ansi.Truncate(s, w, "")
	pad := max(0, w-ansi.StringWidth(s))
	base := backgroundSequence(hex)
	s = strings.NewReplacer("\x1b[0m", "\x1b[0m"+base, "\x1b[m", "\x1b[m"+base, "\x1b[49m", "\x1b[49m"+base).Replace(s)
	return base + s + base + strings.Repeat(" ", pad) + "\x1b[0m"
}
func paintBlock(s string, w, h int, hex string) []string {
	lines := strings.Split(s, "\n")
	rows := make([]string, max(0, h))
	for i := range rows {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		rows[i] = paintRow(line, w, hex)
	}
	return rows
}
func cardRows(w int, content string) string {
	if w < 6 {
		return paintRow(content, w, "#292929")
	}
	raw := lg.NewStyle().Width(w).Border(lg.RoundedBorder()).BorderForeground(border).Padding(0, 1).Foreground(fg).Render(content)
	rows := strings.Split(raw, "\n")
	for i := range rows {
		rows[i] = paintRow(rows[i], w, "#292929")
	}
	return strings.Join(rows, "\n")
}
func stateText(s string, ok bool) string {
	if ok {
		return colorText(s, "#66BB80")
	}
	return colorText(s, "#E3B341")
}
func packageName(kind string) string {
	switch kind {
	case "dedicated":
		return "Dedicated installation"
	case "npx":
		return "Legacy npm cache"
	case "global":
		return "Global npm installation"
	}
	return "Other installation"
}
func versionLabel(v string) string {
	if v == "" {
		return "Unknown"
	}
	return "v" + strings.TrimPrefix(safe(v), "v")
}

func quickLinks(w int) string {
	labels := []string{"[2] Connection", "[3] Packages", "[4] Processes", "[6] Recovery"}
	if w < 44 {
		return label(strings.Join(labels, "  "))
	}
	a := (w - 1) / 2
	b := w - a - 1
	return paintRow(" "+bold(labels[0]), a, "#2F2F2F") + " " + paintRow(" "+bold(labels[1]), b, "#2F2F2F") + "\n" +
		paintRow(" "+bold(labels[2]), a, "#2F2F2F") + " " + paintRow(" "+bold(labels[3]), b, "#2F2F2F")
}
