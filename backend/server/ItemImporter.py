from pathlib import Path
import shutil
import difflib
import errno
import os


def pull_close_files(basepath: str, targetfile: str, cutoff: float = 0.6) -> list:
    """
    Finds files with names similar to the target file.
    Returns a list of Path objects.
    """
    starting_point = Path(basepath)
    if not starting_point.is_dir():
        print(f"The path {starting_point} is not a valid one")
        return []

    print(f"Searching for files close to '{targetfile}'...")

    # 1. Find all actual files in the directory recursively
    all_files = [p for p in starting_point.rglob("*") if p.is_file()]

    # 2. Extract just the file names for comparison
    filenames = [p.name for p in all_files]

    # 3. Find the close matches among the filenames
    # cutoff = 0.6 means 60% similarity match minimum
    close_names = difflib.get_close_matches(
        targetfile, filenames, n=10, cutoff=cutoff)

    # 4. Map the matching filenames back to their full Path objects
    matched_paths = []
    for name in close_names:
        for p in all_files:
            if p.name == name:
                matched_paths.append(p)
                break  # each name only once

    return matched_paths


def pull_substring_files(basepath: str, targetfile: str) -> list:
    """
    Returns any file where the target string exists anywhere inside the filename.
    """
    starting_point = Path(basepath)
    target_lower = targetfile.lower()

    return [
        p for p in starting_point.rglob("*")
        if p.is_file() and target_lower in p.name.lower()
    ]


def copy_file(source_path: str, destination_path: str) -> int:
    """
    Copies a file to a new destination, returning a specific error code.

    Returns:
        0: Success
        1: Source not found (ENOENT)
        2: Permission denied (EACCES)
        3: Source is a directory, expected file (EISDIR)
        4: Disk full (ENOSPC)
        5: Generic I/O error (EIO)
    """
    src = Path(source_path)
    dst = Path(destination_path)

    # LOW-LEVEL CHECK: Verify source existence before allocating buffers
    if not src.exists():
        return 1

    # Ensure we are handling a file, not a directory
    if src.is_dir():
        return 3

    try:
        # Create destination parent directories
        dst.parent.mkdir(parents=True, exist_ok=True)

        # Standard file copy reads blocks of data into memory buffers
        # and writes them out using OS read()/write() systems.
        shutil.copy2(src, dst)  # Preserves metadata (utime, permissions)
        return 0

    except PermissionError:
        # Triggered by restricted file permissions or locked handles
        return 2

    except IsADirectoryError:
        # Triggered if destination path conflicts with an existing folder
        return 3

    except OSError as e:
        # Check system error numbers for disk fullness
        if e.errno == errno.ENOSPC:
            return 4
        # Fallback for unhandled low-level hardware or filesystem errors
        return 5


def pull_file_path(basepath: str, targetfile: str) -> list:
    """
    Searches for files matching the target filename.
    Returns a list of matching file paths (strings), or NOFILE (0) if none found.
    """
    starting_point = Path(basepath)
    if not starting_point.is_dir():
        print(f"The path {starting_point} is not a valid one")
        return 0  # INVALIDPATH

    print("Attempting searching for file")

    # Use rglob to find files matching the target name
    # Exact name match: item.name == targetfile
    matching_list = [
        str(item) for item in starting_point.rglob(targetfile)
        if item.is_file() and item.name == targetfile
    ]

    if len(matching_list) < 1:
        return 0  # NOFILE

    return matching_list


# Legacy compatibility functions (kept for reference)
def pull_close_files_legacy(basepath: str, targetfile: str, cutoff: float = 0.6) -> list:
    """Legacy version using substring matching."""
    all_files = [p for p in Path(basepath).rglob("*") if p.is_file()]
    filenames = [p.name for p in all_files]
    close_names = difflib.get_close_matches(targetfile, filenames, n=10, cutoff=cutoff)
    matched_paths = []
    for name in close_names:
        for p in all_files:
            if p.name == name:
                matched_paths.append(p)
                break
    return matched_paths


def pull_substring_files_legacy(basepath: str, targetfile: str) -> list:
    """Legacy version searching for substring in filename."""
    target_lower = targetfile.lower()
    return [
        p for p in Path(basepath).rglob("*")
        if p.is_file() and target_lower in p.name.lower()
    ]


if __name__ == "__main__":
    pass