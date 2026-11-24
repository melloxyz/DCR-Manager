package app

import "testing"

func TestLoginDecision(t *testing.T) {
	s := Snapshot{}
	if LoginDecision(s) != "inventory-unavailable" {
		t.Fatal("unknown inventory allowed")
	}
	s.ProcessControlReady = true
	if LoginDecision(s) != "prerequisites-missing" {
		t.Fatal("missing runtime allowed")
	}
	s.Status.NodeAvailable = true
	s.Status.Installed = true
	if LoginDecision(s) != "start" {
		t.Fatal("valid launch refused")
	}
	s.Status.RemoteProcesses = 1
	if LoginDecision(s) != "already-running" {
		t.Fatal("duplicate launch allowed")
	}
}
