package tui

import (
	"dcr-manager/internal/app"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

func TestTroubleshootingNavigationOnly(t *testing.T) {
	m := New(nil)
	m.ready = true
	m.data = app.Snapshot{Backups: []app.Backup{{Status: "rollback-incomplete"}}}
	m, c := actionKey(m, 'h')
	if c != nil || m.page != 7 || !m.troubleshooting {
		t.Fatal("help not opened")
	}
	if !strings.Contains(ansi.Strip(m.vp.GetContent()), "Unfinished package transaction") {
		t.Fatal("missing guidance")
	}
	m, c = actionKey(m, 'g')
	if c != nil || m.page != 5 || m.busy {
		t.Fatal("guidance must navigate only")
	}
}
func TestTroubleshootingStaleAndFilter(t *testing.T) {
	m := New(nil)
	m.ready = true
	m.data.Logs = []app.Log{{Name: "lock", Codes: []string{"EBUSY"}}, {Name: "network", Codes: []string{"ENOTFOUND"}}}
	m, _ = actionKey(m, 'h')
	m.err = "stale"
	m, c := actionKey(m, 'g')
	if c != nil || m.page != 7 {
		t.Fatal("stale recommendation navigated")
	}
	m.err = ""
	m, _ = actionKey(m, 'l')
	m, _ = actionKey(m, 'f')
	v := ansi.Strip(m.vp.GetContent())
	if !strings.Contains(v, "EBUSY") || strings.Contains(v, "ENOTFOUND") {
		t.Fatal("filter did not work")
	}
}

func TestTroubleshootingOpaqueFrame(t *testing.T) {
	for _, size := range [][2]int{{112, 28}, {55, 18}, {42, 10}} {
		m := New(nil)
		m.ready = true
		m.width, m.height = size[0], size[1]
		m.data = app.Snapshot{Backups: []app.Backup{{Status: "rollback-incomplete"}}}
		m, _ = actionKey(m, 'h')
		rows := strings.Split(m.View().Content, "\n")
		if len(rows) != size[1] {
			t.Fatal("incorrect frame height")
		}
		for _, r := range rows {
			assertOpaque(t, r)
		}
	}
}
