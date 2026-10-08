# 059 Image thumbnails in Files

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | M (1-3 days) |
| Area | frontend/backend |
| Phase | Week 5 — frontend depth |
| Related | `028`, `046` |

## 1. Why this matters

A photo library identified by `IMG_20260112_081522.jpg` × 2,000 is unnavigable as text rows. Small thumbs make Files a real library — and the renderer is shared with `046` preview.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Rows are text-only (`library.ts:44-55`); no image pipeline; `FileInfo` has no thumb (`storage.go:69-75`). Backend has no resize lib. Paging (`028`) doesn't exist yet, so thumbs would multiply DOM cost.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Text rows for visual content = wrong tool.
- Full-size loads would OOM the WebView; naive `<img src=file://>` leaks absolute paths + breaks Wails asset sandbox.
- 2,000 thumbs at once kills scrolling (needs `028` paging first).

## 4. Proposal (what "good" looks like)

After `028` paging: backend `Thumbnail(path, 128px)` (pure-Go resizer, disk cache under `Data/thumbs/`, LRU cap e.g. 500MB) served via Wails asset handler or base64; frontend shows 48px thumbs in file rows (images only: jpg/png/webp/gif-first-frame); missing/broken → kind icon. Lazy-load per page (IntersectionObserver or page-load batch).

## 5. Step-by-step implementation

1. Land `028` paging first (gate).
2. Pick resizer lib (pure-Go; no cgo); `Thumbnail` with cache key = sha+mtime+size.
3. Serve thumbs (base64 data URLs simplest; measure payload — paginate 100/page max).
4. Files rows: thumb `<img loading=lazy>` + fallback icon; disable for non-images.
5. Cache prune (size cap, oldest-first) + Settings 'Clear thumbnail cache'.
6. Perf test: 100-image page renders <1s; cache hit <100ms.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] 100-photo page shows thumbs, scrolls smoothly.
- [ ] Non-images unaffected (no thumb attempts).
- [ ] Cache bounded; clear button works.

## 7. Tests to add / update

Thumb determinism test (same input → same bytes); cache-cap test; broken-image fallback test.

## 8. Risks and rollback

Dep weight + WebView image quirks (heic/avif won't decode — show icon, document). Memory: never full-decode 50MP raws at full res — decode+resize streaming.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`046` (reuses renderer), video-poster frames (explicit future, not this).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
