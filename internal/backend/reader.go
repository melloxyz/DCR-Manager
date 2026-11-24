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

func Root() (string, error) {
	paths := []string{}
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Dir(exe), filepath.Dir(filepath.Dir(exe)))
	}
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, cwd)
	}
	for _, p := range paths {
		if info, err := os.Stat(filepath.Join(p, "src", "Bridge.ps1")); err == nil && !info.IsDir() {
			return p, nil
		}
	}
	return "", errors.New("source bundle not found; keep the executable beside src or in bin")
}
func Read(root string) app.Reader {
	return func(parent context.Context) (app.Snapshot, error) {
		var s app.Snapshot
		if runtime.GOOS != "windows" {
			return s, errors.New("Windows backend required")
		}
		ctx, cancel := context.WithTimeout(parent, 45*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "powershell.exe", "-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", filepath.Join(root, "src", "Bridge.ps1"))
		configure(cmd)
		out, err := cmd.Output()
		if ctx.Err() != nil {
			return s, errors.New("diagnostic timed out or was cancelled")
		}
		if err != nil {
			return s, fmt.Errorf("diagnostic command failed (%T); use legacy doctor for details", err)
		}
		if err = json.Unmarshal([]byte(strings.TrimPrefix(strings.TrimSpace(string(out)), "\uFEFF")), &s); err != nil {
			return s, errors.New("diagnostic returned invalid JSON")
		}
		if s.Status.Version == "" {
			return s, errors.New("diagnostic returned incomplete data")
		}
		return s, nil
	}
}
