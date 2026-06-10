from pathlib import Path
import os
import shutil #for filter_file_extension()
import datetime #for get_file_metadata()


def get_project_root() -> Path | None:
    """
    The script should be two subfolders deep from the
    project root
    Function returns the project root

    __file__ is the script path
    .resolve() makes it absolute
    .parents[2] moves up 3 levels (file -> folder -> folder -> root)
    """

    path = Path(__file__).resolve().parents[2]
    if not path:
        return None

    return path


"""
function to look for file and return location
params: 
  ->reference file name
-traverses the whole project, from folder to folder, file to file
-if a file, compares the file names
-if file is the target, return path to the file
"""
def nail_file_location(reference_name: str) -> Path | None:
    root_path = get_project_root()
    
    """
    py: validate directory exists
    C: you should use 'stat()' from <sys/stat.h> to check
       using (IS_DIR(statbuf.st_mode))
    """
    if not root_path.is_dir():
        #print(f"Path project root : {root_path} is invalid......")
        return None

    #print(f"Locating the file : {reference_name}")

    """
    py: rglob returns a generator. Give it the reference name,
     searches recursively
    C: you should use opendir() and readdir() from <dirent.h> to
      write a recursive function that does this.
      On hitting subfolder condition 'if entry->d_type == DT_DIR'
       call the search function again -> recurse.
    """
    for item in root_path.rglob(reference_name):
        """
        py: ensures the name file with the name if found
        C: use 'entry->d_type == DT_REG' to check whether file inside the readdir loop
        """
        if item.is_file():
            #return the path to the file
            return item 
    return None

#search for a folder and return the path
def nail_folder_location(folder_name: str) -> Path | None:
    root_path = get_project_root()

    if not root_path.is_dir():
        #print(f"Path project root : {root_path} is invalid......")
        return None

    #print(f"Locating the folder : {folder_name}")

    # Python: Still uses rglob, but we will filter specifically for directories
    for item in root_path.rglob(folder_name):
        # Py: Ensures the matched item is actually a directory/folder
        # C: In your readdir loop, check if 'entry->d_type == DT_DIR'.
        if item.is_dir():
            return item  # Return the Path object to the folder
            
    return None


def search_root_subfolder(rsubfodlername: str)-> Path | None:
    rsubfolder_location = nail_folder_location(rsubfodlername)
    if not rsubfolder_location:
        print(f"Server: Failed to get {rsubfodlername} \n Attempting to create")

        root = get_project_root()
        if not root:
            print(f"Server: Failed to get root path,  what was obatained {root}")
            exit(1)

        rsubfolder_location = root / rsubfodlername
        rsubfolder_location.mkdir(parents=True, exists_ok=True)
        if not rsubfolder_location.is_dir():
            return None

        print("Folder created: location -> {rsubfolder_location}")
        return rsubfolder_location
    return rsubfolder_location


#NOT ACTIVELY USED
def traverse_project(root_path: str):

    if not root_path.is_dir():
        print(f"The path {root_path}, is not valid")
        return

    print("Traversing the project......")

    # rrglob("*")recursively finds all files and subfolders
    for item in root_path.rglob("*"):
        if item.is_dir():
            print(f"[Folder] {item.relative_to(root_path)}")

        elif item.is_file():
            file_size_kb = item.stat().st_size / 1024
            print(f"[File] {item.relative_to(root_path)} : Size {file_size_kb:.2f} KB")

#Method 2
def traverse_project2(root_path: str) -> None:
    #using os.walk, it navigates top down through all the subfolders
    for root, dirs, files in os.walk(root_path):
        print(f"Scanning {root_path}")

        #show subfolders in current root
        for sub_dir in dirs:
            print(f"Subfolder : {sub_dir}")

        #show files in current root
        for file in files:
            #full_path = os.path.join(root, file)

            print("File : {file}")

def filter_files_by_extension(folder_path: str, extension: str) -> list[Path]:
    """Finds all files matching a specific extension (e.g., '.txt') recursively."""
    root = Path(folder_path)
    if not root.is_dir():
        return []
    
    # Python uses the C standard library's 'dirent.h' structures (like readdir())
    # to scan directories under the hood during recursion.
    return list(root.rglob(f"*{extension}"))


def delete_path(target_path: str):
        """Safely deletes a file or an entire directory tree."""
        path = Path(target_path)
        
        if not path.exists():
            print("Path does not exist.")
            return

        if path.is_file():
            # C UNDER THE HOOD: Calls the standard C library 'unlink()' function
            path.unlink()
            print(f"Deleted file: {path}")
        elif path.is_dir():
            # C UNDER THE HOOD: Calls 'rmdir()' for empty folders. 
            # shutil.rmtree recursively calls unlink() and rmdir() at the C level.
            shutil.rmtree(path)
            print(f"Deleted directory and all contents: {path}")


def move_path(source_path: str, destination_path: str):
    """Moves a file or a folder to a new location."""
    src = Path(source_path)
    dst = Path(destination_path)
    
    # Ensure destination parent folder exists, create it if missing
    dst.parent.mkdir(parents=True, exist_ok=True)
    
    # C UNDER THE HOOD: If source and destination are on the same drive,
    # Python executes a zero-copy atomic move using the C 'rename()' function.
    shutil.move(str(src), str(dst))
    print(f"Moved {src} to {dst}")


def copy_path(source_path: str, destination_path: str):
    """Copies a file or an entire folder structure to a new destination."""
    src = Path(source_path)
    dst = Path(destination_path)
    
    if src.is_file():
        dst.parent.mkdir(parents=True, exist_ok=True)
        # C UNDER THE HOOD: Standard file copy reads blocks of data into 
        # memory buffers and writes them out using C 'read()' and 'write()' systems.
        shutil.copy2(src, dst) # copy2 preserves original file metadata
        print(f"Copied file to: {dst}")
        
    elif src.is_dir():
        # Recursively copies entire directory trees
        shutil.copytree(src, dst, dirs_exist_ok=True)
        print(f"Copied directory tree to: {dst}")



def get_file_metadata(file_path: str) -> dict:
    """Retrieves size and timestamps for a specific file."""
    path = Path(file_path)
    if not path.is_file():
        return {}
        
    # C UNDER THE HOOD: Populates data by executing the C 'stat()' system call.
    stat_info = path.stat()
    
    return {
        "size_bytes": stat_info.st_size, # C: stat_info.st_size
        "last_modified": datetime.datetime.fromtimestamp(stat_info.st_mtime), # C: st_mtime
        "last_accessed": datetime.datetime.fromtimestamp(stat_info.st_atime)  # C: st_atime
    }



if __name__ == '__main__':
    start_path = get_project_root()
    file_list = filter_files_by_extension(start_path, "json")
    stringed = [str(x) for x in file_list]
    if stringed:
        print(f"files -> {stringed}")
    else:
        print("File list not received......")
