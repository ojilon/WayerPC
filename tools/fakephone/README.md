# fakephone

A tiny command-line "phone" for testing WayerPC without an Android build. It is
a developer/QA tool — the desktop app never needs it, and nothing in it ships.

It speaks **both** protocols:

* **v1** — `/ask`, `/upload` (the classic line protocol).
* **v2** — the `@`-framed two-way link spec'd in
  [`docs/protocol-v2.md`](../../docs/protocol-v2.md).

## Build / run

```bash
go run ./tools/fakephone -help
# or
go build -o fakephone ./tools/fakephone
```

Set `-addr` to what the Console shows (e.g. `192.168.43.20:5000`).

## Commands

| command | what it does |
|---|---|
| `probe` | v2 handshake (`hello` → `hello_ok`) plus a `ping`/`pong` round trip. The fastest "is the PC listening and is v2 on?" check. |
| `ask` | v1 `/ask <query>`: prints the `MATCHES` list, or streams a `FOUND` file and reports the byte count. |
| `upload` | v1 `/upload <size> <name>`: reads `READY`, sends the bytes, expects `DONE`. |
| `phone` | Full v2 phone: hellos as a control link to `-dir`, answers `query` with a numbered list, pushes the picked file over a data connection (`push`/`ready`/bytes/`done`), serves `get`, answers `manifest_req` with the folder listing, and accepts `offer`s by pulling them into `-out`. |

## Examples

```bash
# is v2 available?
fakephone probe -addr 192.168.43.20:5000

# old protocol, still supported
fakephone ask    -addr 192.168.43.20:5000 -query ecology
fakephone upload -addr 192.168.43.20:5000 -file ./IMG_4821.jpg

# act as a phone that owns a folder:
#   PC → Ask phone "notes"  → fakephone answers with notes.txt
#   PC → pick 1             → fakephone pushes it into received/
#   PC → Sync               → fakephone answers with a flat manifest
#   PC → Files → Send       → fakephone accepts and saves into ./from-pc
fakephone phone -addr 192.168.43.20:5000 -dir ./phone-files -out ./from-pc
```

`-did` overrides the device id advertised in `hello` (default
`fakephone-1`), so two fake phones can be told apart in **Settings → Known
phones**.

## Notes and limits

* `phone` serves only the **top level** of `-dir` (the sync model is a flat set
  of names) and skips dot-files.
* It does not implement resume: a `push` it performs always starts at offset 0
  and it honours whatever offset the PC reports in `ready`.
* Inline sends (flow C, single match) are written to `-out` and acknowledged
  with `done`.
* Exit code is non-zero on any protocol error, so it is safe to use in a
  smoke script:

```bash
fakephone probe -addr 127.0.0.1:5000 || echo "v2 not reachable"
```
