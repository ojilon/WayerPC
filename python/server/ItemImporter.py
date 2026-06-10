from pathlib import Path
from Locate import nail_folder_location
import shutil
import difflib

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
    close_names = difflib.get_close_matches(targetfile, filenames, n=10, cutoff=cutoff)
    
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


#actively used
def copy_file(source_path: str, destination_path: str):
    """Copies a file or an entire folder structure to a new destination."""
    src = Path(source_path)
    dst = Path(destination_path)
    
    if src.is_file():
        dst.parent.mkdir(parents=True, exist_ok=True)
        # C UNDER THE HOOD: Standard file copy reads blocks of data into 
        # memory buffers and writes them out using C 'read()' and 'write()' systems.
        shutil.copy2(src, dst) # copy2 preserves original file metadata
        print(f"Copied file to: {dst}")


def pull_file_path(basepath: str, targetfile: str) -> Path | None:
	starting_point = Path(basepath)
	if not starting_point.is_dir():
		print(f"The path {starting_point} is not a valid one")
		return None

	print("Attempting searching for file")

	matching_list = list(starting_point.rglob(targetfile))
	cleaned_list = [Path(x) for x in matching_list]
	return cleaned_list


def pull_file():
	entry = input("Enter folder path and file name seperately \n >")
	parts = entry.split(" ", 2)

	if parts.__len__() < 2:
		print("Usage: path <space> targetfile")
		exit(1)
	basepath = parts[0]
	if not Path(basepath).is_dir():
		print("Usage: path <space> targetfile")
		exit(1)
	targetfile = parts[1]

	found_path = pull_file_path(basepath, targetfile)
	if found_path.__len__() < 2 and found_path:
		shared_folder = nail_folder_location("shared")
		if shared_folder:
			copy_file(found_path[0], shared_folder)
		

if __name__ == "__main__":
	pull_file()