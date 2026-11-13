# Follow-up work

1. Run `go mod tidy` with access to the Go module proxy; review and commit the
   complete dependency graph and checksums. Run the full test suite with real
   SQLite, then build and smoke-test native windows on Windows, Linux and macOS.
2. Test v1 and v2 transfers against real Android devices, including reconnect,
   ask-phone, cancellation and sync. Keep wire format and ports unchanged unless
   both clients are updated together.
3. Test upgrades, installer metadata, native dialogs, data relocation and
   preservation of existing data on Windows. Package platform icons properly.
4. Add headless interaction tests for imports, selection, sync and error paths;
   add integration tests with a fake phone and a real SQLite database.
5. Review shutdown and data relocation while searches and transfers are in
   flight. Bound UI transfer history and virtualize large file lists.
6. Refine native styling and workflows with user feedback, and consider
   accessibility and keyboard navigation on each platform.
