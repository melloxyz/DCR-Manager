# DCR Manager 1.3 release candidate

This is a local Windows AMD64 preview, not a public release or a standalone binary. Extract the whole ZIP into a trusted directory and use Launch-TUI.cmd. The executable needs the adjacent src directory, Windows PowerShell 5.1 and Node.js/npm for Desktop Commander operations. Python 3.11+ is optional for TOML inspection. Do not run a bundle from an untrusted directory. No administrator rights are requested. The bundle excludes local settings, credentials, backups and logs. No upstream software is bundled.

The stable v1.2.1 legacy release is unchanged. English-only, Graphite Dark. To build a local candidate, use Package-Preview.ps1 from a clean committed development tree. The archive contains a file hash manifest and has a SHA256 sidecar; hashes detect corruption, not publisher authenticity. No code signing or public release. The preview is not a replacement for a clean Windows acceptance environment.

## Candidate checks
- Go unit/integration tests, go vet, go mod verify, PowerShell 5.1/7 regression.
- Visual cell dimensions, opaque palette, confirmation visibility and stale snapshot behavior.
- Extracted archive hashes, version and read-only diagnostic smoke test.
- Check actual interactive TUI at 110x35 and smaller viewport sizes: all nine pages, keyboard/mouse, focus, scrolling, confirmation, setup, help and errors.
- Verify real npm install/repair/update on disposable Windows with a separate user profile.
- Verify real Remote start/stop/restart and authentication/reconnection in that profile; never interrupt the maintainer's active session.
- Verify real cache quarantine/restore and package rollback, credentials preservation and actual Windows sign-in behavior.
- Check performance on representative hosts, dependency vulnerability scanning, hosted CI, release licensing/branding and distribution documentation.
- Only after acceptance: promote version and merge main; public distribution remains v1.4.0.
