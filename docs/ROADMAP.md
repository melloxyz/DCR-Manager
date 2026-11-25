# Roadmap

This is the public product roadmap. Implemented means code exists and has fixture/local validation; it does not imply full real-world acceptance. Dates are not promised. The first actual Git baseline is v1.2.1; earlier milestones are historical development, not reconstructed tags.

| Milestone | Focus | Status |
| --- | --- | --- |
| v1.1.0 | Structural foundation | Historical |
| v1.2.x | Advanced management and reliability | Stable legacy baseline v1.2.1 |
| v1.3.0 | User experience and TUI | Maintainer-approved source baseline |
| v1.4.0 | Distribution and portability | In development |
| v2.0.0 | Complete management toolkit | Exploratory |

## v1.1.0: Structural foundation

Modular architecture, permanent installation, process management, selective diagnosis/repair, automated tests and Windows CI configuration. Hosted CI execution and CD are not claimed.

## v1.2.0: Advanced management

Global/local installation inventory, read-only MCP audits, advisory parent-process assessment (not proof of orphaning), rollback attempts, backup/retention review and basic Node/npm compatibility. Complex engine ranges and automated retention deletion are outside the validated scope.

## v1.3.0: User experience

Keyboard/mouse TUI, Graphite Dark, local system health, advisory troubleshooting, guided first-run setup and opt-in Windows startup. English only by maintainer decision; other languages may be evaluated later. Go migration was brought forward from v1.4.

| Stage | Deliverable | Status |
| --- | --- | --- |
| T1-T4 | Go foundation, design, navigation, health | Implemented |
| T5 | Native Windows lifecycle | Implemented; real upstream reconnect pending |
| T6 | Package/cache maintenance | Implemented; full upstream acceptance pending |
| T7 | Recovery and retention review | Implemented; full upstream acceptance pending |
| T8 | Diagnostics and troubleshooting | Implemented |
| T9 | Settings, English onboarding, startup | Implemented; actual Windows sign-in pending |
| T10 | Stabilization and local candidate | RC1 and RC2 built locally |
| T10.1 | Acceptance and public-source preparation | Source preparation and local artifact tests complete; MIT adopted |
| T10.2 | Fix acceptance findings, candidate rebuild | RC2 completed |
| T10.3 | Stable v1.3.0 promotion | Approved by maintainer; source baseline promoted |

The v1.3.0 source milestone is approved. Remaining independent real-world acceptance evidence is tracked as a v1.4 distribution gate; no public binary release is implied.

## v1.4.0: Distribution

- V14-01 (in progress): Native backend migration. Read-only npm logs, seven MCP JSON/TOML locations and runtime version probes migrated; installation/cache discovery and inventory migrated with legacy cache-token parity checks; recovery inventory migrated with legacy restore-token checks; process inventory and health migrated with independent legacy reconciliation; hosted token parity passed. Native package staging, transaction switching, initial rollback and cache quarantine are implemented internally with isolated fixtures. Backup restore, shared mutex hardening and production integration remain. See [migration plan](V14-MIGRATION.md).
- V14-02: Portable Windows AMD64 executable or self-contained bundle; explicit external prerequisites. Planned: application icon embedded in the Windows executable using a multi-resolution `.ico`, reproducible inclusion in the build, and visual verification in Explorer and shortcuts. Artwork approval and executable embedding remain pending.
- V14-03: Reproducible build/release pipeline, license/notice packaging, checksums and provenance.
- V14-04: Public documentation, installation/uninstallation, version support and contributor workflow.
- V14-05: Clean Windows VM acceptance, real lifecycle, package/cache/restore and sign-in regression.
- V14-06: Public prerelease and stable release after acceptance; review code signing and distribution channels.

Open-source licensing is now an earlier source-publication prerequisite, rather than postponed until distribution. No date or additional platform support is promised.

## v2.0.0: Complete management toolkit

Manage agent integrations, migrate legacy installs, advanced connectivity diagnostics, session/log management and evaluate Linux/macOS support. These remain exploratory; existing read-only inventory does not imply full integration management.

## v1.3.1 maintenance
Dependency maintenance baseline before v1.4: four Dependabot updates, notice refresh, grouped compatible updates. No binary distribution implied.
