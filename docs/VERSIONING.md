# Versioning and releases

Use SemVer, Conventional Commits, and annotated release tags. Keep main releasable; feature branches use feat/<topic>, fix/<topic>, or chore/<topic>. Merge only after tests and review. Version bumps update VERSION, README, UI banner, diagnostic report version and CHANGELOG together until version metadata is centralized. Use patch releases for fixes, minor releases for backward-compatible features, major releases for incompatible public contracts.

The initial Git baseline is v1.2.1: prior 1.0 through 1.2 milestones were developed without commits. Do not invent historical commits or tags. Changelog preserves earlier milestones. Tags record actual source snapshots; they are not certification of production readiness.

Before a public release: run both PowerShell suites, test installation/restart/repair on a clean Windows VM, review security, decide license and upstream branding, build reproducible artifacts and document checksums. No device state, credentials, logs or legacy local backups belong in Git. No remote or public release was created by baseline initialization.

The Go preview version lives in internal/app/version.go. During prerelease, VERSION and legacy report/banner still identify the stable v1.2.1 implementation; do not label it as 1.3.0. RC tags (v1.3.0-rc.N) identify local candidate snapshots, not stable release acceptance. Package-Preview.ps1 builds from a clean tree, validates an extracted copy and writes a checksum. Do not overwrite candidate archives or move existing tags. Merge and stable tag require acceptance; no public publishing in T10.
