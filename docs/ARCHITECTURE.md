# Architecture

## Data flow

```
                        GitLab API
                            |
                       glf --sync
                            |
            +---------------+---------------+
            |               |               |
    internal/gitlab    internal/cache   internal/index
    (parallel fetch)   (projects.txt)   (description.bleve)
            |               |               |
            +-------+-------+-------+-------+
                    |               |
               glf <query>     glf --json
                    |               |
            internal/search    JSON stdout
            (Bleve + history)      |
                    |         raycast-glf-extension
              internal/tui
              (Bubble Tea)
```

### Sync (`glf --sync`)

1. `internal/gitlab` fetches projects from the GitLab API using parallel pagination (up to 10 concurrent requests per page batch). It also fetches starred and member project lists for metadata enrichment.
2. `internal/cache` writes the project list to `projects.txt` in pipe-delimited format (`path|name|description`, one per line). Timestamps are saved to `.last_sync_time` / `.last_full_sync_time`.
3. `internal/index` builds a Bleve full-text index over three fields with boost weights: `ProjectName` (3.0), `ProjectPath` (2.0), `Description` (1.0). The index is schema-versioned (currently v4) and auto-recreated on version mismatch.

**Incremental sync** passes `last_activity_after` to the GitLab API so only recently changed projects are fetched. The sync mode (full vs incremental) is determined by `internal/sync` based on time since last full sync and a configurable threshold.

### Search (`glf <query>`)

Handled by `internal/search/combined.go`:

- **Empty query**: returns all projects sorted by history score (most recently/frequently used first).
- **Non-empty query**: runs a Bleve search across all indexed fields, then combines results with history and starred bonuses.

**Ranking formula**:

```
totalScore = bleveScore + (historyScore * relevanceMultiplier) + (starredBonus * relevanceMultiplier)
```

`relevanceMultiplier` is a non-linear function of `bleveScore` that maps the range `[0.1, 1.4]` to `[0.0, 1.0]` through 6 piecewise-linear segments. Its purpose is to prevent history/starred bonuses from promoting projects that barely match the search query. A project with `bleveScore < 0.1` gets zero history boost regardless of how often it was used.

`starredBonus` is +3 for starred projects, 0 otherwise.

### History scoring (`internal/history/`)

Each project selection is stored as an individual timestamp. The score is computed by summing exponential decay contributions from each timestamp:

```
score = sum( e^(-lambda * days_since_use) )   for each timestamp
```

- `lambda = ln(2) / 30` (half-life of 30 days)
- Timestamps older than 100 days are discarded
- Score is capped at 30 per project

Two tiers:
- **Global** selections contribute 1.0 per timestamp.
- **Query-specific** selections contribute 2.5 per timestamp (so a project chosen specifically for query "backend" ranks higher when searching "backend" again).

Storage format: Go `gob` encoding at `history.gob`. Writes are atomic (temp file + rename).

## JSON mode API contract

Used by `raycast-glf-extension` and other integrations. Activated by `glf --json <query>`.

**stdout** (search response):

```json
{
  "query":   "backend",
  "results": [
    {
      "path":        "group/project",
      "name":        "project",
      "description": "...",
      "url":         "https://gitlab.example.com/group/project",
      "starred":     true,
      "excluded":    false,
      "archived":    false,
      "member":      true,
      "score":       1.42
    }
  ],
  "total": 1,
  "limit": 50
}
```

`score` is only present when `--scores` is passed.

**Recording selections** (for history): `glf --json-record <project-path> --json-record-query <query>` writes to history without producing search output.

**Error response**: `{"error": "message"}` on stderr, exit code 1.

## Storage layout

```
~/.config/glf/
    config.yaml             # gitlab.url, gitlab.token, gitlab.timeout,
                            # cache.dir (default ~/.cache/glf),
                            # excluded_paths (glob patterns)

~/.cache/glf/               # default, overridden by cache.dir in config
    projects.txt            # pipe-delimited project list
    description.bleve/      # Bleve index directory (auto-managed)
    history.gob             # selection history (gob-encoded)
    .last_sync_time         # RFC3339, last successful sync
    .last_full_sync_time    # RFC3339, last successful full sync
    .username               # cached GitLab username (plain text)
```

## Module map

| Package | Responsibility |
|---------|---------------|
| `cmd/glf` | CLI entry point, cobra commands, JSON output, TUI orchestration |
| `internal/gitlab` | GitLab API client wrapper, parallel paginated fetching |
| `internal/cache` | Flat-file project cache, sync timestamps, username cache |
| `internal/index` | Bleve index lifecycle (create/open/rebuild), full-text search with field boosting |
| `internal/search` | Combines Bleve scores with history/starred bonuses, relevance-gated ranking |
| `internal/history` | Selection tracking with exponential decay, query-specific boost, gob persistence |
| `internal/config` | Viper-based config loading from YAML + env vars |
| `internal/model` | Shared `Project` struct |
| `internal/tui` | Bubble Tea interactive UI |
| `internal/sync` | Sync mode decision logic (full vs incremental) |
| `internal/logger` | Debug logging |
