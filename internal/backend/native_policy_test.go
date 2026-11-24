package backend

import (
	"dcr-manager/internal/app"
	"testing"
)

func TestNativeClassification(t *testing.T) {
	path := "C:/Users/test/DesktopCommander/node_modules/@wonderwhy-er/desktop-commander/dist/index.js"
	rows := []nativeCandidate{
		{ID: app.ProcessIdentity{PID: 1, Created: 100}, Args: []string{"node", "C:/Users/test/DesktopCommander/node_modules/.bin//../@wonderwhy-er/desktop-commander/dist/index.js", "remote"}},
		{ID: app.ProcessIdentity{PID: 2, Created: 101}, Parent: 1, Args: []string{"node", path}},
		{ID: app.ProcessIdentity{PID: 3, Created: 99}, Parent: 1, Args: []string{"node", path}},
		{ID: app.ProcessIdentity{PID: 4, Created: 102}, Parent: 1, Args: []string{"node", "C:/other.js", path, "remote"}},
		{ID: app.ProcessIdentity{PID: 5, Created: 103}, Args: []string{"node", path + "-fake", "remote"}},
		{ID: app.ProcessIdentity{PID: 6, Created: 104}, Args: []string{"node", path}},
		{ID: app.ProcessIdentity{PID: 7, Created: 105}, Args: []string{"node", path, "notremote"}},
	}
	r := classifyRemote(rows)
	if len(r) != 2 || r[0].ID.PID != 1 || r[1].ID.PID != 2 {
		t.Fatalf("wrong remote set: %+v", r)
	}
}
func TestIdentityRejectsPIDReuse(t *testing.T) {
	a := []app.ProcessIdentity{{PID: 1, Created: 100}}
	b := []app.ProcessIdentity{{PID: 1, Created: 101}}
	if sameIdentities(a, b) || sameIdentities(a, []app.ProcessIdentity{{PID: 1}}) {
		t.Fatal("reused PID accepted")
	}
	if !sameIdentities(a, a) {
		t.Fatal("valid identity rejected")
	}
}

func TestReconcileNativeFailsClosed(t *testing.T) {
	s := app.Snapshot{Status: app.Status{RemoteProcesses: 1}, Processes: []app.Process{{PID: 12, Role: "Remote"}}}
	rows := []nativeCandidate{{ID: app.ProcessIdentity{PID: 12, Created: 99}, Args: []string{"node", "C:/test/node_modules/@wonderwhy-er/desktop-commander/dist/index.js", "remote"}}}
	reconcileNative(&s, rows, nil)
	if !s.ProcessControlReady || s.Processes[0].Created != 99 {
		t.Fatal("valid inventory refused")
	}
	rows[0].ID.PID = 13
	reconcileNative(&s, rows, nil)
	if s.ProcessControlReady || s.Processes[0].Created != 0 {
		t.Fatal("mismatch did not clear authorization")
	}
}
