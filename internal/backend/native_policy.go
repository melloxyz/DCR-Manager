package backend

import (
	"dcr-manager/internal/app"
	"errors"
	"path"
	"sort"
	"strings"
)

type nativeCandidate struct {
	ID     app.ProcessIdentity
	Parent int
	Args   []string
}

func dcrEntry(args []string) bool {
	if len(args) < 2 {
		return false
	}
	p := strings.ToLower(strings.ReplaceAll(args[1], "\\", "/"))
	// Only recognize an actual absolute script argument, never arbitrary args.
	absolute := len(p) > 3 && p[1] == ':' && p[2] == '/' || strings.HasPrefix(p, "//")
	if !absolute {
		return false
	}
	p = path.Clean(p)
	return strings.HasSuffix(p, "/node_modules/@wonderwhy-er/desktop-commander/dist/index.js") ||
		strings.HasSuffix(p, "/node_modules/.bin/desktop-commander")
}
func classifyRemote(rows []nativeCandidate) []nativeCandidate {
	known := map[int]nativeCandidate{}
	roots := map[int]bool{}
	for _, r := range rows {
		if r.ID.PID <= 0 || r.ID.Created == 0 || !dcrEntry(r.Args) {
			continue
		}
		known[r.ID.PID] = r
		if len(r.Args) > 2 && r.Args[2] == "remote" {
			roots[r.ID.PID] = true
		}
	}
	for {
		changed := false
		for pid, r := range known {
			p, ok := known[r.Parent]
			if !roots[pid] && ok && roots[r.Parent] && p.ID.Created <= r.ID.Created {
				roots[pid] = true
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	var out []nativeCandidate
	for _, r := range known {
		if roots[r.ID.PID] {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID.PID < out[j].ID.PID })
	return out
}
func identities(rows []nativeCandidate) []app.ProcessIdentity {
	out := make([]app.ProcessIdentity, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.ID)
	}
	return out
}
func sameIdentities(a, b []app.ProcessIdentity) bool {
	if len(a) != len(b) {
		return false
	}
	m := map[int]uint64{}
	for _, x := range a {
		if x.PID <= 0 || x.Created == 0 || m[x.PID] != 0 {
			return false
		}
		m[x.PID] = x.Created
	}
	for _, x := range b {
		if x.Created == 0 || m[x.PID] != x.Created {
			return false
		}
		delete(m, x.PID)
	}
	return len(m) == 0
}
func validateAction(a app.RemoteAction, ids []app.ProcessIdentity) error {
	if !a.Valid() {
		return errors.New("unknown Remote action")
	}
	if (a == app.Start) != (len(ids) == 0) {
		return errors.New("Remote state changed; refresh first")
	}
	if !sameIdentities(ids, ids) {
		return errors.New("invalid process identity")
	}
	return nil
}

// Cross-check the independently obtained legacy inventory before enabling
// mutation. A mismatch never grants authority to either source alone.
func attachNativeIdentities(s *app.Snapshot) {
	rows, err := nativeRows()
	reconcileNative(s, rows, err)
}

func reconcileNative(s *app.Snapshot, rows []nativeCandidate, err error) {
	s.ProcessControlReady = false
	s.ProcessControlIssue = "Native process inspection unavailable; refresh or use legacy diagnostics."
	for i := range s.Processes {
		s.Processes[i].Created = 0
	}
	if err != nil {
		return
	}
	native := classifyRemote(rows)
	ids := map[int]uint64{}
	for _, r := range native {
		ids[r.ID.PID] = r.ID.Created
	}
	count := 0
	for _, p := range s.Processes {
		if p.Role != "Remote" {
			continue
		}
		count++
		created, ok := ids[p.PID]
		if !ok || created == 0 {
			s.ProcessControlIssue = "Process inventories disagree; refresh before acting."
			return
		}
		delete(ids, p.PID)
	}
	if len(ids) != 0 || count != s.Status.RemoteProcesses {
		s.ProcessControlIssue = "Process inventories disagree; refresh before acting."
		return
	}
	for i := range s.Processes {
		if s.Processes[i].Role != "Remote" {
			continue
		}
		for _, r := range native {
			if r.ID.PID == s.Processes[i].PID {
				s.Processes[i].Created = r.ID.Created
				break
			}
		}
	}
	s.ProcessControlReady = true
	s.ProcessControlIssue = ""
}
