# Security policy

## Status

DCR Manager v1.3 is a development preview. No security support SLA or production readiness is promised. The stable legacy baseline is v1.2.1; both versions require careful local use. Use a trusted source directory and do not execute untrusted modified bundles.

## Reporting a vulnerability

Once the public repository exists, use its private vulnerability reporting feature if enabled. Until then, contact the maintainer privately through an existing trusted channel. Do not post proof-of-concept exploits, credentials, raw logs, user paths or Remote device/account identifiers in public issues. Include an affected version, sanitized reproduction, impact and suggested fix if available. Do not test on someone else's system.

## Trust boundaries

The manager runs as the current Windows user. It invokes installed Node.js/npm and local PowerShell scripts, inspects recognized processes and known config files, and moves selected package/cache files after confirmation. Upstream npm lifecycle scripts can execute with user privileges; only install trusted packages. Backups are not signed or encrypted by this tool. The manager cannot prevent all races with external processes, filesystem changes or abrupt power loss.

Process identity, stale snapshot guards, explicit confirmation, staged replacement and backup retention reduce risk; they do not provide a sandbox. No automatic global cache cleanup, agent configuration rewriting, or startup registration. The Remote service and credentials belong to upstream.

## Release checks

Run Go vulnerability analysis, dependency/license review, history/secret review, full fixture tests, isolated real-world acceptance and artifact hash checks. See docs/RELEASE-CHECKLIST.md and docs/PUBLICATION.md. The checks are not a security certification.
