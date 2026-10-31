# Sync: comparing this PC with the phone

Sync never moves a file on its own. It compares the two sides, shows one
direction arrow per row, and copies only what you tick — **nothing is ever
deleted**, on either side.

## The two sides

| side | what it is |
|---|---|
| **This PC** | `received/` + `shared/`, flattened to a set of bare file names. If the same name exists in both folders the **newer mtime wins** (that is the copy a phone would actually receive). |
| **Phone** | the flat list the phone returns for a `manifest_req` with `scope:"sync"` (`docs/protocol-v2.md` §4, flow F). |

Names are compared case-insensitively and folder separators are dropped, so
`Data/received/Report.PDF` and `report.pdf` on the phone are the same entry.

## Verdicts

Computed by `internal/syncplan` (pure functions, no I/O):

| verdict | when | action |
|---|---|---|
| `only_pc` | the name is on the PC only | `push` (PC → phone) |
| `only_phone` | the name is on the phone only | `pull` (phone → PC) |
| `newer_pc` | on both, different, the PC copy has the newer mtime | `push` |
| `newer_phone` | on both, different, the phone copy has the newer mtime | `pull` |
| `same` | equal size **and** mtimes within 2 s, or equal sha256 | `none` |

* 2 seconds of tolerance absorbs file-system mtime rounding and the phone's
  whole-second timestamps.
* An equal sha256 (when both sides report one) beats mtime entirely.
* "Different" rows are the only ones whose arrow can be flipped in the UI;
  a one-sided file has exactly one possible direction.

## Applying

`ApplySync(names)` re-scans first (a stale list must never move the wrong copy),
then for each selected row:

* `push` → protocol v2 `offer` → `accept` → `pull` → bytes → `done`;
* `pull` → protocol v2 `get` → data connection → `push` → bytes → `done`.

Every file emits the normal `transfer` events (so the 50 px strip and the
Console table show it) plus one `sync-progress` update
`{runId, name, action, state, index, total}` with state
`active` → `done` / `error`. When the queue finishes a final `sync-result`
refreshes the columns.

## UI (concept 07)

* Two columns, one row per file, a single filled **Sync selected (N)** button.
* Headings: *Only on PC · N*, *Different version · N*, *Only on phone · N*, and
  a collapsed *In both · N files (identical)* line on each side.
* Arrows flip on "different" rows; one-sided rows keep their direction and say
  so in the tooltip.
* Without a control link the whole pane shows why and prints the address the
  phone should dial.
