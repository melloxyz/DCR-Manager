# V14-01 native backend migration

Goal: remove PowerShell/Python from the Go application's runtime path without weakening safety, privacy or recoverability. The legacy PowerShell application remains available until separately retired. Windows AMD64 and English-only Graphite Dark remain the supported development target. Node.js/npm are upstream prerequisites; portable does not mean bundling DCR or Node.

| Stage | Scope | State |
| --- | --- | --- |
| V14-01A | Native bounded npm history and six JSON MCP locations | Implemented; local parity and regression |
| V14-01B | Native TOML parsing, runtime/version and npm command discovery | Implemented; local parity and regression |
| V14-01C | Native installation/cache/process health/recovery inventory with independent process guard parity | Implemented locally; hosted token parity pending |
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


## V14-01C incremental inventory migration

| Substage | Scope | State |
| --- | --- | --- |
| V14-01C.1 | Native npm directory discovery, installation and cache inventory | Implemented; local parity and regression |
| V14-01C.2 | Native recovery inventory, transaction journals and restore-token parity | Implemented; local parity and regression; hosted parity pending |
| V14-01C.3 | Native process health and independent mutation-guard parity | Implemented; local parity and regression |

C.1 obtains npm cache and global node_modules paths through fixed read-only npm commands. The command runner retains the caller working directory to match existing maintenance behavior, uses the system cmd.exe and fixed arguments, bounded output and deadlines. Discovery requires absolute valid paths. Go inspects dedicated/global installations and exclusive npx cache directories using bounded manifest reads, path/identity checks and fixed output classifications. The inventory is not an exhaustive scan of all possible Node/npm installations. The cache scan has a cooperative five-second deadline and a 10,000-directory-entry cap. Extra dependency/workspace declarations are conservatively rejected. Unknown/unsupported layouts may differ from the permissive historical inventory. A dedicated status mismatch rejects the snapshot rather than displaying unsafe confirmation data.

Cache tokens reproduce the legacy SHA-256 contract: lowercased path, Windows creation time converted to .NET ticks, uppercase SHA-256 of the raw top-level manifest and package version. Tokens are identity checks, not cryptographic authenticity or full-tree integrity. Until the mutation engine is migrated, native cache rows become actionable only if the legacy cache root, candidate count and exact tokens agree. Disagreement marks cache unknown and removes actionable candidates. Cache restore is blocked when roots disagree. Existing PowerShell cache status/validation and recovery/process inspection therefore remain in the bridge; native installation rows replace the old installation scan. Schema 3 NativeInventory is added without changing default, schema 1 or schema 2 behavior.

Tests cover exclusive versus mixed cache fixtures, dedicated/global/cache inventories, changed manifest tokens, missing/unknown and cancelled scans, fail-closed parity and exact token equality against the legacy PowerShell implementation in an isolated Windows fixture. Local read-only comparison found one dedicated and one legacy npx installation, one eligible cache with the same token, and three active Remote processes. Native and legacy installation rows agreed. No active installation, cache, backup or process was modified. The new native implementation still uses PowerShell for remaining inventory/guard checks and all mutations. Clean-machine and reparse/junction acceptance remain pending.


## V14-01C.2: Native recovery inventory

The Go reader now scans the known state directory for exact package-backup, cache-backup, stage and restore-stage names. Native inventory uses a shared cooperative eight-second deadline, at most 10,000 top-level directory entries, at most 100,000 tree entries per item, bounded reads of manifests/metadata/journals, and ancestor/reparse/identity checks. Unreadable or unsafe items cannot be restored. Retention is advisory at 30 days. Transaction journals must have schema 1 and known state values; stages never get restore tokens. Logical byte counts and entry counts are not a full cryptographic content audit.

Package backups require exclusive package ownership, an exact pinned version, a regular entry point and a lockfile. Cache backups require valid sidecar metadata identifying the same backup, an empty direct hex child of the current npm _npx directory and a matching volume. SHA-256 tokens reproduce the legacy contract using lowercased path, Windows creation ticks, modification ticks, byte and entry counts, version, manifest hashes and cache metadata hash. Native tokens must match the independent legacy ready row before restoration is enabled. Missing rows, incomplete inventory, root disagreement or token mismatch fail closed. The legacy bridge still performs independent recovery/cache inspection and all mutations. No actual backup files or credentials are modified by the new reader.

Local validation: isolated package and cache backup fixtures match legacy restore tokens exactly; stale manifest tokens, occupied destinations, retention review, cancelled scans and transaction state are covered. The real state inventory was empty in both implementations. The previous C.1 hosted Go job failed TestCacheTokenLegacyParity despite local success; the fixture test now uses UTF-8 output, suppresses PowerShell warnings and separately reports parsing/count/identity mismatches. This improves diagnosis, not evidence of a confirmed root cause. Hosted CI must pass before C.1/C.2 acceptance is closed. Symlink privilege-dependent test and clean-profile acceptance remain pending.


## V14-01C.3: Native process inventory and health

The Go reader now constructs DCR process rows, Remote/MCP roles, location and root counts from native Node command-line inspection and verified creation identities. Parent assessment uses an independent Toolhelp process snapshot and limited-information handles for relevant parent PIDs. Missing parents, reused PIDs and inaccessible/unknown creation times are distinct advisory states; they do not prove an orphan or justify termination. No raw arguments are serialized.

Until C/E acceptance is complete, compare the entire native DCR process set (PID, parent, role, location, counts) with the independent legacy inventory, not only Remote PIDs. On mismatch or inspection error preserve the legacy display but clear process identities and refuse controls. The native assessment is used only after successful inventory reconciliation. A failed process wait/empty command line also fails closed. Actual process stop/start logic retains the existing native handle and identity checks.

The Go maintenance adapter performs an additional read-only native Node-process preflight before delegating to the PowerShell transaction. Package and restore requests refuse active DCR references; cache quarantine refuses references to the selected cache path. Inaccessible or uninspectable Node processes block maintenance. This preflight does not replace the legacy in-mutex guard, which checks broader process references, validates paths/tokens and repeats checks during the transaction. No native mutation engine is enabled yet. Native classification deliberately supports known exact entry scripts, not arbitrary launchers. External races remain possible.

Local validation: all three active Remote rows and their parent assessments matched the legacy snapshot (one Remote root, zero separate MCP processes, two present parents and one missing parent). No actual process was stopped. Tests cover role/location classification, parent PID reuse and unknown parent times, full-set reconciliation and conservative native maintenance preflight. Hosted C.1/C.2 parity remains a release gate: removing inherited PSModulePath allowed legacy fixture rows to be produced on the runner but the tokens still differ. Additional diagnostics report component equality without exposing private paths. Do not interpret local agreement as hosted acceptance.

## Hosted token parity investigation

The first hosted failures were not token mismatches: the legacy cache scan yielded no rows and recovery validation returned unsafe. Removing the inherited PowerShell 7 PSModulePath from Windows PowerShell child processes allowed both fixture types to be read in the runner. This isolation now applies only to the legacy child environment in the application as well as tests; it does not modify the user's shell or global configuration.

The next hosted run proved that the cache fixture path differed (short 8.3 alias versus expanded Windows path), while creation ticks and the raw manifest hash agreed. Native cache/recovery tokens now use GetLongPathName and verify the resolved path still identifies the same non-reparse filesystem object. Failure to resolve is fail-closed. The existing tests still compare complete legacy tokens; a short-alias regression fixture is added when the volume supports it. Local parity passes; hosted confirmation of the fix is required before closing the C acceptance gate.
