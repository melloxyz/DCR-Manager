package tui

import (
	tea "charm.land/bubbletea/v2"
	"dcr-manager/internal/app"
	"strings"
)

type settingsDone struct {
	value app.Settings
	err   error
	kind  string
}

func (m Model) WithSettings(s app.Settings, op app.SettingsOperator) Model {
	m.settings = s
	m.settingsOperator = op
	if s.PreferencesKnown && !s.Preferences.OnboardingDone {
		m.onboarding = true
		m.selectPage(8)
	} else if s.Preferences.Landing == "connection" {
		m.selectPage(1)
	}
	m.sync()
	return m
}
func (m Model) settingsCmd(r app.SettingsRequest) (Model, tea.Cmd, bool) {
	m.busy = true
	m.settingsNotice = ""
	m.sync()
	op := m.settingsOperator
	return m, func() tea.Msg { s, e := op(r); return settingsDone{s, e, r.Kind} }, true
}
func (m Model) settingsKey(k string) (Model, tea.Cmd, bool) {
	if m.page != 8 || m.settingsOperator == nil {
		return m, nil, false
	}
	if m.settingsPending.Kind != "" {
		switch k {
		case "n", "esc", "q", "ctrl+c":
			m.settingsPending = app.SettingsRequest{}
			m.settingsNotice = "Cancelled. No changes made."
			m.sync()
		case "y":
			r := m.settingsPending
			m.settingsPending = app.SettingsRequest{}
			if m.width < 50 || m.height < 14 || m.settings.Startup != r.StartupBefore {
				m.settingsNotice = "Change refused; resize or reload."
				m.sync()
				return m, nil, true
			}
			return m.settingsCmd(r)
		}
		return m, nil, true
	}
	switch k {
	case "e":
		return m.settingsCmd(app.SettingsRequest{Kind: "reload"})
	case "o":
		m.onboarding = !m.onboarding
		m.vp.GotoTop()
		m.sync()
		return m, nil, true
	case "i", "c":
		if !m.onboarding {
			return m, nil, false
		}
		if k == "i" {
			m.selectPage(2)
		} else {
			m.selectPage(1)
		}
		return m, nil, true
	case "f", "p":
		if k == "f" && !m.onboarding || k == "p" && m.onboarding {
			return m, nil, false
		}
		if !m.settings.PreferencesKnown || m.width < 50 || m.height < 14 {
			m.settingsNotice = "Settings unavailable; reload or resize."
			m.sync()
			return m, nil, true
		}
		r := app.SettingsRequest{Kind: "preferences", Before: m.settings.Preferences, After: m.settings.Preferences}
		if k == "f" {
			r.After.OnboardingDone = true
		} else if r.After.Landing == "overview" {
			r.After.Landing = "connection"
		} else {
			r.After.Landing = "overview"
		}
		return m.settingsCmd(r)
	case "a", "w", "x":
		if m.onboarding {
			return m, nil, false
		}
		if m.width < 50 || m.height < 14 {
			m.settingsNotice = "Resize before changing startup."
			m.sync()
			return m, nil, true
		}
		next := ""
		if m.settings.Startup == "off" || m.settings.Startup == "manager" || m.settings.Startup == "remote" {
			switch k {
			case "a":
				next = "manager"
			case "w":
				next = "remote"
			case "x":
				next = "off"
			}
		}
		if next != "" && next == m.settings.Startup {
			m.settingsNotice = "Startup already in requested state."
			m.sync()
			return m, nil, true
		}
		if next == "" {
			m.settingsNotice = "Startup registration unavailable or not owned. Reload or review Windows Startup Apps."
			m.sync()
			return m, nil, true
		}
		m.settingsPending = app.SettingsRequest{Kind: "startup", StartupBefore: m.settings.Startup, StartupAfter: next}
		m.settingsNotice = ""
		m.vp.GotoTop()
		m.sync()
		return m, nil, true
	}
	return m, nil, false
}
func startupLabel(v string) string {
	switch v {
	case "off":
		return "Not registered"
	case "manager":
		return "Manager registered"
	case "remote":
		return "Remote registered"
	case "conflict":
		return "Conflicting entry"
	default:
		return "Unknown"
	}
}
func (m Model) settingsContent(w int) string {
	if m.settingsOperator == nil {
		return "\n" + panel(w, bold("Graphite Dark")+"\n"+label("English / No telemetry"))
	}
	if m.busy {
		return "\n" + panel(w, colorText("Saving or reading settings...", "#E3B341"))
	}
	if m.settingsPending.Kind != "" {
		desc := "Open DCR Manager at Windows sign-in."
		if m.settingsPending.StartupAfter == "remote" {
			desc = "Attempt to start Remote at sign-in if not running."
		}
		if m.settingsPending.StartupAfter == "off" {
			desc = "Remove the owned startup entry."
		}
		return "\n" + panel(w, colorText("Change Windows startup?", "#E3B341")+"\n"+label(desc)+"\n"+label("Current user only. No installation or credentials change.")+"\n"+bold("[Y] Confirm   [N / Esc] Cancel"))
	}
	notice := ""
	if m.settingsNotice != "" {
		notice = "\n\n" + panel(w, safe(m.settingsNotice))
	}
	if m.onboarding {
		check := func(ok bool) string {
			if !m.ready || m.loading || m.err != "" {
				return label("Not assessed")
			}
			return stateText(map[bool]string{true: "Detected", false: "Missing"}[ok], ok)
		}
		s := m.data.Status
		return "\n" + panel(w, bold("Welcome to DCR Manager")+"\n"+label("1. Check prerequisites; install Node.js/npm manually if missing.")+"\n"+line(w-4, label("Node.js"), check(s.NodeAvailable))+"\n"+line(w-4, label("npm"), check(s.NpmAvailable))) +
			section("2. Set up Desktop Commander") + panel(w, line(w-4, label("Dedicated installation"), check(s.Installed))+"\n"+bold("[I] Open Packages")+"\n"+label("Install or repair with P, then confirm. No automatic installation.")) +
			section("3. Connect") + panel(w, bold("[C] Open Connection")+"\n"+label("Start Remote if needed; complete authentication in its console.")+"\n"+label("Check the official device portal for connectivity.")) +
			"\n\n" + label("[F] Finish / skip setup   [O] Settings   [R] Refresh") + notice
	}
	p := m.settings.Preferences
	setup := "Not completed"
	if p.OnboardingDone {
		setup = "Completed"
	}
	if !m.settings.PreferencesKnown {
		setup = "Unknown"
	}
	out := "\n" + panel(w, strings.Join([]string{line(w-4, label("Theme"), "Graphite Dark"), line(w-4, label("Language"), "English"), line(w-4, label("Telemetry"), "None"), line(w-4, label("Landing page [P]"), safe(p.Landing)), line(w-4, label("Setup [O]"), setup)}, "\n"))
	out += section("Windows startup") + panel(w, line(w-4, bold("Current user"), startupLabel(m.settings.Startup))+"\n"+label("[A] Manager   [W] Remote   [X] Disable")+"\n"+label("Registration does not prove Windows Startup Apps is enabled."))
	out += section("Local preferences") + panel(w, label("[E] Reload settings")+"\n"+label("Preferences are stored outside the project."))
	if !m.settings.PreferencesKnown {
		out += "\n" + label("Settings unreadable or unsupported; original file preserved.")
	}
	return out + notice
}
