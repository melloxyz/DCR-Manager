# DCR Manager

An independent, Windows-first terminal utility for managing Desktop Commander Remote (DCR). Built with Go and Charm v2, with a Graphite Dark interface and a tested PowerShell maintenance backend.

**Development status:** v1.3.1 is the maintenance source baseline on main. The PowerShell fallback remains v1.2.1. Public binary distribution and full clean-machine validation are planned for v1.4.0. This is an open-source work in progress; release and publication gates are tracked in [the release checklist](docs/RELEASE-CHECKLIST.md) and [licensing notes](docs/LICENSING.md).

DCR Manager is not an official Desktop Commander product and is not affiliated with its maintainers. See the [upstream project](https://github.com/wonderwhy-er/DesktopCommanderMCP) for the actual Remote service and authentication.

## What it does

| Module | Capabilities |
| --- | --- |
| Overview | Local health, runtime, installation and process inventory |
| Connection | Start, stop and restart recognized Remote processes |
| Packages | Dedicated npm installation, staged repair and explicit update |
| Processes | Inspect recognized DCR process identities and parent relationships |
| Cache | Selective quarantine of exclusive DCR npm cache directories |
| Recovery | Validated backup restore, transaction review and advisory retention |
| MCP Clients | Read-only audit of known user configuration locations |
| Logs | Bounded, sanitized npm history and troubleshooting guidance |
| Settings | First-run guidance, landing page and opt-in Windows startup |

## Requirements

- Windows AMD64, Windows PowerShell 5.1, a terminal with Unicode and truecolor support (Windows Terminal recommended; 110x35 or larger).
- Node.js and npm for Desktop Commander installation and lifecycle operations. The upstream package determines its supported Node.js versions.
- Python 3.11+ with the Windows py launcher is optional for the legacy PowerShell TOML audit only; the v1.4 development Go application parses TOML natively.
- Go, Git and PowerShell 7 for development and full regression testing. Use the Go version declared in go.mod.

No administrator privileges are requested. The preview is a **multi-file bundle**, not a standalone executable.

## Run from source

From the repository directory on Windows:

```powershell
.\Build-TUI.cmd
.\Launch-TUI.cmd
```

Keep the executable and its adjacent src directory together. First-run setup guides users through prerequisites, confirmed installation and explicit Remote startup. No automatic installation, authentication or startup registration occurs.

For read-only diagnostics:

```powershell
.\bin\dcr-manager.exe --version
.\bin\dcr-manager.exe --snapshot
.\bin\dcr-manager.exe --settings
```

Review diagnostic output before sharing it. Local process IDs and timestamps may be present. The legacy PowerShell manager remains accessible through Launch.cmd.

## Navigation

- 1-9: select a module; arrows navigate; Tab changes focus.
- Page Up/Down: scroll; R refreshes; H opens troubleshooting; Q exits.
- Follow the module-specific key hints. Y confirms sensitive operations; N or Esc cancels.

## Safety and privacy

The dedicated installation lives under %LOCALAPPDATA%\DesktopCommander; maintenance state under %LOCALAPPDATA%\DCRManager-state; preferences under %APPDATA%\DCRManager. These are not repository files.

The manager does not implement telemetry, globally wipe the npm cache, kill arbitrary Node.js processes or rewrite agent configurations. Package changes are staged and preserve the Remote credential directory. Stop affected DCR/MCP clients before changing packages. Backups are retained; automatic cleanup is not enabled. The current implementation performs forced process termination for explicit stop/restart. Local process status does not prove Remote server connectivity. The upstream service and npm have their own network behavior. The PowerShell bridge uses process-scoped ExecutionPolicy Bypass.

Do not use the preview on a critical installation without backups. Structural backup validation is not full cryptographic authentication. External filesystem/process races and unexpected shutdowns remain possible. See [recovery](docs/RECOVERY.md), [security](SECURITY.md), and [release readiness](docs/RELEASE-CHECKLIST.md).

## Development and community

- [Roadmap](docs/ROADMAP.md): product milestones, scope and outstanding work.
- [Architecture](docs/ARCHITECTURE.md): components, contracts and trust boundaries.
- [Contributing](CONTRIBUTING.md): local setup, testing and pull requests.
- [Changelog](CHANGELOG.md): actual version history.
- [Third-party notices](THIRD_PARTY_NOTICES.md): dependencies and vendored development references.

Run go test ./..., go vet ./..., go mod verify, Test.cmd and the PowerShell 7 suite before submitting changes. See [versioning](docs/VERSIONING.md). The existing release-candidate ZIP is a local test artifact, not an endorsed public download. The initial hosted Go and Windows test workflows passed. Public binaries are not yet released.

## License

[MIT License](LICENSE). Third-party components retain their respective licenses; see [licensing notes](docs/LICENSING.md).
