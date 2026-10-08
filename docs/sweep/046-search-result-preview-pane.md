# 046 Search result preview pane

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | M (1-3 days) |
| Area | search/frontend |
| Phase | Week 5 — frontend depth |
| Related | `041`, `059` |

## 1. Why this matters

Import results are name+path rows (`import.ts:49-60`) — for 40 similar `IMG_*.jpg` there's no way to pick without importing all + checking Files. A preview pane (image thumb, text head, file meta) makes selection confident.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

No preview: checkbox rows only. `RevealInExplorer` is the only inspection (`app.go:486-491`). `Hit` has no size/kind (`041` adds it). No thumbnail pipeline (`059`).

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Over-importing (select-all 200) to avoid missing the right file.
- No meta (size/date) at decision time.
- Images unidentifiable by name.

## 4. Proposal (what "good" looks like)

Selecting a hit shows a side pane: name, path, size, modified, kind icon; image → thumbnail (`059` renderer reused); text (<200KB, text-sniffed) → first 20 lines; else → meta only. Preview reads locally, no import. Depends on `041` size/kind — do that first.

## 5. Step-by-step implementation

1. Land `041` (Hit size/kind) first.
2. Add `App.PreviewFile(path)` returning `{kind, size, modtime, textHead?, thumbPath?}` (cap reads; refuse >50MB preview).
3. `import.ts`: side pane on row select (single-click previews, checkbox imports — resolve the interaction explicitly).
4. Security: preview only paths from last search results (no arbitrary-path read primitive).
5. Handle missing/unreadable gracefully ('cannot preview').

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Image hit previews visually; text hit shows head; binary shows meta.
- [ ] Preview never imports (catalog count unchanged).
- [ ] 50MB+ file refuses preview with a clear message (no hang).

## 7. Tests to add / update

Preview-helper tests (text head truncation, binary refusal, missing path).

## 8. Risks and rollback

Arbitrary-file-read primitive — constrain to search-result paths server-side (validate against last results or re-stat + allowlist roots).

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`059` (shared thumbnail renderer), Android-side preview (out of scope).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
