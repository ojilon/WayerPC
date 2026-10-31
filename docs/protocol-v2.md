# WayerPC protocol v2 — byte-exact specification

Status: **implemented on the PC side** (WayerPC 0.3.0). The Android client is a
separate repository (`ojilon/Wayer`); everything Android-side here is the
contract, not a claim about what the app does today. See
[`docs/android-changes.md`](android-changes.md) for the client checklist.

Protocol v1 — `/ask`, `/upload`, the `MATCHES` / `FOUND` / `READY` / `DONE`
lines and the 500 ms newline-less settle window — is **unchanged**. A phone that
never sends the v2 marker sees byte-identical behaviour to 0.2.x.

---

## 1. Opting in

v2 is negotiated per TCP connection by the **first byte**:

| first byte | meaning | served by |
|---|---|---|
| `@` (`0x40`) | protocol v2 frame | v2 session layer (`internal/session`) |
| anything else | legacy v1 command line | v1 handlers (`internal/server`), untouched |

Consequences:

* The phone chooses. There is no capability exchange, no port change, no
  handshake on a second connection.
* A v1 client that happens to send a line starting with `@` would be
  misinterpreted; no v1 command starts with `@`, so this cannot happen in
  practice.
* The PC can refuse v2 (`Settings → Phone protocol → Disable`, persisted as
  `protocol_v2: false` in `config.json`). A refused `@` connection is closed
  after logging `[V2] … asked for protocol v2 but it is disabled here`.
* v1 keeps working while v2 is enabled and while it is disabled.

## 2. Framing

Every v2 message is one frame:

```
'@' <decimal byte length, ASCII digits> '\n' <UTF-8 JSON object>
```

* The marker is exactly one byte, `0x40`.
* The length counts the JSON payload only — not the marker, not the `\n`, not
  any padding.
* No leading zeros, no spaces, no `+`. `1..4194304` (4 MiB) is the accepted
  range; a longer declared length or a declared length of `0` is a
  `bad_request`.
* The payload is a single JSON object with a string field `t` (the message
  type). Unknown `t` values get `error: unsupported`.
* Raw bytes (file bodies) are **never** framed. After a `push`/`pull`/`file`
  frame both sides switch to raw streaming for exactly the announced number of
  bytes, then return to framing on the same connection.

Example (`hello`):

```
@58\n{"t":"hello","proto":2,"role":"control","device":"Pixel 8","did":"a1b2"}
```

The JSON is written compactly (`json.Marshal`, keys in struct order). A receiver
must not depend on key order; a sender must not send duplicate keys.

### Limits

| constant | value | rationale |
|---|---|---|
| `MaxFrame` | 4 MiB (4 194 304) | a JSON control frame never needs more; a manifest travels in parts |
| `ProtoVersion` | 2 | carried in `hello` + `hello_ok` |
| `DefaultPing` | 20 s | keepalive cadence on the control link |
| `PongTimeout` | 60 s | silent control link is dropped after this |
| `QueryTTL` | 120 s | how long a pending request (query/pick/offer/get) stays valid |
| `MaxMatches` | 50 | numbered list cap per query |
| `MaxDataConns` | 4 | concurrent data connections per PC |
| `MaxUploadBytes` | 2 GiB | one file, both directions |
| request log | 200 rows | newest kept |

## 3. Connections

* **The phone always dials.** The PC never connects out.
* One **control** connection per phone, long-lived for the whole session.
  Re-helloing with the same `did` on a new connection **replaces** the old one:
  the PC closes the previous socket and keeps the newest.
* **Data** connections are short-lived and carry exactly one transfer:
  `hello(role=data)` → `hello_ok` → one `push` *or* one `pull` → bytes → `done`
  → close. At most `MaxDataConns` may exist at once; a fifth gets
  `error{busy}` before the handshake completes.
* Every request carries an `id` chosen by the requester. PC-originated ids look
  like `pc-<n>`; phone-originated ids are opaque strings chosen by the phone and
  must be unique per control connection.

## 4. Messages

`t` values are the strings below. Fields marked *req* are required.

### Handshake

| `t` | fields | direction |
|---|---|---|
| `hello` | `proto` (req, must be 2), `role` (`control`\|`data`, req), `device` (friendly name), `did` (stable device id) | phone → PC |
| `hello_ok` | `proto` (2), `pc` (PC name), `ping` (seconds, default 20) | PC → phone |
| `error` | `id`, `code`, `msg` | either |
| `ok` | `id` | either (acknowledges abort/simple requests) |
| `ping` | — | either |
| `pong` | — | either |

`hello` must be the first frame on every connection. Anything else →
`error{bad_request}` and close. `proto != 2` → `error{unsupported}` and close.
A `role=data` hello with an unknown `did` → `error{bad_request}` ("no control
link for this did").

### Flow C — the phone asks for a PC file

| `t` | fields | notes |
|---|---|---|
| `query` | `id`, `q`, `max` | `max` ≤ 50 |
| `matches` | `id`, `items[]{n,name,size,mtime}`, `ttl` | numbered list, `ttl` = 120 (seconds the list stays pickable) |
| `pick` | `id`, `n` | the phone chooses a number |
| `abort` | `id` | cancels a query or a pick |
| `file` | `id`, `name`, `size`, `mtime`, `sha256` | sent on the control connection *or* a data connection |
| raw bytes | — | exactly `size` bytes, immediately after `file` |
| `done` | `id`, `bytes`, `sha256` | receiver's confirmation |

Rules:

1. `query` with exactly one match → the PC may answer `file` + bytes inline on
   the control connection (`done` follows on the same connection).
2. `query` with several matches → `matches`. The phone answers `pick{n}`; the PC
   answers `file` + bytes, either inline or by having the phone open a data
   connection and send `pull{id}` there. Both are valid; a receiver must accept
   either.
3. `query` with no match → `error{id, code:"not_found"}`.
4. A `pick` for an unknown, already-aborted or older-than-`ttl` id →
   `error{code:"expired"}`.
5. `abort` always answers `ok`.

### Flow B — the PC asks the phone for a file

| `t` | fields | notes |
|---|---|---|
| `query` | `id`, `q`, `max` | PC → phone (same message as above) |
| `matches` | `id`, `items[]`, `ttl` | phone answers |
| `pick` | `id`, `n` | PC answers |
| `push` | `id`, `name`, `size`, `mtime`, `sha256`, `offset` | phone opens a data connection and offers the file |
| `ready` | `id`, `offset` | the offset the PC already has |
| raw bytes | — | `size - offset` bytes |
| `done` | `id`, `bytes`, `sha256` | PC confirms |

`offset` in `push` is the phone's requested start (normally 0); the PC's `ready`
is authoritative. The phone must send `size - ready.offset` bytes.

### Flow D — the PC offers a file

| `t` | fields | notes |
|---|---|---|
| `offer` | `id`, `name`, `size`, `mtime`, `sha256` | PC → phone |
| `accept` | `id` | phone accepts |
| `reject` | `id`, `code` | phone refuses (usually `denied`) |
| `pull` | `id`, `offset` | phone asks for the bytes on a data connection |
| `file` | `id`, `name`, `size`, `mtime`, `sha256` | PC answers on that connection |
| raw bytes | — | `size - offset` bytes |
| `done` | `id`, `bytes`, `sha256` | phone confirms |

`pull` for an id that is not an accepted offer → `error{code:"expired"}`. A
`done` whose `sha256` differs from the offered one fails the offer with
`hash_mismatch`.

### Flow E — the PC asks for one known file

| `t` | fields | notes |
|---|---|---|
| `get` | `id`, `path` | `path` is a phone-side path/name |
| `push` | `id`, `name`, `size`, `sha256` | phone answers with a data connection, exactly like flow B |

### Flow F — sync manifests

| `t` | fields | notes |
|---|---|---|
| `manifest_req` | `id`, `scope` | `scope` = `"sync"` (flat name set) |
| `manifest` | `id`, `part`, `last`, `items[]{p,size,mtime,sha256}` | 1+ parts; `last:true` ends the stream |

Parts arrive in order; the receiver concatenates them and sorts by `p`. Up to 32
buffered parts; a flood gets `error{busy}`. `sha256` may be omitted per item.

## 5. Errors

`error{id, code, msg}` — `id` echoes the request (empty when the failure is not
tied to one, e.g. a bad handshake). `msg` is human text for logs, never parsed.

| code | when |
|---|---|
| `bad_request` | malformed frame, missing field, unsafe name |
| `unsupported` | unknown `t`, wrong `proto` |
| `not_found` | no such file/name on the sender |
| `expired` | id unknown, aborted, or past `ttl` |
| `aborted` | the transfer was interrupted before completion |
| `busy` | too many data connections / manifest flood |
| `denied` | the receiving user refused an offer |
| `too_large` | `size > 2 GiB` |
| `hash_mismatch` | the sha256 on `done` differs from the bytes |
| `internal` | local I/O failure (disk full, permissions) |

## 6. Names, paths and safety

The receiver sanitises every incoming name **before** touching the filesystem.
Rejected (→ `bad_request`), in this order:

1. any `..` path component or the substring `..`;
2. control characters (`< 0x20`, `0x7f`);
3. empty result after taking the basename.

Then the basename is taken (last `/` or `\` wins), leading/trailing spaces are
trimmed, and finally these are rejected:

* names starting with `.` (hidden, and the resume folder is `.partial`);
* trailing `.` (Windows strips it, so `a.` would silently become `a`);
* Windows reserved device names (`CON`, `PRN`, `AUX`, `NUL`, `COM1..9`,
  `LPT1..9`, case-insensitively, with or without an extension);
* any of `<>:"|?*`;
* more than 200 bytes (UTF-8).

Accepted examples: `report.pdf`, `my notes (2).txt`, `a.b.c`. Rejected:
`../secret`, `..\secret`, `a/../b`, `con.txt`, `a:b`, `.hidden`, `boom\x00`.

## 7. Receiving, resume and collisions

* Uploads stream into a partial file while a sha256 runs.
* **With** `sha256` (recommended): the partial is
  `received/.partial/<lowercase sha256>.part`. A repeat `push` with the same
  hash resumes: `ready.offset` reports what is already stored, the PC verifies
  the prefix hash before continuing.
* **Without** `sha256`: the partial is `received/<name>.part`, truncated at the
  start of each push; there is no resume.
* Partials older than **24 h** are removed whenever a new upload starts.
* A final hash mismatch deletes the partial and answers `hash_mismatch`; the
  bytes are never promoted to a real file.
* A completed upload is moved to `received/`. If the name is taken, both files
  are kept: the new one becomes `name (1).ext`, `name (2).ext`, …
* `hash_cache` (SQLite) remembers `path → (size, mtime, sha256)` so unchanged
  files are never hashed twice.

## 8. Keepalive and teardown

* Each side sends `ping` every `hello_ok.ping` seconds (default 20) and answers
  `pong`.
* A control link that has not been heard from for 60 s is closed by the PC; the
  phone reconnects and re-hellos.
* Data connections are bounded by a 5-minute read deadline while waiting for
  `push`/`pull`, and by the transfer's own progress for the bytes.
* Closing the control link invalidates every pending request on it (waiters see
  `expired`).

## 9. PC-side state

* `devices` (SQLite): `did`, `name`, `last_seen`, `last_ip` — survives restarts;
  `GetDevices()` merges it with the live link state.
* Request log (in memory, ring of 200): phone-initiated queries with
  `{id, ts, device, query, matchCount, picked, pickedName, outcome, bundle}`.
  This is what the **Find → Requests** pane shows.
* Transfers reuse the same `transfer` events as v1, so the 50 px strip and the
  Console activity table work unchanged.

## 10. Verifying an implementation

```
go test ./internal/wire/ ./internal/session/ ./internal/server/
```

`internal/session/session_test.go` drives the whole protocol over `net.Pipe`
with a fake phone: handshake rejection, all four flows, multi-part manifests,
resume, hash mismatch, name sanitising, collisions, control replacement, ping
timeout, the connection cap and the 2 GiB cap. `internal/server/v2_test.go`
proves the first-byte dispatch leaves v1 untouched.

A from-scratch client can be exercised against `tools/fakephone` (phase 7).
