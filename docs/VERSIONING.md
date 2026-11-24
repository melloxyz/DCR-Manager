# Versioning and releases

Use SemVer, Conventional Commits, and annotated release tags. Keep main releasable; feature branches use feat/<topic>, fix/<topic>, or chore/<topic>. Merge only after tests and review. Version bumps update VERSION, README, UI banner, diagnostic report version and CHANGELOG together until version metadata is centralized. Use patch releases for fixes, minor releases for backward-compatible features, major releases for incompatible public contracts.

The initial Git baseline is v1.2.1: prior 1.0 through 1.2 milestones were developed without commits. Do not invent historical commits or tags. Changelog preserves earlier milestones. Tags record actual source snapshots; they are not certification of production readiness.

Before a public release: run both PowerShell suites, test installation/restart/repair on a clean Windows VM, review security, decide license and upstream branding, build reproducible artifacts and document checksums. No device state, credentials, logs or legacy local backups belong in Git. No remote or public release was created by baseline initialization.

The Go application version lives in internal/app/version.go; VERSION records the current product source milestone. Legacy Manager.ps1 and its report format retain their own v1.2.1 identity. RC tags identify candidate snapshots. A stable source tag records maintainer acceptance, not independent evidence of every clean-machine test. Public binaries require the distribution gates. Do not overwrite archives or move published tags.

Source publication may precede stable binary distribution once licensing and privacy gates are closed. Preserve the current local RC1 tag and archive; toolchain/security changes belong to a subsequent candidate, never overwrite an existing release artifact.
