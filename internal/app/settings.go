package app

type Preferences struct {
	Schema         int
	OnboardingDone bool
	Landing        string
}

func DefaultPreferences() Preferences { return Preferences{Schema: 1, Landing: "overview"} }

type Settings struct {
	Preferences      Preferences
	PreferencesKnown bool
	Startup          string
}
type SettingsRequest struct {
	Kind                        string
	Before, After               Preferences
	StartupBefore, StartupAfter string
}
type SettingsOperator func(SettingsRequest) (Settings, error)

func LoginDecision(s Snapshot) string {
	if !s.ProcessControlReady {
		return "inventory-unavailable"
	}
	if s.Status.RemoteProcesses > 0 {
		return "already-running"
	}
	if !s.Status.Installed || !s.Status.NodeAvailable {
		return "prerequisites-missing"
	}
	return "start"
}
