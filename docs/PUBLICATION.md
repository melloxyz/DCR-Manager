# Source publication checklist

Source publication and binary release are separate gates. A public repository can show work-in-progress before a stable binary release, once the source license and privacy review are resolved.

## Before publishing source

- Adopt the project license and copyright holder; retain third-party notices.
- Review author and committer identities in all reachable Git commits and annotated tags. A non-noreply commit email exists in the current history. Do not publish it inadvertently. Choose whether to preserve that history knowingly, prepare a sanitized public history, or publish a separate clean source snapshot. Never rewrite or move existing local release tags without explicit maintainer approval.
- Verify secret scanning and tracked paths, review potential findings; no runtime data or artifacts. The initial pattern scan was limited, not an exhaustive secret scanner.
- Review README, SECURITY.md, CONTRIBUTING.md, roadmap and dependency notices.
- Set GitHub repository description to independent community Windows TUI; enable Issues, private vulnerability reporting and branch protections if supported. CI requires read-only repository access, no production credentials.
- Do not attach local ZIPs, local logs, credentials or backup directories. No public releases during the v1.3 acceptance phase.
- Validate the public tree and branch/tag references before any push. No remote has been configured by this checklist.

## After source publication

Run hosted CI, enable dependency update and vulnerability alerts, review issue templates and confirm public docs render correctly. Keep the development branch visible and clearly mark prerelease status. Publish stable binaries only after the release checklist and licensing/notice packaging gates are completed.
