package backend

import (
	"dcr-manager/internal/app"
	"testing"
)

func TestNativeProcessInventoryAndHealth(t *testing.T) {
	entry := "C:/Users/test/DesktopCommander/node_modules/@wonderwhy-er/desktop-commander/dist/index.js"
	rows := []nativeCandidate{
		{ID: app.ProcessIdentity{PID: 10, Created: 100}, Parent: 5, Args: []string{"node", entry, "remote"}},
		{ID: app.ProcessIdentity{PID: 11, Created: 110}, Parent: 10, Args: []string{"node", entry}},
		{ID: app.ProcessIdentity{PID: 12, Created: 90}, Parent: 5, Args: []string{"node", entry}},
		{ID: app.ProcessIdentity{PID: 13, Created: 120}, Parent: 99, Args: []string{"node", entry}},
		{ID: app.ProcessIdentity{PID: 14, Created: 130}, Parent: 10, Args: []string{"node", "C:/unrelated.js"}},
		{ID: app.ProcessIdentity{PID: 15, Created: 140}, Parent: 7, Args: []string{"node", entry}},
	}
	p, h, roots := nativeProcessInventory(rows, map[int]uint64{5: 95, 10: 100, 7: 0}, "C:/Users/test/DesktopCommander")
	if len(p) != 5 || roots != 1 || p[0].Role != "Remote" || p[1].Role != "Remote" || p[2].Role != "MCP" || p[0].Location != "Dedicated" {
		t.Fatal(p, roots)
	}
	if h[0].Assessment != "parent-present" || h[1].Assessment != "parent-present" || h[2].Assessment != "parent-pid-reused" || h[3].Assessment != "parent-missing" || h[4].Assessment != "parent-unknown" {
		t.Fatal(h)
	}
}
func TestNativeProcessParityFullSet(t *testing.T) {
	s := app.Snapshot{Status: app.Status{RemoteProcesses: 1, RemoteRoots: 1}, Processes: []app.Process{{PID: 10, Parent: 5, Role: "Remote", Location: "Dedicated"}}}
	p := []app.Process{{PID: 10, Parent: 5, Role: "Remote", Location: "Dedicated", Created: 100}}
	reconcileProcessInventory(&s, p, nil, 1, nil)
	if !s.ProcessControlReady || s.Processes[0].Created != 100 {
		t.Fatal(s)
	}
	p = append(p, app.Process{PID: 11, Parent: 5, Role: "MCP", Location: "Other", Created: 110})
	reconcileProcessInventory(&s, p, nil, 1, nil)
	if s.ProcessControlReady || s.Processes[0].Created != 0 {
		t.Fatal("unmatched MCP authorized")
	}
}
func TestNativeMaintenanceGuard(t *testing.T) {
	target := "C:/cache/_npx/1234abcd"
	entry := target + "/node_modules/@wonderwhy-er/desktop-commander/dist/index.js"
	row := nativeCandidate{ID: app.ProcessIdentity{PID: 10, Created: 100}, Args: []string{"node", entry}}
	if nativeGuard([]nativeCandidate{row}, app.UpdatePackage, "") == nil {
		t.Fatal("active package allowed")
	}
	if nativeGuard([]nativeCandidate{row}, app.QuarantineCache, target) == nil {
		t.Fatal("active cache allowed")
	}
	if nativeGuard([]nativeCandidate{row}, app.QuarantineCache, "C:/cache/_npx/abcdef01") != nil {
		t.Fatal("unrelated cache refused")
	}
	row.Args = nil
	if nativeGuard([]nativeCandidate{row}, app.RepairPackage, "") == nil {
		t.Fatal("uninspectable Node allowed")
	}
}
