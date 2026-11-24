package backend

import (
	"context"
	"dcr-manager/internal/app"
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var versionPattern = regexp.MustCompile("^[0-9]+\\.[0-9]+\\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$")
var cacheIDPattern = regexp.MustCompile("^[a-f0-9]{8,64}$")
var cacheTokenPattern = regexp.MustCompile("^[a-f0-9]{64}$")

func maintenanceArgs(r app.MaintenanceRequest) ([]string, error) {
	switch r.Action {
	case app.RepairPackage, app.UpdatePackage:
		if r.CacheID != "" || (!versionPattern.MatchString(r.Expected) && r.Expected != "absent" && !(r.Action == app.UpdatePackage && r.Expected == "unknown")) {
			return nil, errors.New("invalid package version expectation")
		}
	case app.QuarantineCache:
		if !cacheIDPattern.MatchString(r.CacheID) || !cacheTokenPattern.MatchString(r.Expected) {
			return nil, errors.New("invalid cache identity")
		}
	default:
		return nil, errors.New("unknown maintenance action")
	}
	args := []string{"-Action", string(r.Action), "-Expected", r.Expected}
	if r.CacheID != "" {
		args = append(args, "-CacheID", r.CacheID)
	}
	return args, nil
}

type maintenanceResult struct {
	OK   bool
	Code string
}

func maintenanceError(code string) error {
	switch code {
	case "EBUSY", "EPERM", "EACCES":
		return errors.New(code + ": resource locked or access denied; close affected clients and retry")
	case "ACTIVE":
		return errors.New("DCR or MCP still active; stop affected clients before maintenance")
	case "STALE":
		return errors.New("target changed; refresh before retrying")
	case "LOCKED":
		return errors.New("another maintenance manager is open")
	case "ROLLBACK":
		return errors.New("rollback incomplete; preserve staging and backups, see Recovery")
	case "UNSAFE":
		return errors.New("unsafe or unsupported directory layout; no automatic changes")
	case "NPM":
		return errors.New("npm staging failed; original installation not replaced")
	default:
		return errors.New("maintenance not verified; refresh and inspect Recovery before retrying")
	}
}
func Maintain(root string) app.Maintainer {
	return func(ctx context.Context, r app.MaintenanceRequest) error {
		args, err := maintenanceArgs(r)
		if err != nil {
			return err
		}
		if runtime.GOOS != "windows" {
			return errors.New("Windows backend required")
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		// Do not kill the bridge in the middle of a package transaction.
		// npm fetch retries/timeouts are bounded in the script.
		base := []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", filepath.Join(root, "src", "MaintenanceAction.ps1")}
		cmd := exec.Command("powershell.exe", append(base, args...)...)
		configure(cmd)
		out, err := cmd.Output()
		var result maintenanceResult
		if json.Unmarshal([]byte(strings.TrimPrefix(strings.TrimSpace(string(out)), "\uFEFF")), &result) != nil {
			return maintenanceError("")
		}
		if err != nil || !result.OK {
			return maintenanceError(result.Code)
		}
		return nil
	}
}
