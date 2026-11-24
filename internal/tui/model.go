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

const Version = app.Version

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
	settings                    app.Settings
	settingsOperator            app.SettingsOperator
	settingsPending             app.SettingsRequest
	settingsNotice              string
	onboarding                  bool
	troubleshooting             bool
	findingSelection, logFilter int
	reader                      app.Reader
	operator                    app.RemoteOperator
	maintainer                  app.Maintainer
	maintenancePending          app.MaintenanceRequest
	maintenanceNotice           string
	cacheSelection              int
	backupSelection             int
	pending                     app.RemoteAction
	busy                        bool
	actionNotice                string
	data                        app.Snapshot
	ready, loading, focus       bool
	err                         string
	page, width, height         int
	vp                          viewport.Model
	updated                     time.Time
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
	if m.width < 76 || m.height < 18 {
		return 0
	}
	return 23
}
func (m Model) paneWidth() int { return max(1, m.width-m.sidebarWidth()-4) }
func (m *Model) sync() {
	m.vp.SetWidth(m.paneWidth())
	m.vp.SetHeight(max(1, m.height-3))
	m.vp.SetContent(m.content(m.paneWidth()))
	if m.confirming() {
		m.vp.GotoTop()
	}
}
func (m *Model) selectPage(p int) {
	next := (p + len(pages)) % len(pages)
	if next != m.page {
		m.maintenanceNotice = ""
	}
	m.page = next
	m.vp.GotoTop()
	m.sync()
}
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = v.Width, v.Height
		m.sync()
		return m, nil
	case settingsDone:
		m.busy = false
		m.settings = v.value
		if v.err != nil {
			m.settingsNotice = safe(v.err.Error())
		} else {
			m.settingsNotice = "Settings refreshed and verified."
			if v.kind == "preferences" && m.settings.Preferences.OnboardingDone {
				m.onboarding = false
			}
		}
		m.sync()
		return m, nil
	case maintenanceDone:
		m.busy = false
		m.maintenancePending = app.MaintenanceRequest{}
		if v.err != nil {
			m.maintenanceNotice = "Maintenance failed: " + safe(v.err.Error())
		} else {
			m.maintenanceNotice = "Maintenance verified. Backups preserved."
		}
		if m.reader != nil {
			m.loading = true
			m.sync()
			return m, m.load()
		}
		m.sync()
		return m, nil
	case actionResult:
		m.busy = false
		m.pending = ""
		if v.err != nil {
			m.actionNotice = "Operation failed: " + safe(v.err.Error())
		} else {
			m.actionNotice = "Local operation verified. Server connection not verified."
		}
		if m.reader != nil {
			m.loading = true
			m.sync()
			return m, m.load()
		}
		m.sync()
		return m, nil
	case resultMsg:
		m.loading = false
		if v.err != nil {
			m.err = v.err.Error()
		} else {
			m.data = v.data
			m.cacheSelection = 0
			m.backupSelection = 0
			m.findingSelection = 0
			m.ready = true
			m.updated = time.Now()
			m.err = ""
		}
		m.sync()
		return m, nil
	case tea.MouseClickMsg:
		if m.busy || m.pending != "" || m.maintenancePending.Action != "" || m.settingsPending.Kind != "" {
			return m, nil
		}
		mouse := v.Mouse()
		if m.sidebarWidth() > 0 && mouse.Button == tea.MouseLeft && mouse.X >= 0 && mouse.X < m.sidebarWidth() && mouse.Y >= 6 && mouse.Y < min(6+len(pages), m.height-1) {
			m.selectPage(mouse.Y - 6)
			return m, nil
		}
	case tea.KeyPressMsg:
		if next, cmd, handled := m.actionKey(v.String()); handled {
			return next, cmd
		}
		if next, cmd, handled := m.diagnosticKey(v.String()); handled {
			return next, cmd
		}
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
	if m.confirming() || m.busy {
		return m, nil
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
func panel(w int, content string) string  { return cardRows(w, content) }
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
func (m Model) confirming() bool {
	return m.pending != "" || m.maintenancePending.Action != "" || m.settingsPending.Kind != ""
}
func (m Model) confirmationFits() bool {
	if !m.confirming() || m.width < 50 || m.height < 14 {
		return false
	}
	return strings.Count(m.content(m.paneWidth()), "\n")+1 <= m.height-3
}
func (m Model) content(w int) string {
	if m.confirming() {
		out := bold("Confirm action") + "\n"
		switch {
		case m.pending != "":
			return out + m.actionPanel(w)
		case m.maintenancePending.Action != "":
			return out + m.maintenanceOverlay(w)
		default:
			return out + m.settingsContent(w)
		}
	}
	heading := []string{"System overview", "Connection", "Packages", "Processes", "Cache manager", "Recovery center", "MCP clients", "Diagnostic logs", "Settings"}[m.page]
	desc := []string{"Monitor your Desktop Commander environment.", "Local Remote process status and session inventory.", "Installed package versions and runtime requirements.", "Inspect the DCR process tree.", "Find exclusive Desktop Commander npm caches.", "Review preserved backups and recovery options.", "Read-only audit of known user configurations.", "Sanitized npm error history.", "Graphite Dark / workspace preferences."}[m.page]
	if m.page == 7 && m.troubleshooting {
		heading = "Troubleshooting"
		desc = "Local evidence and guided next steps."
	}
	out := label("Workspace / "+pages[m.page]) + "\n\n" + bold(heading) + "\n" + label(desc) + "\n"
	if m.loading {
		out += "\n" + colorText("Refreshing diagnostics…", "#E3B341") + "\n"
	}
	if m.err != "" {
		out += "\n" + colorText("Diagnostic failed: "+safe(m.err), "#E57373") + "\n"
	}
	if m.err != "" && m.page != 8 {
		return out + "\n" + label("Snapshot unavailable. Press R to retry.")
	}
	if m.loading && m.page != 8 && !m.busy {
		return out + "\n" + label("Waiting for a fresh snapshot.")
	}
	if !m.ready && m.page != 8 {
		return out + "\n" + label("No verified snapshot yet. Press R to retry.")
	}
	s := m.data.Status
	switch m.page {
	case 0:
		remote, rc := "Stopped", "#E3B341"
		if s.RemoteProcesses > 0 {
			remote, rc = "Running locally", "#66BB80"
		}
		version := versionLabel(s.Version)
		if s.Version == "" || s.Version == "absent" || s.Version == "unknown" {
			version = versionLabel(s.Version)
		}
		cacheValue := fmt.Sprintf("%02d", s.CacheCandidates)
		cacheDesc := "No candidates"
		if !s.CacheKnown {
			cacheValue = "Not assessed"
			cacheDesc = "Inventory unavailable"
		} else if s.CacheCandidates > 0 {
			cacheDesc = "Maintenance available"
		}
		cc := "#ECECEC"
		if s.CacheCandidates > 0 {
			cc = "#E3B341"
		}
		cards := [][4]string{{"Remote status", remote, fmt.Sprintf("%d local session(s)", s.RemoteRoots), rc}, {"Installed version", version, "Dedicated install", "#ECECEC"}, {"Active processes", fmt.Sprintf("%02d", s.RemoteProcesses), "Remote process count", "#ECECEC"}, {"Legacy cache", cacheValue, cacheDesc, cc}}
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
		out += "\n" + panel(w, bold("[H] Troubleshooting")+"\n"+label("Review findings and suggested next steps."))
		out += section("Quick navigation") + quickLinks(w) + "\n"
		if s.CacheCandidates > 0 {
			out += "\n" + colorText("Maintenance available", "#E3B341") + "  " + label("Legacy npm cache detected. Press 5 to inspect.")
		}
	case 1:
		out += m.connectionContent(w)
	case 2:
		out += m.packagesContent(w)
	case 3:
		out += m.processesContent(w)
	case 4:
		out += m.cacheContent(w)
	case 5:
		out += m.recoveryContent(w)
	case 6:
		out += m.clientsContent(w)
	case 7:
		out += m.logsContent(w)
	case 8:
		out += m.settingsContent(w)
	}
	return out
}
func RenderSnapshot(s app.Snapshot, w, h int) string { return RenderPageSnapshot(s, w, h, 0) }
func RenderPageSnapshot(s app.Snapshot, w, h, page int) string {
	return New(nil).RenderAt(s, w, h, page)
}

// RenderAt uses configured controls but never executes their commands.
func (m Model) RenderAt(s app.Snapshot, w, h, page int) string {
	m.ready = true
	m.loading = false
	m.err = ""
	m.data = s
	m.updated = time.Now()
	m.width = w
	m.height = h
	m.selectPage(page)
	return m.View().Content
}
