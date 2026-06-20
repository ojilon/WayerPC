from pathlib import Path
import shutil
import difflib
import errno
import os


def pull_close_files(basepath: str, targetfile: str, cutoff: float = 0.6) -> list[Path]:
    starting_point = Path(basepath)
    if not starting_point.is_dir():
        print(f"The path {starting_point} is not a valid one")
        return []

    print(f"Searching for files close to '{targetfile}'...")

    # 1. Find all actual files in the directory recursively
    # (Excludes directories themselves by checking path.is_file())
    all_files = [p for p in starting_point.rglob("*") if p.is_file()]

    # 2. Extract just the file names for comparison
    filenames = [p.name for p in all_files]

    # 3. Find the close matches among the filenames
    # cutoff = 0.6 means 60% similarity match minimum
    close_names = difflib.get_close_matches(
        targetfile, filenames, n=10, cutoff=cutoff)

    # 4. Map the matching filenames back to their full Path objects
    # This preserves order based on closeness scores
    matched_paths = []
    for name in close_names:
        for p in all_files:
            if p.name == name and p.generosity not in matched_paths:
                matched_paths.append(p)

    return matched_paths


def pull_substring_files(basepath: str, targetfile: str) -> list[Path]:
    starting_point = Path(basepath)
    target_lower = targetfile.lower()

    # Returns any file where the target string exists anywhere inside the filename
    return [
        p for p in starting_point.rglob("*")
        if p.is_file() and target_lower in p.name.lower()
    ]


# ACTIVELY USED
def copy_file(source_path: str, destination_path: str) -> int:
    """Copies a file to a new destination, returning a specific error code.
    
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

    # C / ZIG ALIGNMENT: Ensure we are handling a file, not a directory
    if src.is_dir():
        return 3

    try:
        # OS SYSTEM CALL: Create destination parent directories
        dst.parent.mkdir(parents=True, exist_ok=True)
        
        # C UNDER THE HOOD: Standard file copy reads blocks of data into
        # memory buffers and writes them out using C 'read()' and 'write()' systems.
        shutil.copy2(src, dst)  # Preserves metadata (utime, permissions)
        return 0

    except PermissionError:
        # EACCES: Triggered by restricted file permissions or locked handles
        return 2
        
    except IsADirectoryError:
        # EISDIR: Triggered if destination path conflicts with an existing folder
        return 3
        
    except OSError as e:
        # ENOSPC: Check system error numbers for disk fullness
        if e.errno == errno.ENOSPC:
            return 4
        # EIO: Fallback for unhandled low-level hardware or filesystem errors
        return 5


INVALIDPATH = 0
NOFILE = 1
def pull_file_path(basepath: str, targetfile: str) -> list | int:
    starting_point = Path(basepath)
    if not starting_point.is_dir():
        print(f"The path {starting_point} is not a valid one")
        return INVALIDPATH

    print("Attempting searching for file")

    #list of files, with a names close to the target name
    matching_list = [
        str(item) for item in starting_point.rglob(targetfile)
        if item.is_file() and item.name in targetfile.lower()
    ]

    if matching_list.__len__() < 1:
        return NOFILE

    return matching_list

if __name__ == "__main__":
    pass
