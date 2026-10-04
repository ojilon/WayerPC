"""Import tab — search user storage or pick via Explorer, then register paths in SQLite."""

import os
import sys
import threading
import subprocess
from pathlib import Path
from tkinter import filedialog

import customtkinter as ctk

from backend.usersearch.user_search import search_user_storage
from backend.catalog.import_catalog import add_imported_path, add_imported_folder
from theme import COLORS, FONT_UI



class ImportTab(ctk.CTkFrame):
    def __init__(self, master, log_callback, on_catalog_changed=None):
        super().__init__(master, fg_color=COLORS["card"], corner_radius=12)
        self.log = log_callback
        self.on_catalog_changed = on_catalog_changed
        self._hits: list[dict] = []
        self._build()

    def _build(self):
        header = ctk.CTkLabel(
            self,
            text="Import library",
            font=ctk.CTkFont(family=FONT_UI, size=18, weight="bold"),
            text_color=COLORS["text"],
        )
        header.pack(anchor="w", padx=20, pady=(18, 4))
        ctk.CTkLabel(
            self,
            text="Search your documents and drives (system files and AppData are skipped). "
            "Selected items are recorded in the catalog — they are not copied into shared/.",
            font=ctk.CTkFont(family=FONT_UI, size=12),
            text_color=COLORS["muted"],
            wraplength=720,
            justify="left",
        ).pack(anchor="w", padx=20, pady=(0, 12))

        bar = ctk.CTkFrame(self, fg_color="transparent")
        bar.pack(fill="x", padx=20, pady=(0, 10))
        bar.grid_columnconfigure(0, weight=1)

        self.query = ctk.CTkEntry(
            bar,
            placeholder_text="File or folder name…",
            height=36,
            border_color=COLORS["border"],
        )
        self.query.grid(row=0, column=0, sticky="ew", padx=(0, 8))
        self.query.bind("<Return>", lambda _e: self._start_search())

        ctk.CTkButton(
            bar,
            text="Search",
            width=100,
            height=36,
            fg_color=COLORS["accent"],
            hover_color=COLORS["accent_hover"],
            command=self._start_search,
        ).grid(row=0, column=1, padx=(0, 8))

        ctk.CTkButton(
            bar,
            text="Browse files",
            width=120,
            height=36,
            fg_color=COLORS["card_alt"],
            hover_color=COLORS["accent_dim"],
            command=self._browse_files,
        ).grid(row=0, column=2, padx=(0, 8))

        ctk.CTkButton(
            bar,
            text="Browse folder",
            width=120,
            height=36,
            fg_color=COLORS["card_alt"],
            hover_color=COLORS["accent_dim"],
            command=self._browse_folder,
        ).grid(row=0, column=3)

        self.status = ctk.CTkLabel(
            self, text="", font=ctk.CTkFont(size=12), text_color=COLORS["muted"]
        )
        self.status.pack(anchor="w", padx=20)

        self.list_frame = ctk.CTkScrollableFrame(
            self, fg_color=COLORS["card_alt"], corner_radius=10
        )
        self.list_frame.pack(fill="both", expand=True, padx=20, pady=(8, 12))

        actions = ctk.CTkFrame(self, fg_color="transparent")
        actions.pack(fill="x", padx=20, pady=(0, 16))
        ctk.CTkButton(
            actions,
            text="Import selected",
            height=36,
            fg_color=COLORS["accent"],
            hover_color=COLORS["accent_hover"],
            command=self._import_selected,
        ).pack(side="left")
        ctk.CTkButton(
            actions,
            text="Import all results",
            height=36,
            fg_color="transparent",
            border_width=1,
            border_color=COLORS["border"],
            command=self._import_all,
        ).pack(side="left", padx=8)

        self._checks: list[tuple[ctk.CTkCheckBox, dict]] = []

    def _start_search(self):
        q = self.query.get().strip()
        if not q:
            self.status.configure(text="Enter a file or folder name.")
            return
        self.status.configure(text=f"Searching for “{q}”…")
        self.log(f"[IMPORT] Searching user storage for {q!r}")
        threading.Thread(target=self._search_worker, args=(q,), daemon=True).start()

    def _search_worker(self, query: str):
        try:
            hits = search_user_storage(query)
        except Exception as e:
            self.after(0, lambda: self.status.configure(text=f"Search failed: {e}"))
            return
        self.after(0, lambda: self._show_hits(hits))

    def _show_hits(self, hits: list[dict]):
        self._hits = hits
        for child in self.list_frame.winfo_children():
            child.destroy()
        self._checks.clear()
        if not hits:
            self.status.configure(text="No matches in user storage.")
            return
        self.status.configure(text=f"{len(hits)} result(s) — tick the ones to import.")
        for hit in hits:
            row = ctk.CTkFrame(self.list_frame, fg_color="transparent")
            row.pack(fill="x", pady=3)
            var_holder = {"on": False}

            def toggle(h=hit, holder=var_holder):
                holder["on"] = not holder["on"]

            cb = ctk.CTkCheckBox(
                row,
                text=f"{hit['name']}",
                font=ctk.CTkFont(family=FONT_UI, size=13),
                command=toggle,
            )
            cb.pack(side="left", padx=(4, 8))
            loc = ctk.CTkLabel(
                row,
                text=hit["path"],
                font=ctk.CTkFont(size=11),
                text_color=COLORS["muted"],
                anchor="w",
            )
            loc.pack(side="left", fill="x", expand=True)
            self._checks.append((cb, hit))

    def _selected_hits(self) -> list[dict]:
        chosen = []
        for cb, hit in self._checks:
            try:
                if cb.get():
                    chosen.append(hit)
            except Exception:
                continue
        return chosen

    def _import_paths(self, paths: list[str]):
        ok = 0
        for p in paths:
            target = Path(p)
            if target.is_dir():
                n, _ = add_imported_folder(p)
                ok += n
            else:
                status, _ = add_imported_path(p)
                if status == 0:
                    ok += 1
        self.log(f"[IMPORT] Cataloged {ok} path(s)")
        self.status.configure(text=f"Imported {ok} path(s) into the catalog.")
        if self.on_catalog_changed:
            self.on_catalog_changed()

    def _import_selected(self):
        hits = self._selected_hits()
        if not hits:
            self.status.configure(text="Select at least one result.")
            return
        self._import_paths([h["path"] for h in hits])

    def _import_all(self):
        if not self._hits:
            self.status.configure(text="No results to import.")
            return
        self._import_paths([h["path"] for h in self._hits])

    def _browse_files(self):
        paths = filedialog.askopenfilenames(title="Choose files to import")
        if not paths:
            return
        self._import_paths(list(paths))

    def _browse_folder(self):
        folder = filedialog.askdirectory(title="Choose a folder to import")
        if not folder:
            return
        self._import_paths([folder])


def open_path_in_explorer(path: str) -> None:
    p = Path(path)
    target = p if p.is_dir() else p.parent
    if os.name == "nt":
        os.startfile(target)  # type: ignore[attr-defined]
    elif sys.platform == "darwin":
        subprocess.Popen(["open", str(target)])
    else:
        subprocess.Popen(["xdg-open", str(target)])
