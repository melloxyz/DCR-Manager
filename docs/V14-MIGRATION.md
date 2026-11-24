# V14-01 native backend migration

Goal: remove PowerShell/Python from the Go application's runtime path without weakening safety, privacy or recoverability. The legacy PowerShell application remains available until separately retired. Windows AMD64 and English-only Graphite Dark remain the supported development target. Node.js/npm are upstream prerequisites; portable does not mean bundling DCR or Node.

| Stage | Scope | State |
| --- | --- | --- |
| V14-01A | Native bounded npm history and six JSON MCP locations | Implemented; local parity and regression |
| V14-01B | Native TOML parsing, runtime/version and npm command discovery | Pending |
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
