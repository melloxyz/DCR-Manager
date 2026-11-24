package backend

import (
	"dcr-manager/internal/app"
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s, e := readPreferences(path)
	if e != nil || s.OnboardingDone || s.Landing != "overview" {
		t.Fatal(s, e)
	}
	s.OnboardingDone = true
	s.Landing = "connection"
	if e = writePreferences(path, s); e != nil {
		t.Fatal(e)
	}
	got, e := readPreferences(path)
	if e != nil || got != s {
		t.Fatal(got, e)
	}
}
func TestSettingsInvalidPreserved(t *testing.T) {
	p := filepath.Join(t.TempDir(), "settings.json")
	raw := []byte("{invalid")
	if e := os.WriteFile(p, raw, 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := readPreferences(p); e == nil {
		t.Fatal("invalid accepted")
	}
	if e := writePreferences(p, app.DefaultPreferences()); e == nil {
		t.Fatal("overwrote invalid")
	}
	got, _ := os.ReadFile(p)
	if string(got) != string(raw) {
		t.Fatal("modified invalid settings")
	}
}
func TestStartupOwnership(t *testing.T) {
	if startupStatus("", false, "manager", "remote") != "off" || startupStatus("manager", true, "manager", "remote") != "manager" || startupStatus("remote", true, "manager", "remote") != "remote" || startupStatus("other", true, "manager", "remote") != "conflict" {
		t.Fatal("wrong startup ownership")
	}
}

func TestPreferencesReplaceAndRejectUnsupportedSchema(t *testing.T) {
	p := filepath.Join(t.TempDir(), "settings.json")
	s := app.DefaultPreferences()
	if e := writePreferences(p, s); e != nil {
		t.Fatal(e)
	}
	s.Landing = "connection"
	if e := writePreferences(p, s); e != nil {
		t.Fatal(e)
	}
	got, e := readPreferences(p)
	if e != nil || got != s {
		t.Fatal(got, e)
	}
	s.Schema = 2
	if e = writePreferences(p, s); e == nil {
		t.Fatal("unsupported schema accepted")
	}
	got, e = readPreferences(p)
	if e != nil || got.Schema != 1 {
		t.Fatal("original lost")
	}
}
