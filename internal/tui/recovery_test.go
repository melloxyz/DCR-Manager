package tui

import (
	"context"
	"dcr-manager/internal/app"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

func recoveryFixture() Model {
	m := maintenanceFixture(5)
	m.data.RecoveryKnown = true
	m.data.Backups = []app.Backup{{Name: "package-backup-0123456789abcdef0123456789abcdef", Kind: "package", Version: "0.3.3", Status: "ready", Token: strings.Repeat("a", 64), Bytes: 1024, SizeKnown: true}, {Name: "cache-backup-0123456789abcdef0123456789abcdef", Kind: "cache", Version: "0.3.4", Status: "ready", Token: strings.Repeat("b", 64)}}
	m.sync()
	return m
}
func TestRecoverySelectConfirmAsync(t *testing.T) {
	m := recoveryFixture()
	calls := 0
	m.maintainer = func(_ context.Context, r app.MaintenanceRequest) error {
		calls++
		if r.Action != app.RestoreBackup || r.CacheID != m.data.Backups[1].Name || r.Expected != m.data.Backups[1].Token {
			t.Fatal(r)
		}
		return nil
	}
	m, _ = actionKey(m, 'b')
	m, c := actionKey(m, 'v')
	if c != nil || m.maintenancePending.Action != app.RestoreBackup {
		t.Fatal("missing confirmation")
	}
	if !strings.Contains(ansi.Strip(m.View().Content), "cache-backup") {
		t.Fatal("missing target")
	}
	m, c = actionKey(m, 'y')
	if c == nil || calls != 0 || !m.busy {
		t.Fatal("not async")
	}
	c()
	if calls != 1 {
		t.Fatal("not called")
	}
}
func TestRecoveryBlocksActiveAndInvalid(t *testing.T) {
	m := recoveryFixture()
	m.maintainer = func(context.Context, app.MaintenanceRequest) error { t.Fatal("unsafe"); return nil }
	m.data.Status.RemoteProcesses = 1
	m, c := actionKey(m, 'v')
	if c != nil || m.maintenancePending.Action != "" {
		t.Fatal("active restore allowed")
	}
	m.data.Status.RemoteProcesses = 0
	m.data.Backups[0].Status = "incomplete"
	m, c = actionKey(m, 'v')
	if c != nil || m.maintenancePending.Action != "" {
		t.Fatal("invalid backup allowed")
	}
}

func TestRecoveryResizeAndStaleSelection(t *testing.T) {
	m := recoveryFixture()
	m.maintainer = func(context.Context, app.MaintenanceRequest) error { t.Fatal("unsafe restore"); return nil }
	m, _ = actionKey(m, 'v')
	m.width, m.height = 40, 8
	m, c := actionKey(m, 'y')
	if c != nil || m.busy || m.maintenancePending.Action != "" {
		t.Fatal("hidden confirmation accepted")
	}
	m.width, m.height = 100, 30
	m, _ = actionKey(m, 'v')
	m.data.Backups[0].Token = strings.Repeat("c", 64)
	m, c = actionKey(m, 'y')
	if c != nil || m.busy {
		t.Fatal("changed target accepted")
	}
}
