#include <stdio.h>
#include <string.h>
#include <dirent.h>

// Return codes for Python to differentiate errors
#define STATUS_SUCCESS 0
#define STATUS_DIR_ERROR -1
#define STATUS_NOT_FOUND -2

int search_file(const char*  location_of_shared, const char* filename,char*project_root, char* out_path, size_t max_len) {
     
    DIR* dir = opendir(location_of_shared);

    // Case: directory itself couldn't be opened
    if (dir == NULL) {
        return STATUS_DIR_ERROR; 
    }

    struct dirent* entry;
    int found = 0;

    // Standard while loop instead of do-while
    while ((entry = readdir(dir)) != NULL) {
        // Skip current and parent directory shortcuts
        if (strcmp(entry->d_name, ".") == 0 || strcmp(entry->d_name, "..") == 0) {
            continue;
        }

        if (strcmp(entry->d_name, filename) == 0) {
            snprintf(out_path, max_len, "%s\\%s", location_of_shared, entry->d_name);
            found = 1;
            break; 
        }
    }

    closedir(dir);

    // Case 2: Directory read fine, but file isn't there
    if (found) {
        return STATUS_SUCCESS;
    } else {
        return STATUS_NOT_FOUND;
    }
}
