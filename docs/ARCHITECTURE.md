# Architecture

## Layering

- cmd/dcr-manager: CLI dispatch, read-only snapshot and settings output, TUI and guarded login startup.
- internal/app: typed snapshots, settings/maintenance/lifecycle contracts, advisory diagnostic rules and version metadata.
- internal/tui: Bubble Tea Model/Update/View, Lip Gloss components, Bubbles viewport, explicit ANSI backgrounds, keyboard/mouse navigation and confirmation flows.
- internal/backend: native Windows process discovery/identity/termination/start, settings persistence and HKCU startup registration, fixed-argument PowerShell adapters.
- src: legacy manager modules and shared staged package/cache/recovery transactions, bounded audit/log collection and optional Python TOML reader.
- tests and internal/**/*_test.go: isolated fixtures, policy tests and visual dimension regressions.

## Data flow

The TUI requests a typed snapshot through a background command. The diagnostic bridge reads local runtime, install, cache, recovery, MCP and npm information. Native process inventory independently verifies PID and creation time before enabling controls. Mutations are explicit user requests, run outside Update, and trigger a fresh diagnostic afterwards. Critical package operations use a shared manager mutex, staged replacement and rollback; lifecycle uses retained Windows handles. Settings use a separate mutex and compare expected previous values.

## State and trust

Runtime state is outside the repository. The executable resolves its scripts from a complete executable-relative bundle, never the arbitrary working directory. Scripts and Node/npm must come from trusted sources. Backups are structural, not cryptographically authenticated. Neither process inspection nor retained npm logs establishes live upstream connectivity. The UI does not call external AI services or send telemetry.

## Migration

Go/Charm was adopted in v1.3 ahead of the original v1.4 evaluation. The preview still requires PowerShell for package, cache, recovery and read-only diagnostics. v1.4 will assess native replacements or embedded safe execution, portable packaging, reproducible releases and clean Windows acceptance. Preserve regression parity and Graphite Dark while migrating.

The v1.4 development reader consumes a versioned internal bridge envelope for the remaining PowerShell fields. Go reads bounded npm log history and the six known JSON MCP files directly. The cache location is internal to the bridge exchange and is not serialized in public snapshots. Legacy invocation without the new switch retains v1.3 behavior.

Bridge schema 2 delegates all known client audits (including TOML) and Node/npm version probes to Go. Native Windows command discovery uses fixed runtime names and a constrained npm.cmd version invocation; cache/global path queries and remaining inventory are still in PowerShell. The old bridge invocation remains available for the stable legacy manager.

Bridge schema 3 delegates npm directory discovery and installation/cache inventory to Go. Native cache candidates must match legacy root, count and tokens before mutations are available. Recovery and process health still rely on the legacy bridge; the next migration steps retain independent guard parity.

Go now provides the recovery inventory and stage classification. Ready backup tokens must agree with independent legacy validation; cache destinations also depend on npm root agreement. The legacy bridge still performs recovery and cache cross-checks, process inventory and process health. No mutation is migrated in C.2.
