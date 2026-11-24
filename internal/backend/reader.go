package backend

import (
	"context"
	"dcr-manager/internal/app"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var bundleScripts = []string{"Bridge.ps1", "Core.psm1", "Operations.psm1", "Maintenance.psm1", "PackageCache.psm1", "Recovery.psm1", "RecoveryCenter.psm1", "Diagnostics.psm1", "MaintenanceAction.ps1", "audit_toml.py"}

func rootForExecutable(exe string) (string, error) {
	dir := filepath.Dir(exe)
	paths := []string{dir}
	if strings.EqualFold(filepath.Base(dir), "bin") {
		paths = append(paths, filepath.Dir(dir))
	}
	for _, p := range paths {
		complete := true
		for _, name := range bundleScripts {
			path := filepath.Join(p, "src", name)
			info, err := os.Lstat(path)
			if err != nil || !info.Mode().IsRegular() {
				complete = false
				break
			}
		}
		if complete {
			return p, nil
		}
	}
	return "", errors.New("complete source bundle not found beside executable or its bin directory")
}
func Root() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", errors.New("cannot locate executable")
	}
	return rootForExecutable(exe)
}
func Read(root string) app.Reader {
	return func(parent context.Context) (app.Snapshot, error) {
		var s app.Snapshot
		if runtime.GOOS != "windows" {
			return s, errors.New("Windows backend required")
		}
		ctx, cancel := context.WithTimeout(parent, 45*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "powershell.exe", "-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", filepath.Join(root, "src", "Bridge.ps1"), "-NativeDiagnostics")
		configure(cmd)
		out, err := cmd.Output()
		if ctx.Err() != nil {
			return s, errors.New("diagnostic timed out or was cancelled")
		}
		if err != nil {
			return s, fmt.Errorf("diagnostic command failed (%T); use legacy doctor for details", err)
		}
		// Internal cache location is consumed here, never added to public Snapshot.
		var wire struct {
			app.Snapshot
			NativeDiagnosticsSchema int
			InternalCacheRoot       string
		}
		if err = json.Unmarshal([]byte(strings.TrimPrefix(strings.TrimSpace(string(out)), "\uFEFF")), &wire); err != nil {
			return s, errors.New("diagnostic returned invalid JSON")
		}
		s = wire.Snapshot
		if wire.NativeDiagnosticsSchema != 1 {
			return app.Snapshot{}, errors.New("diagnostic bridge version mismatch")
		}
		if s.Status.Version == "" {
			return s, errors.New("diagnostic returned incomplete data")
		}
		s.Logs, s.LogsKnown = nativeLogs(ctx, wire.InternalCacheRoot)
		s.Clients = append(jsonClientAudits(os.Getenv("USERPROFILE"), os.Getenv("APPDATA")), s.Clients...)
		if ctx.Err() != nil {
			return app.Snapshot{}, errors.New("diagnostic timed out or was cancelled")
		}
		s.ManagerVersion = app.Version
		attachNativeIdentities(&s)
		return s, nil
	}
}
