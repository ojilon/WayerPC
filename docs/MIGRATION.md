# Native MyGo migration

The desktop window now uses `ui.View` and Go widgets from the pinned MyGo fork.
There is no embedded HTML, web frontend, JavaScript bridge, or webview window.
The native interface calls the existing application methods directly. Server,
search, transfer and sync notifications are queued from backend goroutines and
applied on the UI thread; long-running operations and dialogs run in workers.
The native window uses system-provided window chrome.

The phone wire protocol, ports, storage and catalog are unchanged. The native
screens cover the server, files, local and phone search, sync, import, search
rules and settings. Native layout and workflow details differ from the former
interface; they have not yet been validated with users or a real phone.

The shutdown path was corrected to stop the server without holding the app
mutex: stopping emits activity and previously could deadlock. Reads of mutable
configuration now use the app mutex. Search and sync events are filtered by
run ID so late results cannot overwrite the current view. Sync also reports
completion when a rechecked plan has nothing left to copy, allowing the native
controls to leave their busy state.

In the restricted migration environment the native UI compiled and headless
UI/notification tests ran using an **empty SQLite stub** and temporary module
replacements; those runs do not validate storage, the actual dependency graph,
a real desktop window, or Android interoperability. See BUILD.md.
