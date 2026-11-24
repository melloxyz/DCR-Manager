package tui

import (
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	lg "charm.land/lipgloss/v2"
	"context"
	"dcr-manager/internal/app"
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"time"
)

const Version = "1.3.0-dev"

var pages = []string{"Overview", "Connection", "Packages", "Processes", "Cache", "Recovery", "MCP Clients", "Logs", "Settings"}
var glyphs = []string{"▦", "↔", "□", "≡", "▤", "◇", "⊞", "≔", "⚙"}
var bg = lg.Color("#212121")
var sidebar = lg.Color("#171717")
var surface = lg.Color("#292929")
var border = lg.Color("#383838")
var fg = lg.Color("#ECECEC")
var muted = lg.Color("#A0A0A0")
var green = lg.Color("#66BB80")
var yellow = lg.Color("#E3B341")
var red = lg.Color("#E57373")

type Model struct {
	reader                app.Reader
	data                  app.Snapshot
	ready, loading, focus bool
	err                   string
	page, width, height   int
	vp                    viewport.Model
	updated               time.Time
}
type resultMsg struct {
	data app.Snapshot
	err  error
}

func New(reader app.Reader) Model {
	m := Model{reader: reader, width: 100, height: 30, vp: viewport.New(), loading: reader != nil}
	m.sync()
	return m
}
func (m Model) load() tea.Cmd {
	if m.reader == nil {
		return nil
	}
	return func() tea.Msg { s, e := m.reader(context.Background()); return resultMsg{s, e} }
}
func (m Model) Init() tea.Cmd { return m.load() }
func (m Model) sidebarWidth() int {
	if m.width < 76 {
		return 0
	}
	return 23
}
func (m Model) paneWidth() int { return max(1, m.width-m.sidebarWidth()-2) }
func (m *Model) sync() {
	m.vp.SetWidth(m.paneWidth())
	m.vp.SetHeight(max(1, m.height-4))
	m.vp.SetContent(m.content(m.paneWidth()))
}
func (m *Model) selectPage(p int) { m.page = (p + len(pages)) % len(pages); m.vp.GotoTop(); m.sync() }
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = v.Width, v.Height
		m.sync()
		return m, nil
	case resultMsg:
		m.loading = false
		if v.err != nil {
			m.err = v.err.Error()
		} else {
			m.data = v.data
			m.ready = true
			m.updated = time.Now()
			m.err = ""
		}
		m.sync()
		return m, nil
	case tea.MouseClickMsg:
		mouse := v.Mouse()
		if m.sidebarWidth() > 0 && mouse.X < m.sidebarWidth() && mouse.Y >= 6 && mouse.Y < 6+len(pages) {
			m.selectPage(mouse.Y - 6)
			return m, nil
		}
	case tea.KeyPressMsg:
		switch v.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			if !m.loading && m.reader != nil {
				m.loading = true
				m.sync()
				return m, m.load()
			}
		case "tab":
			m.focus = !m.focus
			return m, nil
		case "esc":
			m.focus = false
			return m, nil
		case "left":
			m.focus = false
			m.selectPage(m.page - 1)
			return m, nil
		case "right":
			m.focus = false
			m.selectPage(m.page + 1)
			return m, nil
		case "up", "k":
			if !m.focus {
				m.selectPage(m.page - 1)
				return m, nil
			}
		case "down", "j":
			if !m.focus {
				m.selectPage(m.page + 1)
				return m, nil
			}
		case "enter":
			m.focus = true
			return m, nil
		default:
			if len(v.String()) == 1 && v.String()[0] >= '1' && v.String()[0] <= '9' {
				m.selectPage(int(v.String()[0] - '1'))
				return m, nil
			}
		}
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}
func label(s string) string { return lg.NewStyle().Foreground(muted).Render(s) }
func bold(s string) string  { return lg.NewStyle().Foreground(fg).Bold(true).Render(s) }
func safe(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 32 || r == 127 || r >= 128 && r <= 159 {
			return -1
		}
		return r
	}, s)
}
func colorText(s string, c string) string { return lg.NewStyle().Foreground(lg.Color(c)).Render(s) }
func panel(w int, content string) string {
	if w < 6 {
		return ansi.Truncate(content, w, "")
	}
	return lg.NewStyle().Width(w).Border(lg.RoundedBorder()).BorderForeground(border).Padding(0, 1).Background(surface).Foreground(fg).Render(content)
}
func line(w int, left, right string) string {
	right = ansi.Truncate(right, max(1, w/2), "…")
	left = ansi.Truncate(left, max(1, w-lg.Width(right)-1), "…")
	return left + strings.Repeat(" ", max(1, w-lg.Width(left)-lg.Width(right))) + right
}
func section(title string) string { return "\n" + bold(title) + "\n" }
func metric(w int, title, val, desc, c string) string {
	inner := max(1, w-4)
	return panel(w, label(ansi.Truncate(title, inner, "…"))+"\n"+lg.NewStyle().Bold(true).Foreground(lg.Color(c)).Render(ansi.Truncate(val, inner, "…"))+"\n"+label(ansi.Truncate(desc, inner, "…")))
}
func (m Model) content(w int) string {
	heading := []string{"System overview", "Connection", "Packages", "Processes", "Cache manager", "Recovery center", "MCP clients", "Diagnostic logs", "Settings"}[m.page]
	desc := []string{"Monitor your Desktop Commander environment.", "Local Remote process status and session inventory.", "Installed package versions and runtime requirements.", "Inspect the DCR process tree.", "Find exclusive Desktop Commander npm caches.", "Review preserved backups and recovery options.", "Read-only audit of known user configurations.", "Sanitized npm error history.", "Graphite Dark / workspace preferences."}[m.page]
	out := label("Workspace / "+pages[m.page]) + "\n\n" + bold(heading) + "\n" + label(desc) + "\n"
	if m.loading {
		out += "\n" + colorText("Refreshing diagnostics…", "#E3B341") + "\n"
	}
	if m.err != "" {
		out += "\n" + colorText("Diagnostic failed: "+safe(m.err), "#E57373") + "\n"
	}
	if !m.ready && m.page != 8 {
		return out + "\n" + label("No verified snapshot yet. Press R to retry.")
	}
	s := m.data.Status
	switch m.page {
	case 0:
		remote, rc := "Stopped", "#E3B341"
		if s.RemoteProcesses > 0 {
			remote, rc = "Running", "#66BB80"
		}
		version := safe(s.Version)
		if version == "" {
			version = "Unknown"
		}
		cacheDesc := "No candidates"
		if !s.CacheKnown {
			cacheDesc = "Cache unavailable"
		} else if s.CacheCandidates > 0 {
			cacheDesc = "Maintenance available"
		}
		cc := "#ECECEC"
		if s.CacheCandidates > 0 {
			cc = "#E3B341"
		}
		cards := [][4]string{{"Remote status", remote, fmt.Sprintf("%d local session(s)", s.RemoteRoots), rc}, {"Installed version", version, "Dedicated install", "#ECECEC"}, {"Active processes", fmt.Sprintf("%02d", s.RemoteProcesses), "Remote process count", "#ECECEC"}, {"Legacy cache", fmt.Sprintf("%02d", s.CacheCandidates), cacheDesc, cc}}
		out += "\n"
		if w >= 48 {
			a := (w - 1) / 2
			b := w - a - 1
			for i := 0; i < 4; i += 2 {
				out += lg.JoinHorizontal(lg.Top, metric(a, cards[i][0], cards[i][1], cards[i][2], cards[i][3]), " ", metric(b, cards[i+1][0], cards[i+1][1], cards[i+1][2], cards[i+1][3])) + "\n"
			}
		} else {
			for _, c := range cards {
				out += metric(w, c[0], c[1], c[2], c[3]) + "\n"
			}
		}
		health := "Not assessed"
		if len(m.data.Health) > 0 {
			health = "Parents present"
			for _, p := range m.data.Health {
				if p.Assessment != "parent-present" {
					health = "Review required"
				}
			}
		}
		install := "Not detected"
		if s.Installed {
			install = "Detected"
		}
		out += section("System health") + panel(w, strings.Join([]string{line(w-4, label("Node.js"), safe(m.data.Runtime.Node)), line(w-4, label("npm"), safe(m.data.Runtime.Npm)), line(w-4, label("Process tree"), health), line(w-4, label("Installation"), install)}, "\n")) + "\n"
		out += section("Quick navigation") + label("[2] Connection   [3] Packages   [4] Processes   [6] Recovery") + "\n"
		if s.CacheCandidates > 0 {
			out += "\n" + colorText("Maintenance available", "#E3B341") + "  " + label("Legacy npm cache detected. Press 5 to inspect.")
		}
	case 1:
		state := "Stopped"
		if s.RemoteProcesses > 0 {
			state = "Running locally"
		}
		out += "\n" + panel(w, bold(state)+"\n"+label(fmt.Sprintf("%d session(s) · %d processes", s.RemoteRoots, s.RemoteProcesses))) + "\n\n" + label("Local activity does not prove server connectivity.") + "\n" + label("Start / stop / restart: use Launch.cmd (legacy manager).")
	case 2:
		for _, p := range m.data.Installations {
			out += "\n" + panel(w, bold(safe(p.Kind))+"  "+safe(p.Version)+"\n"+label("Node requirement: "+safe(p.NodeRange))+"\n"+label(fmt.Sprintf("Entry present: %t", p.EntryExists)))
		}
		if len(m.data.Installations) == 0 {
			out += "\n" + label("No identified installations.")
		}
		out += "\n\n" + label("Repair / update: use Launch.cmd (legacy manager).")
	case 3:
		out += "\n" + panel(w, line(w-4, bold("PID / role"), bold("Parent / location")))
		for _, p := range m.data.Processes {
			out += "\n" + panel(w, line(w-4, fmt.Sprintf("%d  %s", p.PID, safe(p.Role)), fmt.Sprintf("%d  %s", p.Parent, safe(p.Location))))
		}
		out += section("Parent assessment")
		for _, p := range m.data.Health {
			out += line(w, fmt.Sprint(p.PID), safe(p.Assessment)) + "\n"
		}
	case 4:
		out += "\n" + panel(w, bold(fmt.Sprintf("%d exclusive candidate(s)", s.CacheCandidates))+"\n"+label(fmt.Sprintf("npm cache path known: %t", s.CacheKnown)))
		for _, p := range m.data.Installations {
			if p.Kind == "npx" {
				out += "\n" + panel(w, "npx  "+safe(p.Version)+"\n"+label("Legacy package candidate"))
			}
		}
		out += "\n\n" + label("Quarantine / restore: use Launch.cmd (legacy manager).")
	case 5:
		if len(m.data.Backups) == 0 {
			out += "\n" + panel(w, label("No backups detected."))
		}
		for _, p := range m.data.Backups {
			out += "\n" + panel(w, safe(p.Name)+"\n"+label(fmt.Sprintf("%d days old · review: %t", p.AgeDays, p.Review)))
		}
		out += "\n\n" + label("Restoration and rollback: use Launch.cmd (legacy manager).")
	case 6:
		for _, p := range m.data.Clients {
			out += "\n" + panel(w, line(w-4, bold(safe(p.Client)), safe(p.Status))+"\n"+label(fmt.Sprintf("%d direct reference(s) · %d floating", p.Entries, p.Floating)))
		}
		if len(m.data.Clients) == 0 {
			out += "\n" + label("No known client configurations located.")
		}
	case 7:
		for _, p := range m.data.Logs {
			codes := strings.Join(p.Codes, ", ")
			c := "#E3B341"
			if codes == "" {
				codes = "No known error code"
				c = "#A0A0A0"
			}
			out += "\n" + panel(w, label(safe(p.Name))+"\n"+colorText(safe(codes), c))
		}
		if len(m.data.Logs) == 0 {
			out += "\n" + label("No matching npm logs found.")
		}
	case 8:
		out += "\n" + panel(w, strings.Join([]string{line(w-4, label("Theme"), "Graphite Dark"), line(w-4, label("Language"), "English"), line(w-4, label("Telemetry"), "None"), line(w-4, label("Backend"), "PowerShell bridge"), line(w-4, label("Build"), Version)}, "\n"))
		out += "\n\n" + label("Preferences are read-only in this development build.")
	}
	return out
}
func (m Model) View() tea.View {
	w, h := max(1, m.width), max(1, m.height)
	if w < 42 || h < 10 {
		v := tea.NewView(ansi.Truncate("Resize terminal · Q quit", w, ""))
		v.AltScreen = true
		return v
	}
	header := lg.NewStyle().Background(sidebar).Foreground(fg).Width(w).Render(line(w, " DCR Manager  /  "+Version, "Windows "))
	status := "No snapshot"
	if m.ready {
		status = "Local snapshot " + m.updated.Format("15:04:05")
	}
	if m.loading {
		status = "Refreshing…"
	}
	if m.err != "" {
		status = "Last refresh failed"
	}
	footer := lg.NewStyle().Background(sidebar).Foreground(muted).Width(w).Render(ansi.Truncate(" ↑↓ Navigate  Tab Focus  PgUp/PgDn Scroll  R Refresh  Q Quit", w, "…"))
	sub := lg.NewStyle().Foreground(muted).Width(w).Render(ansi.Truncate(" "+status+" · local diagnostics only", w, "…"))
	content := m.vp.View()
	content = lg.NewStyle().Width(m.paneWidth()).Height(max(1, h-4)).Render(content)
	if sw := m.sidebarWidth(); sw > 0 {
		nav := []string{bold(" DCR Manager"), label(" Community toolkit"), "", label(" WORKSPACE")}
		for i, p := range pages {
			style := lg.NewStyle().Foreground(muted).Width(sw)
			if i == m.page {
				style = style.Background(lg.Color("#303030")).Foreground(fg).Bold(true)
			}
			nav = append(nav, style.Render(fmt.Sprintf(" %s  %s", glyphs[i], p)))
		}
		nav = append(nav, "", label(" 1–9  Quick switch"))
		n := strings.Join(nav, "\n")
		lines := strings.Split(n, "\n")
		if len(lines) > h-4 {
			lines = lines[:h-4]
		}
		side := lg.NewStyle().Background(sidebar).Width(sw).Height(max(1, h-4)).Render(strings.Join(lines, "\n"))
		content = lg.JoinHorizontal(lg.Top, side, " ", content)
	}
	full := header + "\n" + sub + "\n" + content + "\n" + footer
	lines := strings.Split(full, "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], w, "")
	}
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen = true
	v.BackgroundColor = bg
	v.ForegroundColor = fg
	v.WindowTitle = "DCR Manager"
	v.MouseMode = tea.MouseModeCellMotion
	return v
}
func RenderSnapshot(s app.Snapshot, w, h int) string {
	m := New(nil)
	m.ready = true
	m.data = s
	m.updated = time.Now()
	m.width = w
	m.height = h
	m.sync()
	return m.View().Content
}
