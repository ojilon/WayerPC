import threading
import time
import os
import sys

import customtkinter as ctk

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from backend.server.server4 import start_server, server_stats, stats_lock
from backend.manage_storage.create_app_folder import setup_storage
from backend.native.dll_loader import sync_native_libs_to_app_folder
from backend.manage_storage.bridge_to_app_folder import (
    ensure_shared_folder,
    ensure_received_folder,
)

from theme import COLORS, FONT_UI, FONT_MONO
from import_panel import ImportTab
from library_panel import LibraryTab


ctk.set_appearance_mode("Dark")
ctk.set_default_color_theme("blue")


class WayerPCApp(ctk.CTk):
    def __init__(self):
        super().__init__()

        app_dir, sub_dir = setup_storage(default_drive_letter="D:\\")
        try:
            ensure_shared_folder()
            ensure_received_folder()
            sync_native_libs_to_app_folder()
        except FileNotFoundError:
            pass

        print("\n--- Ready ---")
        print(f"App Directory : {app_dir}")
        print(f"Sub Directory : {sub_dir}")

        self.title("WayerPC")
        self.geometry("1180x720")
        self.minsize(960, 600)
        self.configure(fg_color=COLORS["bg"])

        self.grid_columnconfigure(1, weight=1)
        self.grid_rowconfigure(0, weight=1)

        self._tab_seq = 1
        self._closable = set()
        self.library_tabs: list[LibraryTab] = []

        self._build_sidebar()
        self._build_workspace()

        self.log_message("Starting the server in the background")
        self.server_worker = threading.Thread(
            target=start_server,
            args=(self.log_message,),
            name="SocketServerThread",
            daemon=True,
        )
        self.server_worker.start()
        self.update_dashboard_metrics()

    # ------------------------------------------------------------------ sidebar
    def _build_sidebar(self):
        self.sidebar = ctk.CTkFrame(self, width=220, corner_radius=0, fg_color=COLORS["sidebar"])
        self.sidebar.grid(row=0, column=0, sticky="nsew")
        self.sidebar.grid_propagate(False)
        self.sidebar.grid_rowconfigure(8, weight=1)

        ctk.CTkLabel(
            self.sidebar,
            text="WayerPC",
            font=ctk.CTkFont(family=FONT_UI, size=22, weight="bold"),
            text_color=COLORS["text"],
        ).grid(row=0, column=0, padx=22, pady=(28, 2), sticky="w")
        ctk.CTkLabel(
            self.sidebar,
            text="Phone ↔ PC transfer",
            font=ctk.CTkFont(size=12),
            text_color=COLORS["muted"],
        ).grid(row=1, column=0, padx=22, pady=(0, 24), sticky="w")

        self._nav_btn("Console", self._focus_console).grid(row=2, column=0, padx=16, pady=4, sticky="ew")
        self._nav_btn("New import tab", lambda: self._add_tab("import")).grid(
            row=3, column=0, padx=16, pady=4, sticky="ew"
        )
        self._nav_btn("View files", lambda: self._add_tab("library")).grid(
            row=4, column=0, padx=16, pady=4, sticky="ew"
        )

        ctk.CTkButton(
            self.sidebar,
            text="Exit",
            fg_color=COLORS["danger"],
            hover_color=COLORS["danger_hover"],
            command=self.quit,
        ).grid(row=9, column=0, padx=16, pady=24, sticky="ew")

    def _nav_btn(self, text, command):
        return ctk.CTkButton(
            self.sidebar,
            text=text,
            fg_color="transparent",
            hover_color=COLORS["accent_dim"],
            anchor="w",
            command=command,
        )

    # ---------------------------------------------------------------- workspace
    def _build_workspace(self):
        wrap = ctk.CTkFrame(self, fg_color="transparent")
        wrap.grid(row=0, column=1, sticky="nsew", padx=18, pady=16)
        wrap.grid_rowconfigure(1, weight=1)
        wrap.grid_columnconfigure(0, weight=1)

        tabbar = ctk.CTkFrame(wrap, fg_color="transparent")
        tabbar.grid(row=0, column=0, sticky="ew", pady=(0, 8))
        tabbar.grid_columnconfigure(0, weight=1)

        self.tabs = ctk.CTkTabview(
            wrap,
            fg_color=COLORS["bg"],
            segmented_button_fg_color=COLORS["card"],
            segmented_button_selected_color=COLORS["accent"],
            segmented_button_selected_hover_color=COLORS["accent_hover"],
            segmented_button_unselected_color=COLORS["card"],
            segmented_button_unselected_hover_color=COLORS["accent_dim"],
        )
        self.tabs.grid(row=1, column=0, sticky="nsew")

        plus = ctk.CTkButton(
            tabbar,
            text="+",
            width=36,
            height=32,
            fg_color=COLORS["card"],
            hover_color=COLORS["accent"],
            command=self._plus_menu,
        )
        plus.pack(side="right")
        self.btn_close_tab = ctk.CTkButton(
            tabbar,
            text="Close tab",
            width=90,
            height=32,
            fg_color="transparent",
            border_width=1,
            border_color=COLORS["border"],
            command=self._close_current,
        )
        self.btn_close_tab.pack(side="right", padx=8)

        self._build_console_tab()
        self._add_tab("import", title="Import")
        self._add_tab("library", title="Received")

    def _plus_menu(self):
        pop = ctk.CTkToplevel(self)
        pop.title("New tab")
        pop.geometry("280x160")
        pop.configure(fg_color=COLORS["card"])
        pop.attributes("-topmost", True)
        ctk.CTkLabel(pop, text="Open a new tab", font=ctk.CTkFont(weight="bold")).pack(pady=(18, 10))
        ctk.CTkButton(
            pop, text="Import", command=lambda: (self._add_tab("import"), pop.destroy())
        ).pack(fill="x", padx=24, pady=4)
        ctk.CTkButton(
            pop, text="View files", command=lambda: (self._add_tab("library"), pop.destroy())
        ).pack(fill="x", padx=24, pady=4)

    def _build_console_tab(self):
        self.tabs.add("Console")
        page = self.tabs.tab("Console")
        page.grid_columnconfigure((0, 1), weight=1)
        page.grid_rowconfigure(1, weight=1)

        status = ctk.CTkFrame(page, fg_color=COLORS["card"], corner_radius=12, height=64)
        status.grid(row=0, column=0, columnspan=2, sticky="ew", pady=(8, 12))
        self.lbl_status = ctk.CTkLabel(
            status,
            text="SYSTEM STATUS: LOADING…",
            font=ctk.CTkFont(size=14, weight="bold"),
            text_color=COLORS["warning"],
        )
        self.lbl_status.pack(side="left", padx=20, pady=14)
        self.lbl_uptime = ctk.CTkLabel(status, text="Uptime: 0.00s", text_color=COLORS["muted"])
        self.lbl_uptime.pack(side="right", padx=20, pady=14)

        stats = ctk.CTkFrame(page, fg_color=COLORS["card"], corner_radius=12)
        stats.grid(row=1, column=0, sticky="nsew", padx=(0, 8))
        ctk.CTkLabel(
            stats, text="Network", font=ctk.CTkFont(weight="bold", size=14)
        ).pack(anchor="w", padx=16, pady=(14, 8))
        self.lbl_active_conn = self._stat(stats, "Active links", "0")
        self.lbl_total_conn = self._stat(stats, "Handled", "0")
        self.lbl_bytes_sent = self._stat(stats, "Outbound", "0.00 MB")
        self.lbl_bytes_received = self._stat(stats, "Inbound", "0.00 MB")

        logs = ctk.CTkFrame(page, fg_color=COLORS["card"], corner_radius=12)
        logs.grid(row=1, column=1, sticky="nsew", padx=(8, 0))
        logs.grid_rowconfigure(1, weight=1)
        logs.grid_columnconfigure(0, weight=1)
        ctk.CTkLabel(
            logs, text="Activity", font=ctk.CTkFont(weight="bold", size=14)
        ).grid(row=0, column=0, sticky="w", padx=16, pady=(14, 6))
        self.log_textbox = ctk.CTkTextbox(
            logs, font=ctk.CTkFont(family=FONT_MONO, size=12), fg_color=COLORS["card_alt"]
        )
        self.log_textbox.grid(row=1, column=0, sticky="nsew", padx=12, pady=(0, 12))
        self.log_textbox.configure(state="disabled")

    def _stat(self, parent, label, value):
        row = ctk.CTkFrame(parent, fg_color=COLORS["card_alt"], corner_radius=8)
        row.pack(fill="x", padx=16, pady=6)
        ctk.CTkLabel(row, text=label, text_color=COLORS["muted"]).pack(side="left", padx=12, pady=10)
        val = ctk.CTkLabel(row, text=value, font=ctk.CTkFont(weight="bold"))
        val.pack(side="right", padx=12)
        return val

    def _add_tab(self, kind: str, title: str | None = None):
        self._tab_seq += 1
        if title is None:
            title = f"Import {self._tab_seq}" if kind == "import" else f"Files {self._tab_seq}"
        # CTkTabview cannot add duplicate names
        existing = set(self.tabs._tab_dict.keys())  # noqa: SLF001
        base = title
        n = 2
        while title in existing:
            title = f"{base} ({n})"
            n += 1
        self.tabs.add(title)
        page = self.tabs.tab(title)
        page.grid_rowconfigure(0, weight=1)
        page.grid_columnconfigure(0, weight=1)
        if kind == "import":
            panel = ImportTab(page, self.log_message, on_catalog_changed=self._refresh_libraries)
        else:
            panel = LibraryTab(page, self.log_message)
            self.library_tabs.append(panel)
        panel.grid(row=0, column=0, sticky="nsew", padx=4, pady=4)
        self._closable.add(title)
        self.tabs.set(title)
        return title

    def _close_current(self):
        name = self.tabs.get()
        if name == "Console" or name not in self._closable:
            self.log_message("The Console tab stays open.")
            return
        self.tabs.delete(name)
        self._closable.discard(name)
        self.tabs.set("Console")

    def _focus_console(self):
        self.tabs.set("Console")

    def _refresh_libraries(self):
        for tab in self.library_tabs:
            try:
                tab.refresh()
            except Exception:
                pass

    # ---------------------------------------------------------------- logging
    def log_message(self, text):
        timestamp = time.strftime("%H:%M:%S")
        formatted_line = f"[{timestamp}] {text}\n"

        def _write():
            self.log_textbox.configure(state="normal")
            self.log_textbox.insert("end", formatted_line)
            self.log_textbox.see("end")
            self.log_textbox.configure(state="disabled")

        try:
            self.after(0, _write)
        except Exception:
            pass

    def update_dashboard_metrics(self):
        with stats_lock:
            running = server_stats["is_running"]
            active = server_stats["active_connections"]
            total = server_stats["total_connections_handled"]
            sent = server_stats["bytes_sent"]
            received = server_stats["bytes_received"]
            start = server_stats["start_time"]

        if running:
            self.lbl_status.configure(text="ONLINE", text_color=COLORS["success"])
            uptime = time.time() - start if start else 0
            self.lbl_uptime.configure(text=f"Uptime  {uptime:.0f}s")
            self.lbl_active_conn.configure(text=f"{active}")
            self.lbl_total_conn.configure(text=f"{total}")
            self.lbl_bytes_sent.configure(text=f"{sent / (1024 * 1024):.2f} MB")
            self.lbl_bytes_received.configure(text=f"{received / (1024 * 1024):.2f} MB")
        else:
            self.lbl_status.configure(text="OFFLINE", text_color=COLORS["danger"])
            self.lbl_uptime.configure(text="Uptime  0s")

        self.after(1000, self.update_dashboard_metrics)


if __name__ == "__main__":
    app = WayerPCApp()
    app.mainloop()
