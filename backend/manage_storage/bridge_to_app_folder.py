import json
import shutil
from pathlib import Path
import os

APP_NAME = "WayerPC"
CONFIG_DIR = Path(os.environ.get("LOCALAPPDATA", Path.home())) / APP_NAME
CONFIG_FILE = CONFIG_DIR / "config.json"


def share_path_to_config() -> Path:
    return CONFIG_FILE


def load_saved_paths(CONFIG_FILE: str) -> dict[str, str] | None:
    """Reads saved paths from the config file if it exists."""
    if Path(CONFIG_FILE).exists():
        try:
            with open(CONFIG_FILE, "r", encoding="utf-8") as f:
                return json.load(f)
        except (json.JSONDecodeError, OSError):
            return None
    return None


def get_app_folder_path() -> Path:
    """Returns the primary App directory path."""
    saved = load_saved_paths(share_path_to_config())
    if not saved or not Path(saved["app_dir"]).exists():
        raise FileNotFoundError("Storage setup has not been initialized.")
    return Path(saved["app_dir"])


def get_data_subfolder_path() -> Path:
    """Returns the primary Subfolder path inside the App folder."""
    saved = load_saved_paths(share_path_to_config())
    if not saved or not Path(saved["sub_dir"]).exists():
        raise FileNotFoundError("Storage setup has not been initialized.")
    return Path(saved["sub_dir"])


def create_new_data_subfolder(subfoldername: str):
    path_to_data_subfolder = get_data_subfolder_path()
    new_data_subfolder_path = path_to_data_subfolder / subfoldername
    new_data_subfolder_path.mkdir(parents=True, exist_ok=True)


def save_path_to_new_data_subfolder(new_data_subfolder_dir: Path, subfoldername: str):
    path_to_config = share_path_to_config()
    data_subfolder_dir = {f"{subfoldername}": str(new_data_subfolder_dir.resolve)}

    with open(path_to_config, encoding="utf-8") as f:
        json.dump(data_subfolder_dir, f, indent=4)


def ensure_shared_folder() -> Path:
    """Ensures the 'shared' folder exists inside the app folder.
    If not, creates an empty 'shared' folder and saves the path."""
    app_dir = get_app_folder_path()
    shared_path = app_dir / "shared"
    
    if not shared_path.is_dir():
        shared_path.mkdir(parents=True, exist_ok=True)
        # Save that shared folder was created
        save_path_to_new_data_subfolder(shared_path, "shared")
    
    return shared_path


def copy_dll_to_app_folder(dll_source: str) -> Path | None:
    """Copies a dll file to the app folder for development testing.
    Returns the destination path if successful, None otherwise."""
    app_dir = get_app_folder_path()
    dest_path = app_dir

    if dll_source is None:
        return None
    dll_source_path = Path(dll_source)
    
    try:
        if dll_source_path.is_file():
            shutil.copy2(dll_source_path, dest_path)
            return dest_path
        elif dll_source_path.is_dir():
            dest_dir = app_dir / Path(dll_source_path).name
            if dest_dir.is_dir():
                shutil.rmtree(dest_dir)
            shutil.copytree(dll_source_path, dest_dir, dirs_exist_ok=True)
            return dest_dir
    except OSError as e:
        print(f"Failed to copy dll: {e}")
    return None


def obtain_folder_location(folder_name: str) -> tuple:
    """
    Searches for a folder by name starting from the app folder.
    Returns (status, path) where:
    - status: 0 = found, 1 = not found, 2 = using app root as fallback
    - path: the Path object to the folder (or app root if not found)
    """
    app_dir = get_app_folder_path()
    if app_dir is None or not app_dir.is_dir():
        return 1, Path(".")

    # Search recursively for the folder within app directory
    for item in app_dir.rglob(folder_name):
        if item.is_dir():
            return 0, item

    #create an empty folder 'shared'
    if folder_name == 'shared':
        return 0, ensure_shared_folder()

    #folder not found
    return 2, app_dir

def obtain_file_location(filename: str) -> tuple:
    app_dir = get_app_folder_path()
    if app_dir is None or not app_dir.is_dir():
        return 1, Path(".")

    for item in app_dir.rglob(filename):
        if item.is_file():
            return 0, item

    return 2, app_dir