# 076 Structured log levels (debug/info/warn/error)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | M (1-3 days) |
| Area | backend/frontend |
| Phase | Week 5 — observability |
| Related | `057`, `025` |

## 1. Why this matters

`logf(line string)` (`app.go:179-192`) is one flat string channel: `[CONNECT]`, `[ASK OK]`, `[TIMEOUT]` prefixes parsed by eye. Filtering ('show errors only'), alerting, and persisted queries (`025`) all want real levels + fields.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Prefix conventions only; Console filter is substring (`console.ts:91-93`); `[UPLOAD OK]` sniffed for catalog refresh (`app.go:186`) — stringly-typed control flow. No level, no structured fields (ip, bytes, duration).

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- 'Show me failures' = type 'ERROR' and hope prefixes agree.
- Level-worthy events (upload ok vs protocol violation) look identical.
- Future metrics (`078`) can't consume strings reliably.

## 4. Proposal (what "good" looks like)

Add `Level` + fields to the log path while keeping the string channel compatible: `logf` stays (wraps `logStructured(info, msg, fields)`), events carry `{level, tag, msg, fields, ts}`; Console filter gains level chips (All/Info/Warn/Error) + text; persisted jsonl (`025`) stores the structure. Migrate call sites incrementally (tag per call, no big bang).

## 5. Step-by-step implementation

1. Define `LogEvent{TS, Level, Tag, Msg, Fields}`; `emit('log', event)` (keep string compat: send object, frontend accepts both during migration).
2. Wrap: `logf` → info; add `logWarn/logErr` helpers; migrate server error paths first.
3. `console.ts`: level chips + color dots + filter composition (level AND text).
4. `025` jsonl writes the struct (this is why levels land before persistence hardens).
5. Replace the `[UPLOAD OK]` substring sniff with a typed field (`catalogChanged=true`).
6. Document tag catalog ( типо table like `013`).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Level filter shows only errors on demand.
- [ ] No string-sniffed control flow remains (`[UPLOAD OK]` contains-check gone).
- [ ] Old string events (if any third-party sender) still render.

## 7. Tests to add / update

Level/filter unit tests (pure); compat test (string event renders).

## 8. Risks and rollback

Event-shape change breaks `dev-mock.ts` + any external consumer — dual-read in frontend during migration, then cut over.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`057` (drawer uses levels), `078` (metrics derive from warn/error rates).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
