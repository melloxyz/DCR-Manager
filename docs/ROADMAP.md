# Roadmap

## 1.1: modular Windows foundation
- Native PowerShell, dependency-free manager and local tests.
- Process inventory, targeted npm-cache discovery, staged repair, sanitized diagnostics.

## Next: hardening
- Mocked process start/stop, rollback fault-injection, real installation in disposable Windows VM, restart/authentication regression, cache path and reparse-point tests.
- Read-only inventory of global npm installations and client MCP config files with secret redaction; opt-in migration and backup/restore workflows.
- Detect stale sessions, connectivity checks based on documented upstream APIs only, managed logs and retention.
- Localization (pt-BR/en), keyboard navigation, accessibility and package validation.

## Distribution
- Decide license, upstream branding, support policy, threat model, release signing and reproducible ZIP artifacts. Do not publish existing machine data.
- Consider Go + Bubble Tea/Lip Gloss as a later compiled TUI if richer keyboard UI or cross-platform support justifies native OS adapters. Keep recovery functionality independent of Node.
