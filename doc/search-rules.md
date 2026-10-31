# Search rules (local ranking)

`Find` ranks the local file index with a small, configurable scoring engine
(`internal/rank`). This document explains how a score is produced, what the
built-in rules do, and how to add a new one.

## The pipeline

```
query ──► rank.NewQuery ──► index.Candidates (SQL prefilter)  ──► rank.Engine.Rank ──► hits
            tokens           name_lower LIKE %tok% AND …           weighted mean        score + reason
```

1. **Parse** — `rank.NewQuery` lowercases the text into tokens (`textmatch.Tokens`)
   and remembers a trailing extension hint (`report pdf` → hint `pdf`).
2. **Prefilter** — the index (`internal/index`, table `file_index`) returns at
   most 2000 rows whose name or folder contains *every* token. The prefilter is
   deliberately dumb: it only bounds the candidate set, it never scores.
3. **Rank** — every candidate is scored by each *enabled* rule with a non-zero
   weight. The final score is the **weighted mean**:

   ```
   score = Σ(weight_i × ruleScore_i) / Σ(weight_i)
   ```

   Scores are clamped to 0..1 and rounded to 2 decimals for display.
4. **Cut off & sort** — hits below the cutoff (default `0.2`) are dropped, then
   sorted by score. Equal scores are broken by comparing the per-rule score
   vector in **config order**, i.e. the rule you dragged to the top wins ties.
   The remaining tie-break is the file name.
5. **Reason** — each hit carries the label of the rule with the largest
   `weight × score` contribution ("exact name", "contains", "same folder", …).

## Built-in rules

| id | Label | Score | Weight default |
|----|-------|-------|----------------|
| `exact_name` | Exact name | 1 when the query equals the file name with the extension ignored (also matches when the extension is included) | 100 |
| `name_contains` | Name contains | fraction of query tokens found in the name | 78 |
| `extension` | Extension | 1 when a token (or `*.ext`) matches the file extension | 55 |
| `recently_modified` | Recently modified | `0.5 ^ (ageDays / 30)` — a 30-day half-life | 42 |
| `folder_hint` | Folder hint | fraction of query tokens found in the parent folder path | 28 |
| `fuzzy_name` | Fuzzy name | bigram Sørensen–Dice similarity between query and name (same metric `/ask` uses) | 36 |

Rules self-register from `init()` in `internal/rank/rules.go`.

## Configuration

The ordered list is stored as JSON in the settings row `match_rules`:

```json
[
  {"id": "exact_name", "enabled": true, "weight": 100},
  {"id": "name_contains", "enabled": true, "weight": 78},
  {"id": "extension", "enabled": true, "weight": 55},
  {"id": "recently_modified", "enabled": true, "weight": 42},
  {"id": "folder_hint", "enabled": true, "weight": 28},
  {"id": "fuzzy_name", "enabled": true, "weight": 36}
]
```

`ParseConfig` is defensive: corrupt JSON falls back to the defaults, unknown
ids and duplicates are dropped, weights are clamped to 0..100, and any rule the
list forgot is appended with its default weight (so a new rule works
immediately after an upgrade). The UI for this list is *Find → Match rules*.

## Index-related settings

| Setting | Default | Meaning |
|---------|---------|---------|
| `index_roots` | `search.UserRoots()` (Documents, Downloads, Desktop, Pictures, Videos, Music + extra drives) | newline/comma separated folders to walk |
| `index_skip` | `.git`, `node_modules` | extra folder names to skip (hidden dot-folders are always skipped) |

The data folder's `received/` and `shared/` plus every catalog entry are always
indexed, whatever the roots are. `search.SkipPath`/`skipSegments` remain the
hard guard (Windows, Program Files, AppData, Recycle Bin, the WayerPC data
folder itself, `/proc`, …) — `internal/index/index_test.go` asserts that the
index and the live search skip exactly the same paths.

The walk is background, cancellable and yields between directories. It is
capped at 500 000 rows (`index.DefaultMaxRows`) and reruns on first start, when
older than 24 h, or from Settings → *Rebuild index*.

File size never excludes a file from the index; a configurable size cap only
applies to **hashing** (protocol v2, `sync.hmac` sizes), so huge files stay
findable without ever being read end-to-end.

## Adding a rule

1. Create `internal/rank/rules_<name>.go` with a type that implements:

   ```go
   type Rule interface {
       ID() string
       Label() string
       Score(q Query, c Candidate) (float64, string)
   }
   ```

   `Score` must return a value in 0..1 and a short reason ("recent",
   "same folder", …). Keep it pure: no I/O and no allocation-heavy work — it
   runs once per candidate.

2. Register it:

   ```go
   func init() { Register(myRule{}) }
   ```

3. That is all. `NormalizeConfig` appends the new rule to existing installs
   with the weight from `defaultWeights` (add an entry there to choose a
   different default), the settings pane lists it automatically, and
   `TestRulesSelfRegister` reminds you to keep registration order stable.

4. Add a table test next to the other rule tests in `internal/rank/rank_test.go`
   (score boundaries, case folding, empty query).

Do **not** change the IDs or the meaning of existing rules: they are persisted
in user settings.
