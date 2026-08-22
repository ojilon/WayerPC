import json
import os
import sys
import ctypes
from pathlib import Path

from backend.native.dll_loader import sync_native_libs_to_app_folder


def load_saved_paths(CONFIG_FILE: str) -> dict[str, str] | None:
    """Reads saved paths from the config file if it exists."""
    if Path(CONFIG_FILE).exists():
        try:
            with open(CONFIG_FILE, "r", encoding="utf-8") as f:
                return json.load(f)
        except (json.JSONDecodeError, OSError):
            return None
    return None

APP_NAME = "WayerPC"
CONFIG_DIR = Path(os.environ.get("LOCALAPPDATA", Path.home())) / APP_NAME
CONFIG_FILE = CONFIG_DIR / "config.json"


def share_path_to_config() -> Path:
    return CONFIG_FILE


def get_available_drives() -> list[Path]:
    """Retrieves all active drive letters on Windows using Win32 API."""
    drives = []
    bitmask = ctypes.windll.kernel32.GetLogicalDrives()
    for letter in "ABCDEFGHIJKLMNOPQRSTUVWXYZ":
        if bitmask & 1:
            drives.append(Path(f"{letter}:\\"))
        bitmask >>= 1
    return drives


def save_paths(app_dir: Path, sub_dir: Path) -> None:
    """Saves the app directory and subfolder paths to JSON."""
    CONFIG_DIR.mkdir(parents=True, exist_ok=True)
    data = {
        "app_dir": str(app_dir.resolve()),
        "sub_dir": str(sub_dir.resolve()),
    }
    with open(CONFIG_FILE, "w", encoding="utf-8") as f:
        json.dump(data, f, indent=4)


def create_app_structure(drive: Path, subfolder_name: str = "Data") -> tuple[Path, Path]:
    """Creates the root app folder and a subfolder inside the selected drive."""
    app_dir = drive / APP_NAME
    sub_dir = app_dir / subfolder_name

    # Create directories (parents=True ensures safety, exist_ok=True prevents errors)
    app_dir.mkdir(parents=True, exist_ok=True)
    sub_dir.mkdir(parents=True, exist_ok=True)

    # Save paths for future application runs
    save_paths(app_dir, sub_dir)
    sync_native_libs_to_app_folder()
    return app_dir, sub_dir


def check_app_folder_exists(drive: Path) -> bool:
    """Checks if the App folder already exists on a specific drive."""
    return (drive / APP_NAME).is_dir()


def prompt_drive_selection(drives: list[Path]) -> Path:
    """CLI prompt for selecting a target drive."""
    print("\n--- Drive Selection ---")
    print("Available drives:")
    for i, drive in enumerate(drives, 1):
        print(f"  [{i}] {drive}")

    while True:
        choice = input(f"Select a drive number (1-{len(drives)}): ").strip()
        if choice.isdigit():
            idx = int(choice) - 1
            if 0 <= idx < len(drives):
                return drives[idx]
        print("Invalid selection. Please try again.")


def get_app_path() -> Path:
    """Returns the app directory path from saved config or raises error."""
    CONFIG_FILE = Path(os.environ.get("LOCALAPPDATA", Path.home())) / "WayerPC" / "config.json"
    saved = load_saved_paths(CONFIG_FILE)
    if saved and Path(saved["app_dir"]).exists():
        return Path(saved["app_dir"])
    raise FileNotFoundError("Storage not initialized. Run setup_storage first.")


def get_subfolder_path() -> Path:
    """Returns the data subfolder path from saved config."""
    CONFIG_FILE = Path(os.environ.get("LOCALAPPDATA", Path.home())) / "WayerPC" / "config.json"
    saved = load_saved_paths(CONFIG_FILE)
    if saved and Path(saved["sub_dir"]).exists():
        return Path(saved["sub_dir"])
    raise FileNotFoundError("Storage not initialized. Run setup_storage first.")


def setup_storage(default_drive_letter: str = "D:\\") -> tuple[Path, Path]:
    """Startup initialization routine called by main.py."""

    # 1. Check if config already exists from a previous run
    if CONFIG_FILE.exists():
        try:
            with open(CONFIG_FILE, "r", encoding="utf-8") as f:
                saved = json.load(f)
            app_path = Path(saved["app_dir"])
            sub_path = Path(saved["sub_dir"])
            if app_path.exists():
                sync_native_libs_to_app_folder()
                return app_path, sub_path
        except (json.JSONDecodeError, OSError):
            pass

    # 2. Get active system drives
    drives = get_available_drives()
    if not drives:
        sys.exit("Error: No drive letters detected on this system.")

    target_drive = None
    default_path = Path(default_drive_letter)

    # 3. Check if preferred drive (D:\) exists
    if default_path in drives:
        target_drive = default_path
    else:
        print(f"Notice: Preferred drive '{default_drive_letter}' was not found.")
        target_drive = prompt_drive_selection(drives)

    # 4. Attempt structure creation on chosen drive with fallback prompt
    while True:
        try:
            return create_app_structure(target_drive)
        except OSError as e:
            print(f"\nFailed to create folders on {target_drive}: {e}")
            print("Please select another drive.")
            target_drive = prompt_drive_selection(drives)
