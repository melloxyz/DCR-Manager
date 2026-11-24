# Development handoff

Last reviewed: 2026-11-24. This is the continuation entry point for the v1.4.0 development branch. Read actual source and Git status before making changes; historical observations here are not live machine state.

## Repository and version policy

- Official repository: https://github.com/melloxyz/DCR-Manager
- Stable source: main and annotated tag v1.3.1, commit 4ab08254ee7f0bce9eb881f6f2208685ffff8478.
- Active development: feat/v1.4-native-backend; functional baseline ec72ee9984aa667128a652464e88369e383a31a2. Later documentation commits may advance HEAD.
- Go app version: 1.4.0-dev. VERSION remains 1.3.1 until promotion. Legacy PowerShell manager remains v1.2.1. MIT license; English-only UI; Graphite Dark.
- The original published history is authoritative. The separate sanitized RC2 publication clone was not used for publication; its history is not a development branch and must not be pushed over main. Preserve public tags and do not force-push.
- Work in the single canonical checkout; keep the stable executable and dev executable separate. No public binary release has been made.

## Read first

1. AGENTS.md
2. README.md
3. docs/VERSIONING.md
4. docs/ROADMAP.md
5. docs/V14-MIGRATION.md (including the most recent sections, which supersede earlier in-progress status)
6. docs/ARCHITECTURE.md
7. docs/RELEASE-CHECKLIST.md
8. CHANGELOG.md
9. Relevant actual source and tests below.

## Completed work

v1.1/v1.2 established modular PowerShell management, dedicated installation, selective repair, rollback, audit and diagnostics. v1.3 migrated the UI to Go/Charm, added native Windows lifecycle, nine TUI pages, settings, first-run guidance and optional login startup. Maintainer approved v1.3.0 source; v1.3.1 integrated four Dependabot updates, refreshed notices and grouped patch updates. Stable main and its tag are preserved. Distribution remains a separate gate.

| Stage | State |
| --- | --- |
| V14-01A | Complete: native npm logs and six JSON MCP audits |
| V14-01B | Complete: native Codex TOML, Node/npm probes; go-toml/v2 v2.4.3 |
| V14-01C.1 | Complete: npm directory discovery, installation/cache inventory |
| V14-01C.2 | Complete: recovery inventory, journals, restore-token parity |
| V14-01C.3 | Complete: process inventory/parent health and independent preflight |
| V14-01D.1a | Complete: internal staging and real npm artifact validation |
| V14-01D.1b | NEXT: native package switch, journal, rollback fixtures |
| V14-01D.2 | Pending: cache quarantine |
| V14-01D.3 | Pending: package/cache restore |
| V14-01D.4 | Pending: shared mutex and fault/recovery hardening |
| V14-01D.5 | Pending: integration, guard parity and replacement of legacy maintenance |
| V14-01E | Pending: full backend acceptance |

V14-02 portable executable/bundle; V14-03 release engineering and notices; V14-04 distribution documentation; V14-05 clean Windows acceptance; V14-06 public release. V2.0 is exploratory. Do not jump ahead or mark v1.4 stable prematurely.

## Architecture and current contracts

- cmd/dcr-manager: CLI, snapshot/settings, render and login entry points. internal/app: typed contracts, diagnostic rules, version; internal/tui: Bubble Tea v2, Lip Gloss v2, Bubbles viewport. Nine pages: Overview, Connection, Packages, Processes, Cache, Recovery, MCP Clients, Logs, Settings.
- internal/backend/reader.go uses bridge schema 3 NativeInventory. Go reads logs, all seven known MCP configs, runtime, npm directories, installs, cache, recovery and process health. Bridge still provides independent status, token, process and recovery parity. Mismatches fail closed; never remove this guard simply to make a test green.
- native_diagnostics*.go, native_toml.go, native_runtime*.go: bounded, sanitized reads/probes. native_inventory*.go, inventory_time*.go: path validation and cache tokens. native_recovery*.go: backup measurement, stage classification, tokens. native_health*.go, native_policy.go, native_windows.go: exact entry classification, process creation identity, native handle lifecycle, parent assessment and preflight.
- legacy_env*.go isolates Windows PowerShell child module discovery from inherited PSModulePath. Hosted failures proved the runner used 8.3 path aliases; GetLongPathName with same-object verification now normalizes legacy token identity. Keep both full-token cross-backend tests.
- native_stage*.go: prepareNativePackage validates stage roots, ownership, expected version, process guard before/after, random stage name, pinned repair or latest install/update, exact manifest/lockfile/version and regular entry. runNativeArtifactStage uses Node/npm-cli.js without shell assembly, isolated npm configuration/cache, fixed args, ignore-scripts and deadline. No activation; stage is not yet wired to Maintain/TUI. Failed stages are preserved.
- maintenance.go still delegates mutations to src/MaintenanceAction.ps1 after independent Go preflight. src/PackageCache.psm1, src/Core.psm1 (Switch-DcrPackageFiles), src/RecoveryCenter.psm1, src/Recovery.psm1 are the migration reference. Do not change the legacy production contract without parity evidence.
- Dedicated install is %LOCALAPPDATA%/DesktopCommander; state is %LOCALAPPDATA%/DCRManager-state; preferences %APPDATA%/DCRManager. Never publish contents. Package transaction may move only node_modules, package.json and package-lock.json. Never move Remote credential directory, shims, unrelated files or arbitrary npm cache.
- Native lifecycle and legacy transactions use Local\\DCRManager Windows mutex. Native process preflight is not a substitute for in-mutex all-process/path checks. PID + creation time, not PID alone. Missing parents are advisory, not proof of orphaning. No real Remote stop/restart, registry startup write or client configuration edits during tests.
- Keep paths absolute, reject reparse points/unsupported layouts, use same-volume no-overwrite moves, preserve backup and journal evidence; do not claim race-free or cryptographic full-tree authenticity.

## Next work: D.1b

Implement the native switch as an isolated primitive first. Start with TDD; read the actual PowerShell switch and recovery token contracts, not just this handoff.

1. Define the transaction plan and preconditions: only the three allowlisted package artifacts, exclusive roots on same volume, stage identity, validated package, expected current version; detect occupied or linked destinations and refuse overwrite. Revalidate whole tree where required; ancestor checks alone are insufficient.
2. Add a compatible schema-1 transaction journal in the stage (switching, completed, switch-failed, rollback-incomplete). Use durable writes and safe replacement; document crash windows and preserve evidence. Do not overwrite unrelated or existing journal/backup destinations.
3. Create a unique package-backup directory. Move prior tracked artifacts to backup and staged artifacts into install via native no-replace Windows moves. Revalidate process guard and expected current state before switching; no cancellation in the middle of a switch unless rollback is safely handled.
4. On injected move/post-verification failures, rollback independently in reverse order, verify outcome, distinguish rollback-incomplete. Keep unrelated data and credential fixture unchanged. No automatic deletion. Verify compatibility with native and legacy recovery inventory/token logic.
5. Tests: fresh install, repair/update, backup content, occupied target, stale input, invalid/reparse paths, failed journal, move failures at each boundary, rollback failure, post-switch mismatch, canceled preflight and exact preserved credentials. Use temporary directories and injected operations; do not target active user installation.
6. Keep Maintain wired to PowerShell until D.4/D.5 shared mutex, guard coverage, recovery and production npm lifecycle acceptance pass. D.1b does not authorize live package replacement. Document results, run regressions, commit conventionally and push only the feature branch. No merge to main without acceptance.

Review outstanding limitations as you go: lockfile validation is structural, not a complete dependency integrity audit; native staging does not yet validate entire trees, hold the shared mutex or execute lifecycle scripts. The standard adjacent npm CLI layout is supported for artifact tests only; alternate launchers, trusted npm configuration and production lifecycle behavior need explicit validation. Reparse/junction/race behavior and crash consistency still need disposable-machine testing.

## Verified baseline and commands

At ec72ee9, 88 top-level Go tests were listed (including environment-dependent and opt-in skips), 57 distinct PowerShell tests. The Go suite, vet, mod verify, govulncheck and dev build passed locally. Real pinned npm artifact test for 0.3.4 passed in 35.48 seconds with --ignore-scripts; the fixture installation remained unchanged. No full production install or authentication was tested. Local read-only snapshot reported three Remote processes, one eligible cache and empty recovery inventory in the earlier environment; refresh instead of assuming these values.

Hosted CI verified at ec72ee9:
- Go TUI: https://github.com/melloxyz/DCR-Manager/actions/runs/45434038675
- Windows tests: https://github.com/melloxyz/DCR-Manager/actions/runs/45434038446

From the repository in Windows PowerShell:

```powershell
git status --short --branch
git log -5 --oneline
git fetch origin --prune
go version
go test ./... -count=1
go vet ./...
go mod verify
go run golang.org/x/vuln/cmd/govulncheck@v1.10.0 ./...
powershell.exe -NoProfile -ExecutionPolicy Bypass -File tests/All.Tests.ps1
pwsh.exe -NoProfile -ExecutionPolicy Bypass -File tests/All.Tests.ps1
go build -trimpath -ldflags '-s -w' -o bin/dcr-manager-v14-dev.exe ./cmd/dcr-manager
.\bin\dcr-manager-v14-dev.exe --version
.\bin\dcr-manager-v14-dev.exe --snapshot
git diff --check
```

Optional real-artifact test (network, isolated, no lifecycle scripts):
```powershell
$env:DCR_NATIVE_ARTIFACT_TEST='1'
try { go test ./internal/backend -run '^TestNativeArtifactAcceptance$' -count=1 -v }
finally { Remove-Item Env:DCR_NATIVE_ARTIFACT_TEST -ErrorAction SilentlyContinue }
```

Do not paste raw snapshot, process arguments, npm logs, credentials or local identifying paths into public issues. Report safe summaries. Do not overwrite bin/dcr-manager.exe (stable v1.3.1) when compiling development. No false test results: list skipped tests and distinguish actual hosted/local evidence. Clean-profile real lifecycle, update, restore, cache, Windows sign-in, visual interactive and distribution acceptance remain pending.

## Handoff maintenance

Update this file after each substantive milestone, with commit(s), CI results, outstanding risks and next step. Keep technical detail in V14-MIGRATION.md and user-facing milestones in ROADMAP.md. When docs conflict, current source and verified latest evidence take precedence. Keep the working tree clean or explicitly document uncommitted changes.
