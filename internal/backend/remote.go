package backend

import (
	"context"
	"dcr-manager/internal/app"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

func actionArgs(action app.RemoteAction, pids []int) ([]string, error) {
	if !action.Valid() {
		return nil, errors.New("unknown Remote action")
	}
	if action == app.Start && len(pids) != 0 || action != app.Start && len(pids) == 0 {
		return nil, errors.New("Remote state changed; refresh first")
	}
	ids := append([]int(nil), pids...)
	sort.Ints(ids)
	parts := make([]string, len(ids))
	for i, id := range ids {
		if id <= 0 || i > 0 && id == ids[i-1] {
			return nil, errors.New("invalid Remote process identity")
		}
		parts[i] = strconv.Itoa(id)
	}
	expected := "none"
	if len(parts) > 0 {
		expected = strings.Join(parts, ",")
	}
	return []string{"-Action", string(action), "-Expected", expected}, nil
}
func Operate(root string) app.RemoteOperator {
	return func(parent context.Context, action app.RemoteAction, pids []int) error {
		args, err := actionArgs(action, pids)
		if err != nil {
			return err
		}
		if runtime.GOOS != "windows" {
			return errors.New("Windows backend required")
		}
		ctx, cancel := context.WithTimeout(parent, 35*time.Second)
		defer cancel()
		base := []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", filepath.Join(root, "src", "RemoteAction.ps1")}
		cmd := exec.CommandContext(ctx, "powershell.exe", append(base, args...)...)
		configure(cmd)
		out, err := cmd.Output()
		if ctx.Err() != nil {
			return errors.New("Remote operation timed out; refresh to verify actual state")
		}
		if err != nil {
			return fmt.Errorf("Remote operation refused or failed; refresh and inspect local state")
		}
		if strings.TrimSpace(string(out)) != "OK" {
			return errors.New("Remote result unverified; refresh")
		}
		return nil
	}
}
