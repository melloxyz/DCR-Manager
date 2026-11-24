package app

import (
	"strings"
	"testing"
)

func TestTroubleshootingEvidence(t *testing.T) {
	s := Snapshot{Status: Status{NodeAvailable: true, NpmAvailable: true, Installed: true, CacheKnown: true, CacheCandidates: 1, RemoteProcesses: 3}, ProcessControlReady: true, RecoveryKnown: true, LogsKnown: true, Runtime: Runtime{Assessment: "meets-minimum"}, Logs: []Log{{Name: "safe", Codes: []string{"EBUSY", "EBUSY", "secret-token"}}}, Clients: []Client{{Status: "checked", Entries: 1, Floating: 1}}}
	f := Diagnose(s)
	ids := map[string]int{}
	for _, x := range f {
		ids[x.ID]++
		if strings.Contains(x.Detail, "secret-token") {
			t.Fatal("raw evidence leaked")
		}
	}
	for _, id := range []string{"cache", "lock-history", "floating"} {
		if ids[id] != 1 {
			t.Fatal("missing or duplicate", id, ids)
		}
	}
	if ids["remote-stopped"] != 0 {
		t.Fatal("false stopped")
	}
}
func TestTroubleshootingPriorities(t *testing.T) {
	s := Snapshot{Backups: []Backup{{Status: "rollback-incomplete"}}, RecoveryKnown: true}
	f := Diagnose(s)
	if len(f) == 0 || f[0].ID != "rollback" {
		t.Fatal("recovery should take priority")
	}
}
func TestLogFilter(t *testing.T) {
	logs := []Log{{Codes: []string{"EBUSY"}}, {Codes: []string{"ENOTFOUND"}}, {Codes: []string{"unknown"}}, {}}
	if len(FilterLogs(logs, "locks")) != 1 || len(FilterLogs(logs, "network")) != 1 || len(FilterLogs(logs, "all")) != 4 {
		t.Fatal("wrong filter")
	}
}
