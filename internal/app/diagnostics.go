package app

import "fmt"

type Finding struct {
	ID, Severity, Title, Detail, Next string
	Page                              int
}

func CodeGroup(code string) string {
	switch code {
	case "EBUSY", "EPERM", "EACCES":
		return "locks"
	case "ENOTFOUND", "ETIMEDOUT", "ECONNRESET":
		return "network"
	default:
		return ""
	}
}
func FilterLogs(logs []Log, group string) []Log {
	var out []Log
	for _, l := range logs {
		if group == "all" {
			out = append(out, l)
			continue
		}
		for _, c := range l.Codes {
			if CodeGroup(c) == group && group != "" {
				out = append(out, l)
				break
			}
		}
	}
	return out
}

// Diagnose is advisory and deterministic; no filesystem or process mutations.
func Diagnose(s Snapshot) []Finding {
	var out []Finding
	add := func(id, severity, title, detail, next string, page int) {
		out = append(out, Finding{id, severity, title, detail, next, page})
	}
	for _, b := range s.Backups {
		if b.Status == "rollback-incomplete" || b.Status == "switching" {
			add("rollback", "critical", "Unfinished package transaction", "A stage reports interrupted or incomplete recovery.", "Preserve all files; review Recovery before package changes.", 5)
			break
		}
	}
	if !s.Status.NodeAvailable || !s.Status.NpmAvailable {
		add("runtime", "critical", "Runtime missing", "Node.js or npm was not detected.", "Install or repair the supported runtime, then refresh.", 2)
	} else if s.Runtime.Assessment == "below-minimum" {
		add("runtime", "critical", "Node.js below requirement", "The installed version is below the declared minimum.", "Review Node.js requirements before starting DCR.", 2)
	} else if s.Runtime.Assessment != "meets-minimum" {
		add("runtime-unknown", "review", "Runtime compatibility unverified", "The available assessment does not confirm compatibility.", "Review the package engine requirement manually.", 2)
	}
	if !s.ProcessControlReady {
		add("inventory", "review", "Process controls locked", "Process identity verification did not succeed.", "Refresh and inspect process inventory; do not kill arbitrary Node processes.", 3)
	}
	if !s.Status.Installed {
		add("install", "review", "Dedicated installation unavailable", "The dedicated package entry was not detected.", "Review Packages; install or repair only after checking active clients.", 2)
	}
	if len(FilterLogs(s.Logs, "locks")) > 0 {
		add("lock-history", "review", "Historical npm lock or permission error", "Retained logs contain EBUSY, EPERM or EACCES. This is not proof of a current lock.", "Close affected clients; inspect Cache or Packages. Avoid global cache deletion.", 4)
	}
	if s.Status.CacheKnown && s.Status.CacheCandidates > 0 {
		add("cache", "review", "Legacy cache candidates", fmt.Sprintf("%d exclusive candidate(s) reported.", s.Status.CacheCandidates), "Inspect the selected cache; quarantine only if appropriate.", 4)
	} else if !s.Status.CacheKnown {
		add("cache-unknown", "review", "npm cache not assessed", "The cache location or inventory is unavailable.", "Refresh and check npm configuration.", 4)
	}
	if len(FilterLogs(s.Logs, "network")) > 0 {
		add("network-history", "review", "Historical npm network error", "Retained npm logs contain DNS, timeout or reset codes.", "Check network and npm registry access; Remote connectivity is a separate check.", 7)
	}
	for _, p := range s.Health {
		if p.Assessment == "parent-missing" || p.Assessment == "parent-pid-reused" {
			add("parent", "review", "Process parent needs review", "A parent is missing or its PID may have been reused.", "Inspect Processes; this does not prove an orphan or stuck process.", 3)
			break
		}
	}
	for _, c := range s.Clients {
		if c.Status == "checked" && c.Floating > 0 {
			add("floating", "review", "Unpinned MCP package reference", "A known client uses a floating DCR package reference.", "Review MCP Clients; consider a pinned version or dedicated entry.", 6)
			break
		}
	}
	for _, c := range s.Clients {
		if c.Status != "checked" && c.Status != "not-found" {
			add("audit", "review", "Some MCP configurations not assessed", "A known configuration could not be parsed or inspected.", "Review audit status; do not assume the client is unconfigured.", 6)
			break
		}
	}
	if !s.LogsKnown {
		add("logs-unknown", "review", "Log scan unavailable or partial", "The bounded scan could not inspect all selected files.", "Review Logs and refresh; absence of evidence is not a clean bill of health.", 7)
	}
	if !s.RecoveryKnown {
		add("recovery-unknown", "review", "Recovery inventory unavailable", "Backups could not be assessed.", "Review Recovery before replacing package files.", 5)
	}
	if s.Status.RemoteProcesses == 0 {
		add("remote-stopped", "info", "Remote not running locally", "No recognized Remote process was detected.", "Start only if you need a Remote session.", 1)
	}
	return out
}
