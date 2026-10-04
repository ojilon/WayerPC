#ifndef FILE_SEARCH_H
#define FILE_SEARCH_H

// Include the C-compatible DLL interface
#include <windows.h>

// Return codes for Python to differentiate errors
#define STATUS_SUCCESS 0
#define STATUS_DIR_ERROR -1
#define STATUS_NOT_FOUND -2
#define STATUS_DLL_LOAD_ERROR -3
#define STATUS_DLL_FUNC_ERROR -4

// Initialize the filesearch DLL (call once at startup)
void search_file_init(void);

// Clean up the filesearch DLL (call once at shutdown)
void search_file_cleanup(void);

// Search for a file using the backend/filesearch C++23 implementation via DLL
// Returns: STATUS_SUCCESS, STATUS_DIR_ERROR, STATUS_NOT_FOUND, etc.
int search_file(const char* location_of_shared, const char* filename, char* project_root, char* out_path, size_t max_len);

#endif