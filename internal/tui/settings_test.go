package tui

import (
	"dcr-manager/internal/app"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

func settingsFixture() Model {
	m := New(nil)
	m.ready = true
	m.data.Status = app.Status{Installed: true, NodeAvailable: true, NpmAvailable: true, Version: "0.3.4"}
	m.settings = app.Settings{Preferences: app.DefaultPreferences(), PreferencesKnown: true, Startup: "off"}
	m.selectPage(8)
	return m
}
func TestStartupConfirmAndCancel(t *testing.T) {
	m := settingsFixture()
	m.settingsOperator = func(app.SettingsRequest) (app.Settings, error) {
		t.Fatal("cancelled operation executed")
		return app.Settings{}, nil
	}
	m, c := actionKey(m, 'a')
	if c != nil || m.settingsPending.StartupAfter != "manager" {
		t.Fatal("startup confirmation missing")
	}
	m, c = actionKey(m, 'n')
	if c != nil || m.settingsPending.Kind != "" {
		t.Fatal("cancel failed")
	}
}
func TestStartupAsyncAndResize(t *testing.T) {
	m := settingsFixture()
	calls := 0
	m.settingsOperator = func(r app.SettingsRequest) (app.Settings, error) {
		calls++
		if r.StartupAfter != "manager" {
			t.Fatal(r)
		}
		s := m.settings
		s.Startup = "manager"
		return s, nil
	}
	m, _ = actionKey(m, 'a')
	m.width, m.height = 40, 8
	m, c := actionKey(m, 'y')
	if c != nil || calls != 0 {
		t.Fatal("hidden action")
	}
	m.width, m.height = 100, 30
	m, _ = actionKey(m, 'a')
	m, c = actionKey(m, 'y')
	if c == nil || calls != 0 || !m.busy {
		t.Fatal("not async")
	}
	n, _ := m.Update(c())
	if calls != 1 || n.(Model).settings.Startup != "manager" || n.(Model).busy {
		t.Fatal("result not applied")
	}
}
func TestOnboardingRoutesWithoutInstalling(t *testing.T) {
	m := settingsFixture()
	m.settingsOperator = func(app.SettingsRequest) (app.Settings, error) {
		t.Fatal("navigation mutated")
		return app.Settings{}, nil
	}
	m.data.Status.Installed = false
	m, _ = actionKey(m, 'o')
	if !m.onboarding {
		t.Fatal("wizard not open")
	}
	if !strings.Contains(ansi.Strip(m.vp.GetContent()), "Packages") {
		t.Fatal("missing guidance")
	}
	m, c := actionKey(m, 'i')
	if c != nil || m.page != 2 || m.busy {
		t.Fatal("install executed rather than navigation")
	}
}
func TestStartupConflictRefused(t *testing.T) {
	m := settingsFixture()
	m.settings.Startup = "conflict"
	m.settingsOperator = func(app.SettingsRequest) (app.Settings, error) {
		t.Fatal("conflict changed")
		return app.Settings{}, nil
	}
	m, c := actionKey(m, 'a')
	if c != nil || m.settingsPending.Kind != "" {
		t.Fatal("conflict accepted")
	}
}

func TestFirstRunAndFinish(t *testing.T) {
	s := app.Settings{Preferences: app.DefaultPreferences(), PreferencesKnown: true, Startup: "off"}
	calls := 0
	m := New(nil).WithSettings(s, func(r app.SettingsRequest) (app.Settings, error) {
		calls++
		if r.Kind != "preferences" || !r.After.OnboardingDone {
			t.Fatal(r)
		}
		s.Preferences = r.After
		return s, nil
	})
	if m.page != 8 || !m.onboarding || calls != 0 {
		t.Fatal("first run did not open read-only setup")
	}
	m, c := actionKey(m, 'f')
	if c == nil || calls != 0 {
		t.Fatal("finish not async")
	}
	n, _ := m.Update(c())
	m = n.(Model)
	if m.onboarding || !m.settings.Preferences.OnboardingDone || calls != 1 {
		t.Fatal("finish not persisted")
	}
}
func TestStartupDisableDirectly(t *testing.T) {
	m := settingsFixture()
	m.settings.Startup = "manager"
	m.settingsOperator = func(app.SettingsRequest) (app.Settings, error) { t.Fatal("unconfirmed"); return app.Settings{}, nil }
	m, c := actionKey(m, 'x')
	if c != nil || m.settingsPending.StartupAfter != "off" {
		t.Fatal("disable not directly available")
	}
}
