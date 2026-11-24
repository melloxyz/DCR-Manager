# Changelog

## 1.3.0
- Promote the maintainer-approved RC2 source to the v1.3.0 baseline on main.
- Preserve MIT licensing, the patched Go toolchain, Graphite Dark and English-only scope.
- Keep legacy PowerShell v1.2.1 available as fallback.
- Separate source milestone from public binary distribution; outstanding real-world acceptance checks move into v1.4 distribution gates.

## 1.3.0-rc.2
- Require patched Go 1.26.11; add govulncheck to CI.
- Prepare English public documentation, contribution/security policies, issue templates and dependency notices.
- Add optional real npm artifact acceptance test with isolated state and lifecycle scripts disabled.
- Separate source publication gates from stable binary release.
- Adopt MIT and include runtime third-party notices in candidate archives.

## 1.3.0-rc.1
- Go/Charm v2 Graphite Dark TUI with keyboard/mouse navigation and local health dashboard.
- Native Windows lifecycle controls with PID/creation identity and retained process handles.
- Staged package repair/install/update, selective npm cache quarantine, validated restore and transaction review.
- Bounded MCP audit/npm history, advisory troubleshooting.
- English first-run guidance, persistent landing page and opt-in Windows startup registration.
- Acceptance hardening: fail closed on stale diagnostics, complete confirmation visibility, explicit unknown cache state and executable-relative bundle discovery.
- Local preview packaging with per-file hashes and archive checksum; stable v1.2.1 remains unchanged pending acceptance.

## 1.2.1
- Hardened rollback with no overwrite, failure aggregation, real file-lock and fault injection tests.
- Cache restoration, parent-process diagnosis, non-destructive backup review, optional Codex TOML and explicit custom config audit.

## 1.2.0
- Read-only dedicated/global/npx installation inventory, JSON client audit, basic Node engine assessment, staged package backup restore, eight additional fixture tests.

## 1.1.0
- Modular architecture and isolated fixture tests.
- Portable npm-cache discovery and process inventory.
- Restart, diagnostic JSON, backup listing, English UI, Windows CI configuration.
