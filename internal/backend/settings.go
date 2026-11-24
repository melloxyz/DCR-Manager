package backend

import (
	"dcr-manager/internal/app"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func preferencesPath() (string, error) {
	dir, e := os.UserConfigDir()
	if e != nil {
		return "", e
	}
	return filepath.Join(dir, "DCRManager", "settings.json"), nil
}
func validPreferences(s app.Preferences) bool {
	return s.Schema == 1 && (s.Landing == "overview" || s.Landing == "connection")
}
func readPreferences(p string) (app.Preferences, error) {
	s := app.DefaultPreferences()
	info, e := os.Lstat(p)
	if os.IsNotExist(e) {
		return s, nil
	}
	if e != nil {
		return s, e
	}
	if !info.Mode().IsRegular() || info.Size() > 4096 {
		return s, errors.New("unsupported settings file")
	}
	f, e := os.Open(p)
	if e != nil {
		return s, e
	}
	defer f.Close()
	d := json.NewDecoder(io.LimitReader(f, 4097))
	d.DisallowUnknownFields()
	var v app.Preferences
	if e = d.Decode(&v); e != nil || !validPreferences(v) {
		return s, errors.New("invalid settings; original preserved")
	}
	var extra any
	if e = d.Decode(&extra); e != io.EOF {
		return s, errors.New("invalid trailing settings data")
	}
	return v, nil
}
func writePreferences(p string, s app.Preferences) error {
	if !validPreferences(s) {
		return errors.New("invalid settings")
	}
	if _, e := readPreferences(p); e != nil {
		return e
	}
	if e := os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(p), "settings-*.tmp")
	if e != nil {
		return e
	}
	name := f.Name()
	defer os.Remove(name)
	e = json.NewEncoder(f).Encode(s)
	if e == nil {
		e = f.Sync()
	}
	closeErr := f.Close()
	if e != nil {
		return e
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(name, p)
}
func startupStatus(value string, exists bool, manager, remote string) string {
	if !exists {
		return "off"
	}
	if value == manager {
		return "manager"
	}
	if value == remote {
		return "remote"
	}
	return "conflict"
}
func validStartup(s string) bool { return s == "off" || s == "manager" || s == "remote" }
func ReadSettings() app.Settings {
	s := app.Settings{Preferences: app.DefaultPreferences(), Startup: "unknown"}
	if p, e := preferencesPath(); e == nil {
		if v, e := readPreferences(p); e == nil {
			s.Preferences = v
			s.PreferencesKnown = true
		}
	}
	if v, e := readStartup(); e == nil {
		s.Startup = v
	}
	return s
}
func ChangeSettings(r app.SettingsRequest) (app.Settings, error) {
	if r.Kind == "reload" {
		return ReadSettings(), nil
	}
	e := withSettingsLock(func() error {
		switch r.Kind {
		case "preferences":
			p, e := preferencesPath()
			if e != nil {
				return e
			}
			old, e := readPreferences(p)
			if e != nil {
				return e
			}
			if old != r.Before {
				return errors.New("settings changed; reload first")
			}
			return writePreferences(p, r.After)
		case "startup":
			if !validStartup(r.StartupBefore) || !validStartup(r.StartupAfter) {
				return errors.New("startup registration unavailable or not owned")
			}
			return changeStartup(r.StartupBefore, r.StartupAfter)
		default:
			return errors.New("unknown settings action")
		}
	})
	if e != nil {
		return ReadSettings(), errors.New("settings change refused or failed: " + sanitizeSettingsError(e))
	}
	got := ReadSettings()
	if r.Kind == "startup" && got.Startup != r.StartupAfter {
		return got, errors.New("startup change could not be verified")
	}
	if r.Kind == "preferences" && (!got.PreferencesKnown || got.Preferences != r.After) {
		return got, errors.New("preferences change could not be verified")
	}
	return got, nil
}
func sanitizeSettingsError(e error) string {
	// Do not return filesystem paths or registry data.
	for _, v := range []string{"settings changed; reload first", "startup registration changed; reload first", "startup registration unavailable or not owned", "invalid settings; original preserved", "unsupported settings file", "another settings operation is active"} {
		if strings.EqualFold(e.Error(), v) {
			return v
		}
	}
	return "check permissions and existing settings"
}
