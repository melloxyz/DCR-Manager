package backend

import (
	"context"
	"dcr-manager/internal/app"
	"errors"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

func processPath(v string) string { return strings.ToLower(strings.ReplaceAll(v, "\\", "/")) }
func nativeProcessInventory(rows []nativeCandidate, parents map[int]uint64, dedicated string) ([]app.Process, []app.Health, int) {
	remote := map[int]bool{}
	for _, r := range classifyRemote(rows) {
		remote[r.ID.PID] = true
	}
	p := []app.Process{}
	health := []app.Health{}
	roots := 0
	sorted := append([]nativeCandidate(nil), rows...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID.PID < sorted[j].ID.PID })
	for _, r := range sorted {
		if r.ID.PID <= 0 || r.ID.Created == 0 || !dcrEntry(r.Args) {
			continue
		}
		script := path.Clean(processPath(r.Args[1]))
		location := "Other"
		if dedicated != "" && strings.HasPrefix(script, strings.TrimSuffix(path.Clean(processPath(dedicated)), "/")+"/") {
			location = "Dedicated"
		} else if strings.Contains(script, "/_npx/") {
			location = "npx"
		}
		role := "MCP"
		if remote[r.ID.PID] {
			role = "Remote"
			if !remote[r.Parent] {
				roots++
			}
		}
		p = append(p, app.Process{PID: r.ID.PID, Created: r.ID.Created, Parent: r.Parent, Role: role, Location: location})
		parent, ok := parents[r.Parent]
		assessment := "parent-present"
		if !ok {
			assessment = "parent-missing"
		} else if parent == 0 {
			assessment = "parent-unknown"
		} else if parent > r.ID.Created {
			assessment = "parent-pid-reused"
		}
		health = append(health, app.Health{PID: r.ID.PID, Parent: r.Parent, Assessment: assessment})
	}
	return p, health, roots
}
func reconcileProcessInventory(s *app.Snapshot, p []app.Process, h []app.Health, roots int, err error) {
	s.ProcessControlReady = false
	s.ProcessControlIssue = "Native process inspection unavailable; refresh before acting."
	for i := range s.Processes {
		s.Processes[i].Created = 0
	}
	if err != nil {
		return
	}
	s.ProcessControlIssue = "Process inventories disagree; refresh before acting."
	old := map[int]app.Process{}
	for _, r := range s.Processes {
		if _, ok := old[r.PID]; ok {
			return
		}
		old[r.PID] = r
	}
	if len(old) != len(p) {
		return
	}
	remote, other := 0, 0
	for _, r := range p {
		o, ok := old[r.PID]
		if !ok || r.Created == 0 || r.Parent != o.Parent || r.Role != o.Role || r.Location != o.Location {
			return
		}
		delete(old, r.PID)
		if r.Role == "Remote" {
			remote++
		} else {
			other++
		}
	}
	if len(old) != 0 || remote != s.Status.RemoteProcesses || other != s.Status.OtherDcrProcesses || roots != s.Status.RemoteRoots {
		return
	}
	s.Processes = p
	s.Health = h
	s.Status.RemoteProcesses = remote
	s.Status.OtherDcrProcesses = other
	s.Status.RemoteRoots = roots
	s.ProcessControlReady = true
	s.ProcessControlIssue = ""
}
func attachNativeInventory(s *app.Snapshot) {
	rows, e := nativeRows()
	if e != nil {
		reconcileProcessInventory(s, nil, nil, 0, e)
		return
	}
	parents, e := nativeParentTimes(rows)
	if e != nil {
		reconcileProcessInventory(s, nil, nil, 0, e)
		return
	}
	root := os.Getenv("LOCALAPPDATA")
	dedicated := ""
	if filepath.IsAbs(root) {
		dedicated = filepath.Join(root, "DesktopCommander")
	}
	p, h, roots := nativeProcessInventory(rows, parents, dedicated)
	reconcileProcessInventory(s, p, h, roots, nil)
}

// Conservative independent preflight, not a replacement for the legacy
// in-mutex guards, which also inspect non-Node processes and revalidate paths.
func nativeGuard(rows []nativeCandidate, action app.MaintenanceAction, target string) error {
	for _, r := range rows {
		if r.ID.Created == 0 || len(r.Args) == 0 {
			return errors.New("Node process inspection incomplete; maintenance refused")
		}
		for _, arg := range r.Args {
			a := processPath(arg)
			if action != app.QuarantineCache && (strings.Contains(a, "/@wonderwhy-er/desktop-commander/") || strings.Contains(a, "/node_modules/.bin/desktop-commander")) {
				return maintenanceError("ACTIVE")
			}
			if target != "" && strings.Contains(a, processPath(target)) {
				return maintenanceError("ACTIVE")
			}
		}
	}
	return nil
}
func preflightNativeMaintenance(ctx context.Context, r app.MaintenanceRequest) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	local := os.Getenv("LOCALAPPDATA")
	if !filepath.IsAbs(local) {
		return maintenanceError("UNSAFE")
	}
	target := filepath.Join(local, "DesktopCommander")
	if r.Action == app.QuarantineCache {
		cache, e := npmDirectory(ctx, "cache")
		if e != nil {
			return maintenanceError("UNSAFE")
		}
		target = filepath.Join(cache, "_npx", r.CacheID)
	} else if r.Action == app.RestoreBackup {
		target = filepath.Join(local, "DCRManager-state", r.CacheID)
	}
	rows, e := nativeRows()
	if e != nil {
		return errors.New("Native process inspection unavailable; maintenance refused")
	}
	if e = ctx.Err(); e != nil {
		return e
	}
	return nativeGuard(rows, r.Action, target)
}
