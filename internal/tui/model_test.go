package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"dcr-manager/internal/app"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

func TestNavigation(t *testing.T) {
	m := New(nil)
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if next.(Model).page != 1 {
		t.Fatal("down must select Connection")
	}
	next, _ = next.Update(tea.KeyPressMsg{Code: '9', Text: "9"})
	if next.(Model).page != 8 {
		t.Fatal("9 must select Settings")
	}
}
func TestDashboardLayout(t *testing.T) {
	for _, size := range [][2]int{{120, 36}, {90, 25}, {55, 18}, {30, 10}} {
		m := New(nil)
		m.width, m.height = size[0], size[1]
		m.data = app.Snapshot{Status: app.Status{Version: "0.3.4", Installed: true, RemoteRoots: 1, RemoteProcesses: 3, CacheCandidates: 1}}
		m.ready = true
		m.sync()
		v := m.View().Content
		for _, line := range strings.Split(v, "\n") {
			if lipgloss.Width(line) > size[0] {
				t.Fatalf("width %d exceeds %d", lipgloss.Width(line), size[0])
			}
		}
		if lipgloss.Height(v) > size[1] {
			t.Fatalf("height %d exceeds %d", lipgloss.Height(v), size[1])
		}
		if size[0] >= 90 && !strings.Contains(ansi.Strip(v), "System overview") {
			t.Fatal("missing dashboard")
		}
	}
}
func TestUnknownNotHealthy(t *testing.T) {
	m := New(nil)
	m.width = 120
	m.height = 36
	m.sync()
	if strings.Contains(ansi.Strip(m.View().Content), "Running") {
		t.Fatal("unknown state presented as running")
	}
}

func TestEveryPageFits(t *testing.T) {
	for page := range pages {
		for _, size := range [][2]int{{120, 36}, {90, 25}, {55, 18}, {42, 10}} {
			m := New(nil)
			m.width, m.height = size[0], size[1]
			m.ready = true
			m.data = app.Snapshot{Status: app.Status{Version: "0.3.4", RemoteProcesses: 3, RemoteRoots: 1, CacheCandidates: 1}, Installations: []app.Installation{{Kind: "dedicated", Version: "0.3.4"}}, Backups: []app.Backup{{Name: "package-backup-0123456789abcdef0123456789abcdef", AgeDays: 12}}, Logs: []app.Log{{Name: "2026-11-24-debug-0.log", Codes: []string{"EBUSY"}}}}
			m.selectPage(page)
			v := m.View().Content
			if lipgloss.Height(v) > size[1] {
				t.Fatalf("page %d height exceeds %d", page, size[1])
			}
			for _, line := range strings.Split(v, "\n") {
				if lipgloss.Width(line) > size[0] {
					t.Fatalf("page %d exceeds %d cells", page, size[0])
				}
			}
		}
	}
}
func TestSidebarClick(t *testing.T) {
	m := New(nil)
	next, _ := m.Update(tea.MouseClickMsg{X: 3, Y: 7, Button: tea.MouseLeft})
	if next.(Model).page != 1 {
		t.Fatal("click must select Connection")
	}
}
func TestTerminalControlSanitization(t *testing.T) {
	if safe("safe\x1b[31m\n\r") != "safe[31m" {
		t.Fatal("unsafe terminal controls")
	}
}
