# V14-01 native backend migration

Goal: remove PowerShell/Python from the Go application's runtime path without weakening safety, privacy or recoverability. The legacy PowerShell application remains available until separately retired. Windows AMD64 and English-only Graphite Dark remain the supported development target. Node.js/npm are upstream prerequisites; portable does not mean bundling DCR or Node.

| Stage | Scope | State |
| --- | --- | --- |
| V14-01A | Native bounded npm history and six JSON MCP locations | Implemented; local parity and regression |
| V14-01B | Native TOML parsing, runtime/version and npm command discovery | Implemented; local parity and regression |
| V14-01C | Native installation/cache/process health/recovery inventory with independent process guard parity | Pending |
| V14-01D | Staged package maintenance, cache quarantine, recovery and shared Windows mutex in Go | Pending |
| V14-01E | Cross-backend parity, fault injection, legacy interoperability, clean-profile acceptance | Pending |

Do not remove a bridge capability before its replacement has fixture coverage, comparison against the existing backend and fail-closed behavior. Preserve package ownership checks, path/reparse restrictions, PID and creation identities, stale-token guards, credentials and rollback journals. Native transaction tests must use isolated directories and must not stop the live Remote. Treat supported engine ranges and npm command discovery as explicit contracts. Evaluate a maintained TOML parser separately, including license and dependency review; no new dependency added in 01A.

## V14-01A
Go now reads six known JSON locations (Claude Desktop, Claude Code, Cursor, Windsurf, Antigravity, VS Code). It recognizes root mcpServers/servers, UTF-8 with optional BOM, exact package references and known launch modes. Invalid, oversized or unsupported files are not assessed; missing files are reported separately. Only counts and fixed classifications are returned. The 5 MiB cap, ancestor checks, reparse rejection and file identity check restrict reads, but do not constitute an OS sandbox against external races. Codex TOML still uses the optional Python-backed legacy audit.

Native npm history scans at most 10,000 directory entries, considers at most 100 recent timestamp-shaped names, reads at most 2 MiB per file and returns at most 20 matching entries; the five-second budget is cooperative between reads. Only the six allowlisted codes, timestamp-shaped names and modification times are returned. A scan that exceeds its limits or cannot read a selected file is partial. It does not prove current locks or server connectivity. The cache location comes from the existing PowerShell npm query, is consumed internally by Go and is excluded from the public Snapshot. The bridge's internal output is not a shareable diagnostic report.

The PowerShell bridge defaults to legacy behavior; its NativeDiagnostics switch skips the six JSON audits and the old log scan. A version marker prevents mixing an incompatible bridge with the new Go reader. Other legacy fields and all maintenance mutations are unchanged.

Validation: TDD tests for privacy, malformed and oversized inputs, cancelled/unknown scans, recent-entry cap and missing environment roots. Symlink rejection fixture requires Windows symlink privileges and may be skipped; actual junction/clean-machine testing remains pending. A live read-only comparison returned identical counts/status for all seven client rows and identical empty retained npm history; this is one environment, not universal equivalence. The active Remote process inventory was independently verified. No production package, cache, credentials or startup entry were modified.

The feature branch uses Go application version 1.4.0-dev; VERSION and main retain stable product source milestone 1.3.1. Build the development binary separately; do not distribute it as a stable release. V14-02 packaging and V14-05 clean-machine acceptance follow native parity.


## V14-01B

The Go application now audits all seven known MCP locations, including Codex TOML. It uses the pinned MIT-licensed github.com/pelletier/go-toml/v2 v2.4.3 parser, with no additional required modules. This is a deliberate dependency instead of an incomplete hand-written TOML parser. Preserve its license when packaging. The same bounded read, ancestor and file identity checks apply. Only root mcp_servers is evaluated; plugin/project configurations, indirect launchers and all possible version ranges are not inferred. Invalid, oversized or unreadable files remain explicitly unassessed. No configuration content, server names or environment values are returned.

Node and npm version probes resolve fixed executable names through PATH, reject relative/current-directory resolution, limit stdout to 4 KiB and use five-second per-probe deadlines plus bounded pipe waiting. Only syntactically valid version tokens are returned. Missing commands and failed/unsupported probes are distinct; failed probes disable the corresponding availability flag. Windows npm.cmd is invoked via the system cmd.exe using a fixed --version argument, disabled AutoRun/delayed expansion, explicit quoting and rejection of shell metacharacters in launcher paths. The installed runtime and PATH must still be trusted. Timeout handling is not a process-tree sandbox. Compatibility assessment supports only exact numeric Node versions and simple >= minimum ranges; compound/prerelease ranges remain unknown.

Bridge schema 2 (NativeRuntime) skips the legacy runtime/version and all client audit calls. Existing default and schema 1 modes remain compatible. The Go application no longer requires audit_toml.py in its bundle or invokes Python for auditing. Legacy PowerShell custom audit remains optional Python-backed. Installation inventory still supplies the declared Node engine range, and PowerShell still obtains npm cache/global paths and performs its inventory/maintenance; those calls move in later stages. The new development binary is not yet standalone.

Local read-only parity: Node v24.15.0, npm 11.12.1, declared >=22.12.0 and meets-minimum matched the legacy snapshot; all seven MCP rows matched status/reference/floating counts. No live process was stopped. Windows fixtures verify fixed npm argument quoting with spaces/parentheses and rejection of unsafe launcher names. Clean-profile acceptance, alternative npm launchers and complex Node engine ranges remain outside this verified scope.
