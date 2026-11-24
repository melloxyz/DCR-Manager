# Licensing decision and attribution

## Project license (pending)

Recommended: MIT, for a small community utility that aims to allow use, modification and redistribution with attribution and a warranty disclaimer. The maintainer must explicitly adopt the license and identify the appropriate copyright holder before publishing as open-source. No root LICENSE has been added without that decision.

Alternatives: Apache-2.0 (explicit patent grant and additional notice requirements), GPL-3.0 (copyleft). Do not assume the upstream license automatically licenses this independent project.

## Upstream

Desktop Commander is a separate project: https://github.com/wonderwhy-er/DesktopCommanderMCP . Its repository declares MIT. DCR Manager is an independent helper and does not bundle the upstream npm package in the preview. The upstream name is used descriptively, not to imply endorsement. Review branding and upstream package terms before publication.

## Third-party material

The Go binary incorporates third-party Go dependencies. Preserve their license and notice texts in future distributed artifacts. Project-scoped development skills have pinned provenance and upstream LICENSE files under .agents/skills. Preserve those notices and review any embedded examples. See THIRD_PARTY_NOTICES.md. The RC1 ZIP was built before this notice collection; do not distribute it as a public artifact.
