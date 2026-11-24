//go:build windows

package backend

import (
	"errors"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"os"
	"path/filepath"
	"runtime"
)

const runKey = "Software\\Microsoft\\Windows\\CurrentVersion\\Run"
const runName = "DCRManager"

func startupCommands() (string, string, error) {
	exe, e := os.Executable()
	if e != nil {
		return "", "", e
	}
	exe, e = filepath.Abs(exe)
	if e != nil {
		return "", "", e
	}
	// Quote even when no spaces: prevents ambiguous executable paths.
	manager := "\"" + exe + "\""
	return manager, manager + " --start-remote-on-login", nil
}
func queryStartup(k registry.Key, name, manager, remote string) (string, error) {
	v, typ, e := k.GetStringValue(name)
	if errors.Is(e, registry.ErrNotExist) {
		return "off", nil
	}
	if e != nil {
		// A non-string value under our name is a conflict, not permission to replace.
		if _, _, q := k.GetValue(name, nil); q == nil {
			return "conflict", nil
		}
		return "unknown", e
	}
	if typ != registry.SZ {
		return "conflict", nil
	}
	return startupStatus(v, true, manager, remote), nil
}
func readStartup() (string, error) {
	m, r, e := startupCommands()
	if e != nil {
		return "unknown", e
	}
	k, e := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if errors.Is(e, registry.ErrNotExist) {
		return "off", nil
	}
	if e != nil {
		return "unknown", e
	}
	defer k.Close()
	return queryStartup(k, runName, m, r)
}
func setStartup(k registry.Key, name, manager, remote, before, after string) error {
	old, e := queryStartup(k, name, manager, remote)
	if e != nil {
		return e
	}
	if old != before || !validStartup(old) || !validStartup(after) {
		return errors.New("startup registration changed; reload first")
	}
	if old == after {
		return nil
	}
	switch after {
	case "off":
		return k.DeleteValue(name)
	case "manager":
		return k.SetStringValue(name, manager)
	case "remote":
		return k.SetStringValue(name, remote)
	}
	return errors.New("startup registration unavailable or not owned")
}
func changeStartup(before, after string) error {
	m, r, e := startupCommands()
	if e != nil {
		return e
	}
	k, e := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE|registry.SET_VALUE)
	if errors.Is(e, registry.ErrNotExist) {
		if before != "off" {
			return errors.New("startup registration changed; reload first")
		}
		if after == "off" {
			return nil
		}
		k, _, e = registry.CreateKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE|registry.SET_VALUE)
	}
	if e != nil {
		return e
	}
	defer k.Close()
	return setStartup(k, runName, m, r, before, after)
}
func withSettingsLock(fn func() error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	name, _ := windows.UTF16PtrFromString("Local\\DCRManagerSettings")
	h, e := windows.CreateMutex(nil, false, name)
	if e != nil && !errors.Is(e, windows.ERROR_ALREADY_EXISTS) {
		return e
	}
	if h == 0 {
		return errors.New("settings lock unavailable")
	}
	defer windows.CloseHandle(h)
	state, e := windows.WaitForSingleObject(h, 0)
	if e != nil {
		return e
	}
	if state != windows.WAIT_OBJECT_0 && state != windows.WAIT_ABANDONED {
		return errors.New("another settings operation is active")
	}
	defer windows.ReleaseMutex(h)
	return fn()
}
