# Development handoff

Last reviewed: 2026-11-25. This is the continuation entry point for the v1.4.0 development branch. Read actual source and Git status before making changes; historical observations here are not live machine state.

## Repository and version policy

- Official repository: https://github.com/melloxyz/DCR-Manager
- Stable source: main and annotated tag v1.3.1, commit 4ab08254ee7f0bce9eb881f6f2208685ffff8478.
- Active development: feat/v1.4-native-backend; functional D.3 baseline b17c62a45495001639775427aa3b2afc39f4d612. Later documentation commits may advance HEAD.
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
| V14-01D.1b | Complete: isolated native switch, journal and initial rollback fixtures; live activation disabled |
| V14-01D.2 | Complete: isolated native cache quarantine and compatible sidecars; live activation disabled |
| V14-01D.3 | Complete: isolated native package/cache restore and legacy parity; live activation disabled |
| V14-01D.4 | NEXT: shared mutex and fault/recovery hardening |
| V14-01D.5 | Pending: integration, guard parity and replacement of legacy maintenance |
| V14-01E | Pending: full backend acceptance |

V14-02 portable executable/bundle; V14-03 release engineering and notices; V14-04 distribution documentation; V14-05 clean Windows acceptance; V14-06 public release. V2.0 is exploratory. Do not jump ahead or mark v1.4 stable prematurely.

## Architecture and current contracts

- cmd/dcr-manager: CLI, snapshot/settings, render and login entry points. internal/app: typed contracts, diagnostic rules, version; internal/tui: Bubble Tea v2, Lip Gloss v2, Bubbles viewport. Nine pages: Overview, Connection, Packages, Processes, Cache, Recovery, MCP Clients, Logs, Settings.
- internal/backend/reader.go uses bridge schema 3 NativeInventory. Go reads logs, all seven known MCP configs, runtime, npm directories, installs, cache, recovery and process health. Bridge still provides independent status, token, process and recovery parity. Mismatches fail closed; never remove this guard simply to make a test green.
- native_diagnostics*.go, native_toml.go, native_runtime*.go: bounded, sanitized reads/probes. native_inventory*.go, inventory_time*.go: path validation and cache tokens. native_recovery*.go: backup measurement, stage classification, tokens. native_health*.go, native_policy.go, native_windows.go: exact entry classification, process creation identity, native handle lifecycle, parent assessment and preflight.
- legacy_env*.go isolates Windows PowerShell child module discovery from inherited PSModulePath. Hosted failures proved the runner used 8.3 path aliases; GetLongPathName with same-object verification now normalizes legacy token identity. Keep both full-token cross-backend tests.
- native_stage*.go: prepareNativePackage validates stage roots, ownership, expected version, process guard before/after, random stage name, pinned repair or latest install/update, exact manifest/lockfile/version and regular entry. runNativeArtifactStage uses Node/npm-cli.js without shell assembly, isolated npm configuration/cache, fixed args, ignore-scripts and deadline. No activation; stage is not yet wired to Maintain/TUI. Failed stages are preserved.
- native_switch*.go: isolated switch consumes the prepared stage identity; validates complete stage and tracked trees, expected versions and process guard; moves only three package artifacts with no-replace Windows write-through moves. Schema-1 journal, unique backup, independent reverse rollback and final tree verification preserve evidence. Windows file IDs are captured before moving paths. See V14-MIGRATION.md for crash windows. Native/legacy recovery parity also covers incomplete rollback backups without ownership manifests, classified unsafe by both engines.
- native_quarantine*.go: isolated cache quarantine selects only the exact `_npx` ID/token, validates exclusive ownership and full-tree safety, repeats process guards and identity/token checks, writes a synced compatible sidecar and renames the cache without replacement. Post-verification preserves evidence on uncertainty. No automatic rollback/delete or production wiring. Ready/incomplete/destination-blocked recovery status and restore-token parity pass against PowerShell; the legacy restore target matches the original cache path.
- native_restore*.go: isolated restore checks exact recovery token, source identity, path/volume safety and source/destination guards. Package restore copies only tracked artifacts into a random restore stage, compares source/staged contents with SHA-256, preserves the original backup and reuses the journaled switch/rollback. Cache restore uses the sidecar's vacant original target and a no-replace directory move; sidecar evidence remains. Legacy regular lockfiles are accepted only for restore; prepared npm staging keeps the stricter v2+ structure validation. No production wiring.
- maintenance.go still delegates mutations to src/MaintenanceAction.ps1 after independent Go preflight. src/PackageCache.psm1, src/Core.psm1 (Switch-DcrPackageFiles), src/RecoveryCenter.psm1, src/Recovery.psm1 are the migration reference. Do not change the legacy production contract without parity evidence.
- Dedicated install is %LOCALAPPDATA%/DesktopCommander; state is %LOCALAPPDATA%/DCRManager-state; preferences %APPDATA%/DCRManager. Never publish contents. Package transaction may move only node_modules, package.json and package-lock.json. Never move Remote credential directory, shims, unrelated files or arbitrary npm cache.
- Native lifecycle and legacy transactions use Local\\DCRManager Windows mutex. Native process preflight is not a substitute for in-mutex all-process/path checks. PID + creation time, not PID alone. Missing parents are advisory, not proof of orphaning. No real Remote stop/restart, registry startup write or client configuration edits during tests.
- Keep paths absolute, reject reparse points/unsupported layouts, use same-volume no-overwrite moves, preserve backup and journal evidence; do not claim race-free or cryptographic full-tree authenticity.

## Next work: D.4

2026-11-25 review: V14-02 explicitly includes an approved multi-resolution application icon embedded in the Windows executable, reproducible build inclusion and Explorer/shortcut verification. Icon artwork and executable resource embedding remain planned. The following authorized continuation implemented isolated D.3 restore; D.4 is now the next functional stage.

D.1b is implemented internally using TDD and temporary fixtures. Tests cover fresh install and repair/update, credential/unrelated-file preservation, all six forward and all six reverse move failure boundaries, real locked files, occupied/replaced targets, stale inputs, reparse paths, journal failures and cancellation. Native/legacy inventory and restore-token parity pass for completed, failed and incomplete rollback transactions. No active installation was switched.

D.2 is implemented internally using TDD and temporary fixtures. It preserves unselected caches, rejects invalid/stale/linked/mixed candidates and occupied evidence, handles early/late guards and cancellation, and detects post-move manifest/version/tree/sidecar changes. Real Windows directory locking and native/legacy recovery parity passed. Missing npm lockfiles remain supported; incomplete entry points stay non-restorable. Sidecars and failed writes remain evidence; no real cache or active installation was changed.

D.3 is implemented internally using TDD and temporary fixtures. Restore preserves the package source backup/token and untracked source files, checks copied file contents, backs up the replaced installation and reuses independent journaled rollback. Cache restore renames only the validated backup to its vacant original `_npx` target and retains the sidecar. Legacy lockfiles, absent/unknown current versions, stale/linked/mixed inputs, copy/move/journal failures, all forward/reverse move boundaries, cancellation and real Windows locks are covered. Native/legacy recovery and restored-cache token parity passed locally.

Next implement D.4 shared mutex and fault/recovery hardening. Read the actual native lifecycle lock and legacy maintenance lock, process/path guards and interrupted-transaction contracts first. Serialize participating native operations with the existing manager mutex, repeat independent all-process/path/token checks inside the lock and preserve evidence on uncertainty. Continue TDD, injected operations and disposable fixtures; do not stop the active Remote or test against real installations, caches, credentials or preferences. Race/crash acceptance must be explicit; D.5 production integration and V14-01E/full backend acceptance remain separate gates.

Keep Maintain wired to PowerShell until D.4/D.5 shared mutex, all-process guards, recovery and production npm lifecycle acceptance pass. Lockfile validation is structural; tracked tree identity/size/time comparisons do not prove cryptographic authenticity. The switch scans complete stage trees, but staging alone does not. No native shared mutex or lifecycle scripts are enabled. Alternate npm launchers, trusted npm configuration, real junction races, power-loss durability and crash replay need disposable-machine validation. Failed stages, journal temporary files and backups are deliberately retained. No main merge, stable tag change or public release is authorized.

## Verified baseline and commands

D.3 local acceptance (2026-11-25): `go test ./... -json -count=1` passed 272 test/subtest events (121 top-level tests), with two top-level skips: 8.3 fixture alias unavailable and real npm artifact network opt-in absent. Windows PowerShell and PowerShell 7 each passed all 57 fixture tests. Vet, module verification, govulncheck v1.10.0, diff checks and the separate dev build passed; executable reports `1.4.0-dev`. All restore mutations targeted temporary fixtures. Prepared-stage regression tests still enforce the stricter lockfile contract.

Hosted D.3 acceptance at b17c62a:
- Go TUI passed: https://github.com/melloxyz/DCR-Manager/actions/runs/45650012098 (module verification, tests, vet, govulncheck and build).
- Windows tests passed: https://github.com/melloxyz/DCR-Manager/actions/runs/45650012053 (Windows PowerShell 5.1 and PowerShell 7).
- The implementation was committed conventionally and pushed normally to the feature branch. main, origin/main and peeled v1.3.1 remained at 4ab08254ee7f0bce9eb881f6f2208685ffff8478. No production maintenance route, stable tag, real credentials/preferences or active Remote was changed.

D.2 local acceptance (2026-11-24): `go test ./... -json -count=1` passed 192 test/subtest events (108 top-level tests), with two top-level skips: 8.3 fixture alias unavailable and real npm artifact network opt-in absent. Windows PowerShell and PowerShell 7 each passed all 57 fixture tests. Vet, module verification, govulncheck v1.10.0, diff checks and the separate dev build passed; executable reports `1.4.0-dev`. The active Remote, actual caches, credentials, client preferences, startup and UI were not modified. Shared mutex, power-loss/race testing, full-tree cryptographic integrity and production acceptance remain unverified.

D.2 implementation commit: 3cfb7ec2563ccfefd130b209286aa0f0136b57f4. Initial hosted Windows tests passed (45462535507). Initial Go TUI run (45462535511) exposed a test-only lexical restore-path comparison: status/version/size/token parity passed, but equivalent Windows path spellings were compared as strings. Test fix 94b3f99 uses validated canonical paths (existing parent identity checked by `canonicalStageRoot`) and also exercises uppercase spelling. Exact restore-token equality remains unchanged. Targeted local parity tests passed after this correction.

Hosted D.2 acceptance at 94b3f99:
- Go TUI passed: https://github.com/melloxyz/DCR-Manager/actions/runs/45462972640 (module verification, tests, vet, govulncheck and build).
- Windows tests passed: https://github.com/melloxyz/DCR-Manager/actions/runs/45462972460 (Windows PowerShell 5.1 and PowerShell 7).
- Implementation and test fix were committed conventionally and pushed normally to the feature branch. main, origin/main and peeled v1.3.1 remain at 4ab08254ee7f0bce9eb881f6f2208685ffff8478. No stable tag, public history or real user configuration changed.

D.1b local acceptance (2026-11-24): `go test ./... -json -count=1` passed 147 test/subtest events (99 top-level tests); two top-level skips: fixture volume did not provide an 8.3 alias, and the real npm artifact test requires explicit network opt-in. Windows PowerShell and PowerShell 7 each passed all 57 fixture tests. `go vet ./...`, `go mod verify`, govulncheck v1.10.0, `git diff --check` and the separate dev build passed; executable reports `1.4.0-dev`. The artifact download, active Remote lifecycle, real credentials/preferences, interactive UI and disposable-machine crash/race acceptance were not exercised.

Hosted D.1b acceptance at implementation commit 9ae1c79:
- Go TUI passed: https://github.com/melloxyz/DCR-Manager/actions/runs/45442116558 (module verification, tests, vet, govulncheck and build).
- Windows tests passed: https://github.com/melloxyz/DCR-Manager/actions/runs/45442116750 (Windows PowerShell 5.1 and PowerShell 7).
- Implementation was committed with Conventional Commits and pushed normally to the feature branch. main, origin/main and the peeled v1.3.1 tag remained at 4ab08254ee7f0bce9eb881f6f2208685ffff8478. No public history was rewritten.

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
