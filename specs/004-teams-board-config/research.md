# Research: Teams & Board Name Unified Configuration

## Decision 1 — Parsing strategy for `TEAMS=COMP|"Board Name",...`

**Decision**: Split on `,` first (entries), then split each entry on the first `|` (component vs. board name). Strip whitespace around each part; strip surrounding double-quotes from the board name if present.

**Rationale**: The pipe character `|` does not appear in Jira component names or board names in practice. Using it as the intra-entry separator avoids the need for a CSV-style parser and keeps the `.env` value readable. Comma remains the inter-entry separator (consistent with the existing `TEAMS` format).

**Alternatives considered**:
- URL-encoding: rejected — ugly in `.env`, non-obvious to operators
- JSON array as env var value: rejected — unusual for `.env` files; requires extra escaping
- New separate var per team (e.g., `TEAM_1_BOARD=...`): rejected — doesn't scale, requires knowing team count up-front

**Parsing pseudocode**:
```
for each entry in strings.Split(rawTeams, ","):
    entry = strings.TrimSpace(entry)
    if empty: skip
    parts = strings.SplitN(entry, "|", 2)
    componentName = strings.TrimSpace(parts[0])
    if len(parts) == 2:
        boardName = strings.TrimSpace(parts[1])
        boardName = strings.Trim(boardName, `"`)   // strip optional quotes
    else:
        boardName = componentName                   // backwards-compatible fallback
```

---

## Decision 2 — Where `TeamEntry` lives

**Decision**: Define `TeamEntry` in `internal/config/config.go` alongside `Config`. No new package.

**Rationale**: It is a configuration type consumed only by cmd binaries via `config.Load()`; it does not cross into `internal/jiraservice` or `internal/word`. Keeping it in the same file as `Config` avoids creating a new package for a two-field struct.

**Alternatives considered**:
- New `internal/config/teams.go` file: acceptable but unnecessary for a two-field struct
- Expose `TeamEntry` from a shared types package: rejected — over-engineered for current scale

---

## Decision 3 — `get-sprint-issues-from-jira` board selection

**Decision**: Use `cfg.Teams[0].BoardName` — the first (and typically only) team's board name.

**Rationale**: This command is a single-team sprint-issue fetcher; the spec explicitly states multi-team looping is out of scope for this binary. Using index `[0]` is the minimal change that keeps the binary functional after `cfg.BoardName` is removed. If `cfg.Teams` is empty, `config.Load()` already returns an error, so `[0]` is safe.

**Alternatives considered**:
- Add a `-board` flag: out of scope for this feature; can be added in a follow-up
- Loop over all teams (same as label report): explicitly excluded per spec

---

## Decision 4 — Error handling in `get-sprint-label-report` multi-team loop

**Decision**: On failure for one team, print an error to stderr, set a `hadError bool` flag, and continue to the next team. After all teams are processed, exit with code 1 if `hadError` is true.

**Rationale**: Matches FR-008 and the acceptance scenario in US3. Failing fast would mean a transient error for one board silently drops all subsequent teams. Continue-and-report gives the operator maximum output.

**Alternatives considered**:
- Fail fast on first error: rejected — drops all later teams from the report
- Collect all errors and print at the end: functionally equivalent; per-error stderr lines are more observable

---

## Decision 5 — Backwards compatibility for operators with existing `.env`

**Decision**: `JIRA_BOARD_NAME` is silently ignored if present. If an operator has not yet updated their `TEAMS` entries to include `|"Board Name"`, the component name is used as the board name (fallback). This means existing `.env` files continue to work as long as component name == board name.

**Rationale**: Minimises disruption during roll-out. Operators can update their `.env` at their own pace.

**Risk**: If component name ≠ board name and the operator hasn't updated `TEAMS`, the board lookup will silently use the wrong name. Mitigated by clear `.env.example` documentation.
