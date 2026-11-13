# WayerPC

WayerPC is the desktop companion for the Wayer Android app. It shares and
receives files over the local network, maintains a searchable catalog, and
supports the v2 phone and sync flows.

The desktop interface is written in Go with the native UI toolkit in the
[ojilon/goui](https://github.com/ojilon/goui) MyGo fork. The application does
not create a webview or use a JavaScript frontend. The phone protocol, catalog
and data layout remain separate from the UI.

## Layout

- `main.go`: native window and application lifecycle.
- `desktop.go`: native screens, worker dispatch and UI notifications.
- `app.go`, `v2.go`, `sync.go`: application operations and phone integration.
- `internal/`: storage, search, catalog, networking and protocol logic.
- `legacy/`: previous implementation, retained for reference.
- [Build and test](BUILD.md), [migration notes](docs/MIGRATION.md),
  [suggestions](docs/SUGGESTIONS.md), [protocol v2](docs/protocol-v2.md),
  [search rules](doc/search-rules.md), [sync](doc/sync.md).

Set `WAYERPC_DATA_DIR` to a disposable directory to isolate local data while
testing. See [.data/README.md](.data/README.md).
