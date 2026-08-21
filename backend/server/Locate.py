from pathlib import Path
import os
import shutil
import datetime


def get_project_root() -> Path | None:
    """
    Returns the project root by going up 3 levels from this script.
    __file__ is the script path; .resolve() makes it absolute.
    .parents[2] moves up 3 levels (file -> folder -> folder -> root).
    """
    path = Path(__file__).resolve().parents[2]
    if not path:
        return None
    return path


def nail_folder_location(folder_name: str) -> tuple:
    """
    Searches for a folder by name starting from the project root.
    Returns (status, path) where:
    - status: 0 = found, 1 = not found, 2 = using root as fallback
    - path: the Path object to the folder (or root if not found)
    """
    root_path = get_project_root()
    if root_path is None or not root_path.is_dir():
        return 1, Path(".")

    # Search recursively for the folder
    for item in root_path.rglob(folder_name):
        if item.is_dir():
            return 0, item

    # Not found - return root as fallback
    return 2, root_path


def search_root_subfolder(rsubfodlername: str) -> Path | None:
    """Returns the folder path if it exists, otherwise None."""
    status, rsubfolder_location = nail_folder_location(rsubfodlername)
    if status == 0:
        return rsubfolder_location
    return None


def traverse_folder_for_paths(folder_location: Path) -> list | None:
    """Returns a list of all paths under a folder."""
    if not folder_location.is_dir():
        return None

    path_list = [item for item in folder_location.rglob("*")]
    cleaned_list = [str(p) for p in path_list]
    return cleaned_list


def get_file_metadata(file_path: str) -> dict | None:
    """Retrieves name, type, size, and local M/D/Y modification date for a file."""
    path = Path(file_path)
    if not path.is_file():
        return None

    # Retrieve filesystem data via stat()
    stat_info = path.stat()

    # Convert st_mtime using localtime then format via strftime()
    dt_local = datetime.datetime.fromtimestamp(stat_info.st_mtime, tz=None)
    formatted_date = dt_local.strftime("%m/%d/%Y %I:%M %p")

    return {
        "file_name": path.name,           # Extract from path string / dirent.d_name
        "file_type": path.suffix,         # Extract extension from path string
        "size_bytes": stat_info.st_size,  # stat_info.st_size
        "date_modified": formatted_date   # Formatted local timestamp string
    }


def get_file_info(folder_name: str) -> list | None:
    """
    Gets file info for a folder ('shared' or 'received').
    Returns list of metadata dicts or None if folder doesn't exist.
    """
    folder_path = Path(folder_name)
    if not folder_path.is_dir():
        return None

    status, folder_location = nail_folder_location(folder_name)
    if status != 0:
        return None

    path_list = traverse_folder_for_paths(folder_location)
    if path_list is None:
        return None

    metadata_list = [get_file_metadata(d) for d in path_list]
    return metadata_list


# NOT ACTIVELY USED
def traverse_project(root_path: Path):
    """Traverses the project and prints folder/file info."""
    if not root_path.is_dir():
        print(f"The path {root_path}, is not valid")
        return

    print("Traversing the project......")

    # Recursively find all files and subfolders
    for item in root_path.rglob("*"):
        if item.is_dir():
            print(f"[Folder] {item.relative_to(root_path)}")

        elif item.is_file():
            try:
                file_size_kb = item.stat().st_size / 1024
                print(f"[File] {item.relative_to(root_path)} : Size {file_size_kb:.2f} KB")
            except OSError:
                pass


def filter_files_by_extension(folder_path: str, extension: str) -> list:
    """Finds all files matching a specific extension (e.g., '.txt') recursively."""
    root = Path(folder_path)
    if not root.is_dir():
        return []

    return list(root.rglob(f"*{extension})"))


def delete_path(target_path: str):
    """Safely deletes a file or an entire directory tree."""
    path = Path(target_path)

    if not path.exists():
        print("Path does not exist.")
        return

    if path.is_file():
        path.unlink()
        print(f"Deleted file: {path}")
    elif path.is_dir():
        shutil.rmtree(path)
        print(f"Deleted directory and all contents: {path}")


def move_path(source_path: str, destination_path: str):
    """Moves a file or a folder to a new location."""
    src = Path(source_path)
    dst = Path(destination_path)

    # Ensure destination parent folder exists, create it if missing
    dst.parent.mkdir(parents=True, exist_ok=True)

    # Move using OS rename (atomic on same drive)
    shutil.move(str(src), str(dst))
    print(f"Moved {src} to {dst}")


def copy_path(source_path: str, destination_path: str):
    """Copies a file or an entire folder structure to a new destination."""
    src = Path(source_path)
    dst = Path(destination_path)

    if src.is_file():
        dst.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(src, dst)  # copy2 preserves original file metadata
        print(f"Copied file to: {dst}")

    elif src.is_dir():
        shutil.copytree(src, dst, dirs_exist_ok=True)
        print(f"Copied directory tree to: {dst}")


if __name__ == '__main__':
    root = get_project_root()
    if root:
        print(f"Project root: {root}")
        # Example: get file info for 'shared' folder
        info = get_file_info("shared")
        if info:
            print(f"Found {len(info)} files in shared folder")
            for f in info[:3]:  # Show first 3
                print(f"  - {f['file_name']} ({f['file_type']}, {f['size_bytes']} bytes)")