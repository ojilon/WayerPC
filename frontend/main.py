import threading
import time
import os
import sys

# Add project root to path so backend imports work
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

# Import CustomTkinter for the GUI layer
import customtkinter as ctk

# Import backend logic and thread-safe variables
from backend.server.server4 import start_server, server_stats, stats_lock
from backend.server.Locate import get_file_info

from import_panel import FileImportManager

# Configure CustomTkinter appearance
ctk.set_appearance_mode("Dark")      # Modes: "System", "Dark", "Light"
ctk.set_default_color_theme("blue")  # Themes: "blue", "green", "dark-blue"

class WayerPCApp(ctk.CTk):
    def __init__(self):
        super().__init__()

        # --- Window Setup ---
        self.title("WayerPC")
        self.geometry("850x550")
        self.minsize(700, 450)

        # --- Layout Configuration ---
        # Create a grid: 2 columns (Sidebar + Main Content panel)
        self.grid_columnconfigure(0, weight=0, minsize=200) # Sidebar stays fixed size
        self.grid_columnconfigure(1, weight=1)              # Main content expands
        self.grid_rowconfigure(0, weight=1)

        # --- Components UI Setup ---
        self.create_sidebar()
        self.create_main_dashboard()

        # --- Initialize Backend Worker ---
        self.log_message("Starting the server in the background")
        self.server_worker = threading.Thread(target=start_server, args=(self.log_message,), name="SocketServerThread", daemon=True)
        self.server_worker.start()

        #initialize the import manager module
        self.import_manager = FileImportManager(self, self.log_message)

        # --- Start Dashboard Loop ---
        # Schedule the UI to poll the thread-safe stats dictionary every 1000ms (1 second)
        self.update_dashboard_metrics()

    def create_sidebar(self):
        """Creates the navigation/action sidebar on the left side."""
        self.sidebar = ctk.CTkFrame(self, corner_radius=0)
        self.sidebar.grid(row=0, column=0, sticky="nsew", padx=0, pady=0)
        
        # Title Label
        self.logo_label = ctk.CTkLabel(self.sidebar, text="Quick Action", font=ctk.CTkFont(size=20, weight="bold"))
        self.logo_label.grid(row=0, column=0, padx=20, pady=30)

        # Action Buttons
        self.btn_import = ctk.CTkButton(self.sidebar, text="Import File", command=self.handle_import_file)
        self.btn_import.grid(row=1, column=0, padx=20, pady=10, sticky="ew")

        self.btn_meta_received = ctk.CTkButton(self.sidebar, text="Metadata (Received)", fg_color="transparent", border_width=1, command=lambda: self.fetch_metadata("received"))
        self.btn_meta_received.grid(row=2, column=0, padx=20, pady=10, sticky="ew")

        self.btn_meta_shared = ctk.CTkButton(self.sidebar, text="Metadata (Shared)", fg_color="transparent", border_width=1, command=lambda: self.fetch_metadata("shared"))
        self.btn_meta_shared.grid(row=3, column=0, padx=20, pady=10, sticky="ew")

        # Exit Button placed at the bottom
        self.btn_exit = ctk.CTkButton(self.sidebar, text="Exit Application", fg_color="#B22222", hover_color="#8B0000", command=self.quit)
        self.sidebar.grid_rowconfigure(4, weight=1) # Spacer row
        self.btn_exit.grid(row=5, column=0, padx=20, pady=20, sticky="s")

    def create_main_dashboard(self):
        """Creates the dashboard layout displaying cards for stats and logs."""
        self.main_container = ctk.CTkFrame(self, fg_color="transparent")
        self.main_container.grid(row=0, column=1, sticky="nsew", padx=20, pady=20)
        self.main_container.grid_columnconfigure((0, 1), weight=1) # Split stats into columns
        self.main_container.grid_rowconfigure(1, weight=1)          # Give log block extra vertical scaling space

        # --- Top Header Status Row ---
        self.status_card = ctk.CTkFrame(self.main_container, height=60)
        self.status_card.grid(row=0, column=0, columnspan=2, sticky="ew", pady=(0, 15))
        self.lbl_status = ctk.CTkLabel(self.status_card, text="SYSTEM STATUS: LOADING...", font=ctk.CTkFont(size=14, weight="bold"), text_color="#FFCC00")
        self.lbl_status.pack(side="left", padx=20, pady=15)
        self.lbl_uptime = ctk.CTkLabel(self.status_card, text="Uptime: 0.00s", font=ctk.CTkFont(size=12))
        self.lbl_uptime.pack(side="right", padx=20, pady=15)

        # --- Statistics Grid (Left Box) ---
        # FIXED: Removed text=" Network Metrics " from ctk.CTkFrame parameters
        self.stats_frame = ctk.CTkFrame(self.main_container)
        self.stats_frame.grid(row=1, column=0, sticky="nsew", padx=(0, 10), pady=(0, 10))
        self.stats_frame.grid_columnconfigure(1, weight=1)
        
        # This label now cleanly handles the text framing header safely
        self.stats_title = ctk.CTkLabel(self.stats_frame, text="Network Metrics", font=ctk.CTkFont(weight="bold", size=13))
        self.stats_title.grid(row=0, column=0, columnspan=2, padx=15, pady=(10, 5), sticky="w")
        
        self.lbl_active_conn = self.create_stat_row(self.stats_frame, "Active Links:", "0 concurrent devices", 1)
        self.lbl_total_conn = self.create_stat_row(self.stats_frame, "Total Handled:", "0 clients", 2)
        self.lbl_bytes_sent = self.create_stat_row(self.stats_frame, "Total Outbound:", "0.00 MB", 3)
        self.lbl_bytes_received = self.create_stat_row(self.stats_frame, "Total Inbound:", "0.00 MB", 4)

        # --- Log Output Engine (Right Box Window) ---
        # FIXED: Removed text=" System Activities Window Logs " from ctk.CTkFrame parameters
        self.log_frame = ctk.CTkFrame(self.main_container)
        self.log_frame.grid(row=1, column=1, sticky="nsew", padx=(10, 0), pady=(0, 10))
        self.log_frame.grid_rowconfigure(1, weight=1)
        self.log_frame.grid_columnconfigure(0, weight=1)

        # This label now cleanly handles the log framing header safely
        self.log_title = ctk.CTkLabel(self.log_frame, text="System Activities Window Logs", font=ctk.CTkFont(weight="bold", size=13))
        self.log_title.grid(row=0, column=0, padx=15, pady=(10, 5), sticky="w")

        # Scrollable textbox component makes reading past logs clean
        self.log_textbox = ctk.CTkTextbox(self.log_frame, font=ctk.CTkFont(family="Consolas", size=11))
        self.log_textbox.grid(row=1, column=0, sticky="nsew", padx=10, pady=10)
        self.log_textbox.configure(state="disabled") # Set to read-only initially

    def create_stat_row(self, parent, label_text, default_val, row_idx):
        """Helper to cleanly structure statistical UI row pairings inside frames."""
        lbl_title = ctk.CTkLabel(parent, text=label_text, font=ctk.CTkFont(weight="bold"), anchor="w")
        lbl_title.grid(row=row_idx, column=0, padx=15, pady=12, sticky="w")
        
        lbl_val = ctk.CTkLabel(parent, text=default_val, anchor="e")
        lbl_val.grid(row=row_idx, column=1, padx=15, pady=12, sticky="e")
        return lbl_val

    def log_message(self, text):
        """Thread-safe mechanism to write diagnostic logs onto the console pane frame."""
        timestamp = time.strftime("%H:%M:%S")
        formatted_line = f"[{timestamp}] {text}\n"
        
        # CustomTkinter components require unlocking modifications via normal states temporarily
        self.log_textbox.configure(state="normal")
        self.log_textbox.insert("end", formatted_line)
        self.log_textbox.see("end") # Automatically scroll down to the bottom
        self.log_textbox.configure(state="disabled")

    def update_dashboard_metrics(self):
        """Polls backend dictionary stats safely utilizing memory Mutex locks without locking UI thread."""
        with stats_lock:
            running = server_stats["is_running"]
            active = server_stats["active_connections"]
            total = server_stats["total_connections_handled"]
            sent = server_stats["bytes_sent"]
            received = server_stats["bytes_received"]
            start = server_stats["start_time"]

        if running:
            self.lbl_status.configure(text="SYSTEM STATUS: RUNNING (ONLINE)", text_color="#2ECC71")
            uptime = time.time() - start if start else 0
            self.lbl_uptime.configure(text=f"Uptime: {uptime:.2f}s")
            self.lbl_active_conn.configure(text=f"{active} devices")
            self.lbl_total_conn.configure(text=f"{total} clients")
            self.lbl_bytes_sent.configure(text=f"{sent / (1024*1024):.2f} MB")
            self.lbl_bytes_received.configure(text=f"{received / (1024*1024):.2f} MB")
        else:
            self.lbl_status.configure(text="SYSTEM STATUS: OFFLINE", text_color="#E74C3C")
            self.lbl_uptime.configure(text="Uptime: 0.00s")

        # Recursively registers itself to evaluate data again exactly 1000ms later 
        self.after(1000, self.update_dashboard_metrics)

    def handle_import_file(self):
        """triggered on clicking 'importfile'. """
        self.import_manager.start_import_workflow()

    def fetch_metadata(self, folder_type):
        """Requests file metadata descriptions natively."""
        self.log_message(f"Querying file metadata profiles inside '{folder_type}' context directory...")
        try:
            info_data = get_file_info(folder_type)
            
            # Build a clean, readable string line by line
            log_lines = [f"[{folder_type.upper()}] Details mapped:"]
            
            for file in info_data:
                # Convert bytes to MB for better readability
                size_mb = file['size_bytes'] / (1024 * 1024)
                
                item_text = (
                    f"  - {file['file_name']} ({file['file_type']})\n"
                    f"    Size: {size_mb:.2f} MB | Modified: {file['date_modified']}"
                )
                log_lines.append(item_text)
            
            # Combine everything with newlines and send to your log_message function
            self.log_message("\n".join(log_lines))

        except Exception as e:
            self.log_message(f"Metadata read error: {e}")


if __name__ == "__main__":
    # Standard application loop runner initialization
    app = WayerPCApp()
    app.mainloop()