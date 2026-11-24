package tui

import (
	"context"
	"dcr-manager/internal/app"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

func maintenanceFixture(page int) Model {
	m := actionFixture()
	m.data.Status.RemoteProcesses = 0
	m.data.Status.RemoteRoots = 0
	m.data.Processes = nil
	m.data.Status.NpmAvailable = true
	m.data.Status.CacheKnown = true
	m.data.Status.CacheCandidates = 2
	m.data.Caches = []app.CacheCandidate{{ID: "0123456789abcdef", Version: "0.3.3"}, {ID: "abcdef0123456789", Version: "0.3.4"}}
	m.selectPage(page)
	return m
}
func TestPackageConfirmationAsync(t *testing.T) {
	m := maintenanceFixture(2)
	calls := 0
	m.maintainer = func(_ context.Context, r app.MaintenanceRequest) error {
		calls++
		if r.Action != app.RepairPackage || r.Expected != "0.3.4" {
			t.Fatal(r)
		}
		return nil
	}
	m, c := actionKey(m, 'p')
	if c != nil || m.maintenancePending.Action != app.RepairPackage {
		t.Fatal("missing confirmation")
	}
	if !strings.Contains(ansi.Strip(m.View().Content), "0.3.4") {
		t.Fatal("version missing")
	}
	m, c = actionKey(m, 'y')
	if c == nil || calls != 0 || !m.busy {
		t.Fatal("not asynchronous")
	}
	c()
	if calls != 1 {
		t.Fatal("no operation")
	}
}
func TestPackageBlockedWhileDCRActive(t *testing.T) {
	m := maintenanceFixture(2)
	m.data.Status.OtherDcrProcesses = 1
	m.maintainer = func(context.Context, app.MaintenanceRequest) error { t.Fatal("unsafe operation"); return nil }
	m, c := actionKey(m, 'u')
	if c != nil || m.maintenancePending.Action != "" {
		t.Fatal("active MCP allowed")
	}
}
func TestCacheSelectionAndCancellation(t *testing.T) {
	m := maintenanceFixture(4)
	m.maintainer = func(context.Context, app.MaintenanceRequest) error { t.Fatal("cancel failed"); return nil }
	m, _ = actionKey(m, 'c')
	m, c := actionKey(m, 'z')
	if c != nil || m.maintenancePending.CacheID != "abcdef0123456789" {
		t.Fatal("wrong cache selected")
	}
	m, c = actionKey(m, 'n')
	if c != nil || m.maintenancePending.Action != "" || m.busy {
		t.Fatal("not cancelled")
	}
}

func TestMaintenanceRequiresVisibleConfirmation(t *testing.T) {
	m := maintenanceFixture(2)
	m.maintainer = func(context.Context, app.MaintenanceRequest) error {
		t.Fatal("hidden confirmation executed")
		return nil
	}
	m, _ = actionKey(m, 'u')
	m.width, m.height = 40, 8
	m, c := actionKey(m, 'y')
	if c != nil || m.busy || m.maintenancePending.Action != "" {
		t.Fatal("hidden confirmation allowed")
	}
}
func TestCacheCountMismatchBlocksQuarantine(t *testing.T) {
	m := maintenanceFixture(4)
	m.data.Status.CacheCandidates = 3
	m.maintainer = func(context.Context, app.MaintenanceRequest) error { t.Fatal("ambiguous cache operation"); return nil }
	m, c := actionKey(m, 'z')
	if c != nil || m.maintenancePending.Action != "" {
		t.Fatal("ambiguous candidates allowed")
	}
}
func TestMaintenanceFailureRefreshesDiagnostics(t *testing.T) {
	m := maintenanceFixture(2)
	m.reader = func(context.Context) (app.Snapshot, error) { return app.Snapshot{}, nil }
	m.busy = true
	n, c := m.Update(maintenanceDone{err: context.DeadlineExceeded})
	next := n.(Model)
	if c == nil || next.busy || !next.loading || !strings.Contains(next.maintenanceNotice, "Maintenance failed") {
		t.Fatal("failed operation did not refresh")
	}
}
