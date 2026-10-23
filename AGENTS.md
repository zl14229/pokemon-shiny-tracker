# AGENTS.md — Pokémon Shiny Tracker

## Read first
Before work, read `docs/PRODUCT_SPEC_V1.1.md`, `docs/TASK_BASELINE_V1.1.md`, `docs/ADR_REGISTER_V1.1.md`, and `docs/DATA_SOURCE_POLICY_V1.1.md` (or the repository's equivalent approved paths). Product/Task baseline takes priority over speculative features.

## GitHub workflow (ADR-015)
- User deleted prior GitHub remote. Start with a new clean repository; preserve old local directory as an optional read-only backup, not as completed implementation.
- `dev` is the default integration branch; `prod` is created after first accepted release; each Task starts from latest remote `dev` in `feature/Txxx-name`.
- All normal changes enter `dev` through a GitHub Pull Request; require successful checks, AI self-review, and real acceptance evidence. Use merge commits and preserve source branches. New work always branches from current `dev`, not old snapshots.
- Actual branch protection and workflows must be configured in GitHub; do not claim a protection mechanism exists unless verified. Releases to `prod` require user approval.
- Do not commit secrets, private SQLite database, personal attachments, videos, or runtime backups.

## Data governance (ADR-016)
- PokéAPI v2 is the main structured base source. Cache locally, pace requests, validate stable IDs and mapping, and keep original source metadata.
- 52Poké is for Chinese names and version-specific cross-checking. Follow its site-specific machine reading and copyright rules; do not assume an open bulk download API.
- Serebii, Bulbapedia, Pokémon Database and official material are for source-backed rules and validation. Do not assume bulk scraping rights; unknown or conflicting details remain marked unknown.
- Separate legal shiny availability, encounter probability, method conditions, shiny probability and guide completeness. A record that is obtainable only via transfer is not a directly huntable target in that game.
- Staged updates with provenance, diff, validation, atomic apply and rollback. Personal shiny records are never overwritten by reference-data sync.
- Planner may draw only from the verified local huntable subset; state incomplete game coverage to users, and work offline.

## Task completion
- Follow dependencies and acceptance criteria. For every Task keep change summary, tests/check results, review, risks, documentation update, PR URL and merge SHA; label unknowns honestly.
- Normal implementation/test-fix decisions can proceed autonomously. Pause for product/architecture changes, possible destruction of real data, permissions, paid resources, forced history rewrites, production release, or unrecoverable blockers.
- Do not pretend CI/PR merge/release has happened before tools actually confirm it.
