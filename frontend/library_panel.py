"""Library tab — received/ folder + import catalog."""

import datetime
import os
from pathlib import Path

import customtkinter as ctk

from backend.manage_storage.bridge_to_app_folder import obtain_folder_location
from backend.catalog.import_catalog import list_imported, remove_imported_path
from theme import COLORS, FONT_UI, FONT_MONO



class LibraryTab(ctk.CTkFrame):
    def __init__(self, master, log_callback):
        super().__init__(master, fg_color=COLORS["card"], corner_radius=12)
        self.log = log_callback
        self._build()
        self.refresh()

    def _build(self):
        top = ctk.CTkFrame(self, fg_color="transparent")
        top.pack(fill="x", padx=20, pady=(18, 8))
        ctk.CTkLabel(
            top,
            text="Files",
            font=ctk.CTkFont(family=FONT_UI, size=18, weight="bold"),
        ).pack(side="left")
        ctk.CTkButton(
            top,
            text="Refresh",
            width=90,
            height=32,
            fg_color=COLORS["card_alt"],
            hover_color=COLORS["accent_dim"],
            command=self.refresh,
        ).pack(side="right")

        self.switch = ctk.CTkSegmentedButton(
            self,
            values=["Received", "Imported catalog"],
            command=lambda _v: self.refresh(),
        )
        self.switch.set("Received")
        self.switch.pack(anchor="w", padx=20, pady=(0, 8))

        self.hint = ctk.CTkLabel(
            self, text="", font=ctk.CTkFont(size=12), text_color=COLORS["muted"]
        )
        self.hint.pack(anchor="w", padx=20)

        self.body = ctk.CTkScrollableFrame(self, fg_color=COLORS["card_alt"], corner_radius=10)
        self.body.pack(fill="both", expand=True, padx=20, pady=(8, 16))

    def refresh(self):
        for child in self.body.winfo_children():
            child.destroy()
        mode = self.switch.get()
        if mode == "Received":
            self._show_received()
        else:
            self._show_catalog()

    def _row(self, title: str, subtitle: str, extra: str = ""):
        frame = ctk.CTkFrame(self.body, fg_color=COLORS["card"], corner_radius=8)
        frame.pack(fill="x", pady=4, padx=4)
        ctk.CTkLabel(
            frame,
            text=title,
            font=ctk.CTkFont(family=FONT_UI, size=13, weight="bold"),
            anchor="w",
        ).pack(fill="x", padx=12, pady=(8, 0))
        ctk.CTkLabel(
            frame,
            text=subtitle,
            font=ctk.CTkFont(family=FONT_MONO, size=11),
            text_color=COLORS["muted"],
            anchor="w",
        ).pack(fill="x", padx=12)
        if extra:
            ctk.CTkLabel(
                frame,
                text=extra,
                font=ctk.CTkFont(size=11),
                text_color=COLORS["muted"],
                anchor="w",
            ).pack(fill="x", padx=12, pady=(0, 8))
        return frame

    def _show_received(self):
        status, folder = obtain_folder_location("received")
        if status != 0 or not folder.is_dir():
            self.hint.configure(text="received/ is not available yet.")
            return
        files = [p for p in folder.rglob("*") if p.is_file()]
        self.hint.configure(text=f"{len(files)} file(s) in {folder}")
        if not files:
            ctk.CTkLabel(
                self.body,
                text="Nothing received from the phone yet.",
                text_color=COLORS["muted"],
            ).pack(pady=20)
            return
        for p in sorted(files, key=lambda x: x.stat().st_mtime, reverse=True):
            st = p.stat()
            when = datetime.datetime.fromtimestamp(st.st_mtime).strftime("%Y-%m-%d %H:%M")
            size_mb = st.st_size / (1024 * 1024)
            self._row(p.name, str(p), f"{size_mb:.2f} MB  ·  {when}")

    def _show_catalog(self):
        try:
            rows = list_imported()
        except FileNotFoundError:
            self.hint.configure(text="Storage is not initialized.")
            return
        self.hint.configure(text=f"{len(rows)} imported path(s) — used by /ask on the phone")
        if not rows:
            ctk.CTkLabel(
                self.body,
                text="Catalog is empty. Use an Import tab to add files.",
                text_color=COLORS["muted"],
            ).pack(pady=20)
            return
        for row in rows:
            exists = os.path.isfile(row["path"])
            extra = f"{(row['size_bytes'] or 0) / (1024 * 1024):.2f} MB  ·  {row['imported_at']}"
            if not exists:
                extra += "  ·  missing on disk"
            frame = self._row(row["name"], row["path"], extra)
            ctk.CTkButton(
                frame,
                text="Remove",
                width=80,
                height=28,
                fg_color="transparent",
                border_width=1,
                border_color=COLORS["danger"],
                text_color=COLORS["danger"],
                command=lambda p=row["path"]: self._remove(p),
            ).pack(anchor="e", padx=12, pady=(0, 8))

    def _remove(self, path: str):
        remove_imported_path(path)
        self.log(f"[CATALOG] Removed {path}")
        self.refresh()
