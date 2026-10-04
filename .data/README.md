# `.data/` — local dev/test stub (NOT the installed data folder)

When you run the app from source without an installed config, it uses this
directory as its data root:

- `WAYERPC_DATA_DIR=$PWD/.data wails dev`, or
- automatically when `.data/` exists and no `%LOCALAPPDATA%\WayerPC\config.json` is present.

Layout mirrors a real install (`<chosen>\WayerPC\data`):

```
.data/
├── shared/      # drop folder, also served by /ask
├── received/    # phone uploads land here
└── Data/        # wayerpc.db (SQLite catalog) lives here
```

Only this README and the `.gitkeep` files are committed. The real
`wayerpc.db*` files and any uploads you generate while testing stay local
(see root `.gitignore`). Delete them any time — the app recreates the tree.

To test with a pristine stub: close the app, delete everything except this
README and the `.gitkeep` files, and start again.
