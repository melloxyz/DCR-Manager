# DCR Manager v1.2.1

Independent community utility for Desktop Commander Remote on Windows. Not affiliated with the upstream project.

## Quick start

Requires Windows PowerShell 5.1 (included with Windows). Node.js and npm are required to install/run Desktop Commander itself, not to open the manager. Double-click Launch.cmd. Use option 5 for a missing installation, then option 1. Follow upstream authentication instructions in the Remote terminal if needed.

CLI: `Launch.cmd -Mode status` outputs JSON; `Launch.cmd -Mode doctor` gives a read-only diagnostic. Run `Test.cmd` for local automated tests.

## Safety

Dedicated installation: %LOCALAPPDATA%\DesktopCommander. Backups/reports: %LOCALAPPDATA%\DCRManager-state. Only recognized DCR cache directories with an exclusive package dependency are eligible for quarantine. No global cache wipe, no blanket Node termination. Repair stages packages and preserves the Remote directory; package switching requires all recognized DCR processes to be closed. Exported reports omit command lines, machine names, account identifiers, tokens and raw log content. Review reports before sharing; PIDs and log timestamps remain. Local process status does not establish server connectivity. The launcher uses execution policy Bypass for that process only.

## Scope

Windows-first. No telemetry, persistent manager service, or cloud credentials. Current version manages the dedicated npm install and discovers DCR Node processes and exclusive npx caches. Read-only inventory covers the dedicated install, current npm global root and exclusive npx caches. Client audit checks known user-level JSON locations; TOML audit supports Codex mcp_servers using optional Python 3.11+ (py launcher). Custom JSON/TOML files can be explicitly audited; project configs are not automatically scanned. No client configuration is rewritten. Node compatibility checks only simple minimum engine ranges. No automatic update. Package backups can be restored through menu B with DCR clients stopped; manual recovery is described in docs/RECOVERY.md.

## Development

Core.psm1: testable policy and filesystem operations. Operations.psm1: Windows/npm integration. Manager.ps1: presentation. Tests use isolated temporary fixtures. CI configuration targets Windows PowerShell and PowerShell 7; hosted CI is not yet run.

See docs/ROADMAP.md. License not selected; obtain permission before redistribution. Upstream: https://github.com/wonderwhy-er/DesktopCommanderMCP

CLI additions: Launch.cmd -Mode inventory / clients / runtime. Maintenance.psm1 contains inventory and recovery logic.

Recovery: cache restore refuses overwrite and checks current npm cache path; parent-process diagnosis is advisory; backup retention review flags age >=30 days without deletion. Use Launch.cmd -Mode health / backups, or -Mode config -ConfigPath "path". See docs/RECOVERY.md.

## v1.3 development preview
On feat/v1.3-tui, run Launch-TUI.cmd for the new Graphite Dark Go TUI. Build-TUI.cmd compiles it. The legacy manager remains available with Launch.cmd. See docs/TUI-MIGRATION.md for scope and limitations. The stable release remains v1.2.1.

The development preview now exposes Remote lifecycle actions on Connection (S / X / T); stop and restart require confirmation. Remote lifecycle now uses native Windows APIs in Go; read-only package and client diagnostics still use the PowerShell bridge. See docs/TUI-MIGRATION.md for verification limits and docs/SKILLS.md for project-scoped skills.

T6 preview: Packages (P repair/install, U update), Cache (C select next, Z quarantine), Y confirm and N/Esc cancel. Package operations stage with a separate npm cache and retain backups; cache quarantine moves only verified DCR candidates. Stop DCR/MCP clients before replacing package files. See docs/TUI-MIGRATION.md for scope, tests and limitations.
