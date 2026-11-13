# Build and test

## Requirements

Go 1.27.3 or newer is required by the pinned MyGo fork. No Node.js, frontend
build, or webview runtime is needed by this application. MyGo native UI needs
the platform's graphics and windowing support; consult the fork's native UI
documentation for the target OS. NSIS 3 is needed for the Windows installer.

## From a clean checkout

```sh
cp version.json internal/version/version.json  # keep embedded version in sync
go mod tidy                                  # resolve graph and checksums
go test ./...
go build -o wayerpc .
./wayerpc --version
```

The fork retains the upstream module path `github.com/egoist/mygo`; the
`replace` in `go.mod` pins the actual `github.com/ojilon/goui` commit. The Go
module graph still needs `go mod tidy` in an environment with access to the Go
module proxy. Do not treat it as release-ready until the resulting graph and
checksums have been reviewed and committed.

On Windows, `scripts/build-installer.ps1` syncs the embedded version, builds
the Go executable and invokes NSIS. Use `-Portable` for a zip. Installer and
portable output need validation on Windows after the migration.

## Smoke checks before release

Use a disposable data root. Check native navigation, file/folder dialogs,
server start/stop and window close. Check logs, transfers, imports, search,
phone requests, rules, data relocation and sync. Exercise uploads and downloads
with a real Android client on the same network, using both v1 and v2, including
ask-phone and sync. Confirm restart preserves the catalog and configuration.
