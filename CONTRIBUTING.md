# Contributing

DCR Manager is an independent Windows utility. The source is in pre-publication preparation; contribution and redistribution terms become effective once the project license is adopted.

## Getting started

Use Windows AMD64, Git, the Go version declared in go.mod, Windows PowerShell 5.1 and PowerShell 7. Node.js/npm are needed for lifecycle fixtures and package operations; Python 3.11+ is optional for TOML tests.

Read AGENTS.md, docs/ARCHITECTURE.md, docs/ROADMAP.md and docs/RELEASE-CHECKLIST.md. Create a focused branch from the intended development base. Keep commits small and use Conventional Commits. Do not fabricate tags or amend published history.

```powershell
go mod verify
go test ./... -count=1
go vet ./...
.\Test.cmd
pwsh -NoProfile -File tests/All.Tests.ps1
.\Build-TUI.cmd
```

Run gofmt on changed Go code. Use the existing Graphite Dark components, English UI copy and explicit terminal cell backgrounds. Add regression tests for changes to process detection, package transactions, recovery, input, confirmation or settings. Never test against a live Remote connection or user startup entry; use fixtures or disposable Windows profiles/VMs.

## Issues and pull requests

Describe the behavior, expected result, Windows/terminal/runtime versions and reproduction. Include test results and screenshots only after removing private information. Keep bug reports free of credentials, device IDs, raw logs, personal paths, agent configs or account information. For vulnerabilities, follow SECURITY.md rather than opening a public issue.

PRs should explain scope, compatibility, risk, validation and rollback. Update docs and CHANGELOG when behavior changes. No new runtime dependencies, network calls, telemetry, privileges or automated mutations without justification. Avoid unrelated refactors.

Maintainers review changes before merge. A passing local test suite does not imply a stable release. Public issue/PR workflows begin only after the repository is published.
