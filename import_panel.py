import threading
import customtkinter as ctk
from pathlib import Path

# Import your core filesystem utilities
from python.server.ItemImporter import pull_file_path, copy_file
from python.server.Locate import nail_folder_location

class FileImportManager:
    def __init__(self, main_app_window, log_callback):
        """
        Handles the multi-step file import GUI workflow.
        - main_app_window: Reference to your main CustomTkinter app (self from master.py)
        - log_callback: Reference to your log_message method to print to the main console
        """
        self.app = main_app_window
        self.log = log_callback

    def start_import_workflow(self):
        """Step 1: Get the initial search query from the user via GUI."""
        self.log("Opening import wizard...")
        
        dialog = ctk.CTkInputDialog(text="Enter format: 'basepath targetfile'\nExample: C:/Users/Docs file.txt", title="Import File")
        entry_string = dialog.get_input()
        
        if not entry_string:
            self.log("Import operation canceled.")
            return

        # Start the search process in a background thread to prevent GUI freezing
        search_thread = threading.Thread(target=self._bg_search_worker, args=(entry_string,), daemon=True)
        search_thread.start()

    def _bg_search_worker(self, entry_string):
        """Step 2: Runs in a background thread to safely search files."""
        parts = entry_string.split(" ", 1)
        if len(parts) < 2:
            self.log("Error: Invalid entry format. Use: path <space> targetfile")
            return

        basepath, targetfile = parts[0], parts[1]
        
        if not Path(basepath).is_dir():
            self.log(f"Error: Directory does not exist -> {basepath}")
            return

        self.log(f"Searching for '{targetfile}' inside '{basepath}'...")
        found_paths = pull_file_path(basepath, targetfile)

        # Send the results back to the GUI thread for presentation
        if found_paths == 0:
            self.log(f"Invalid path configuration for basepath: {basepath}")
        elif found_paths == 2 or not found_paths:
            self.log(f"File not found matching: {targetfile}")
        elif len(found_paths) == 1:
            # Exactly one file found! Proceed to copy in a background task
            self.log(f"Match found! Processing single copy...\n {found_paths}\n")
            self._trigger_background_copy(found_paths[0])
        else:
            # Multiple files found! Direct CustomTkinter to open the selection screen
            self.log(f"Warning: Found {len(found_paths)} matches.")
            # Use .after() to safely open UI components from a background thread
            self.app.after(0, lambda: self._show_selection_popup(found_paths))

    def _show_selection_popup(self, match_list):
        """Step 3: Opens a dedicated window with buttons for resolving conflicts."""
        # Create a clean standalone popup Window object
        popup = ctk.CTkToplevel(self.app)
        popup.title("Select Target File")
        popup.geometry("500x350")
        popup.minsize(400, 250)
        
        # Bring popup to the front of the screen
        popup.attributes("-topmost", True)

        label = ctk.CTkLabel(popup, text="Multiple files matched your query.\nClick the exact file you wish to import:", font=ctk.CTkFont(weight="bold"))
        label.pack(padx=20, pady=15)

        # Create a scrollable frame container to hold the list of matches dynamically
        scroll_frame = ctk.CTkScrollableFrame(popup, width=450, height=200)
        scroll_frame.pack(padx=20, pady=10, fill="both", expand=True)

        # Populate the container dynamically with a custom button for every matching item
        for path_item in match_list:
            # We use a default parameter trick (p=path_item) to preserve the string inside the button click action
            btn = ctk.CTkButton(
                scroll_frame, 
                text=path_item, 
                anchor="w",
                fg_color="#2C3E50",
                hover_color="#34495E",
                command=lambda p=path_item: self._on_user_selection(p, popup)
            )
            btn.pack(padx=10, pady=5, fill="x")

    def _on_user_selection(self, selected_path, popup_window):
        """Step 4: Executed when a user clicks their choice button."""
        self.log(f"User selected: {Path(selected_path).name}")
        popup_window.destroy() # Close the selection popup window
        
        # Trigger the physical copy on a background thread
        self._trigger_background_copy(selected_path)

    def _trigger_background_copy(self, file_path):
        """Copies the selected file inside a worker thread so the UI stays smooth."""
        def copy_worker():
            shared_folder = nail_folder_location("shared")
            if shared_folder:
                try:
                    copy_file(file_path, shared_folder)
                    self.log(f"Successfully imported: {Path(file_path).name} -> Shared Folder")
                except Exception as e:
                    self.log(f"Write Failure: Could not copy file. {e}")
            else:
                self.log("Error: Target 'shared' subfolder reference could not be resolved.")

        threading.Thread(target=copy_worker, daemon=True).start()