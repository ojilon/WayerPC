#include <stdio.h>
#include <string.h>
#include <dirent.h>
#include <windows.h>

// Return codes for Python to differentiate errors
#define STATUS_SUCCESS 0
#define STATUS_DIR_ERROR -1
#define STATUS_NOT_FOUND -2
#define STATUS_DLL_LOAD_ERROR -3
#define STATUS_DLL_FUNC_ERROR -4

// Forward declaration for the DLL search function
typedef int (__stdcall *SEARCH_FILE_FUNC)(const char*, const char*, char*, size_t);

static HMODULE g_filesearch_dll = NULL;
static SEARCH_FILE_FUNC g_search_file_func = NULL;

// Initialize the DLL and function pointer
static bool init_filesearch_dll() {
    if (g_filesearch_dll) {
        return true; // Already initialized
    }

    g_filesearch_dll = LoadLibraryA("libfilesearch.dll");
    if (!g_filesearch_dll) {
        return false;
    }

    g_search_file_func = (SEARCH_FILE_FUNC)GetProcAddress(g_filesearch_dll, "search_file_c");
    if (!g_search_file_func) {
        FreeLibrary(g_filesearch_dll);
        g_filesearch_dll = NULL;
        return false;
    }

    return true;
}

// Clean up DLL on exit
static void cleanup_filesearch_dll() {
    if (g_filesearch_dll) {
        FreeLibrary(g_filesearch_dll);
        g_filesearch_dll = NULL;
        g_search_file_func = NULL;
    }
}

int search_file(const char* location_of_shared, const char* filename, char* project_root, char* out_path, size_t max_len) {
    // Try to use the backend/filesearch C++ DLL first
    if (!init_filesearch_dll()) {
        // Fallback to C native search if DLL not available
        (void)project_root; // unused in fallback
        
        DIR* dir = opendir(location_of_shared);
        if (dir == NULL) {
            return STATUS_DIR_ERROR;
        }

        struct dirent* entry;
        int found = 0;

        while ((entry = readdir(dir)) != NULL) {
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

        if (found) {
            return STATUS_SUCCESS;
        } else {
            return STATUS_NOT_FOUND;
        }
    }

    // Use the C++23 backend/filesearch implementation via C-compatible API
    // Convert inputs and call the DLL function
    char out_buf[260] = {0};
    int result = g_search_file_func(location_of_shared, filename, out_buf, 260);

    if (result && out_buf[0] != '\0') {
        snprintf(out_path, max_len, "%s", out_buf);
        return STATUS_SUCCESS;
    } else {
        out_path[0] = '\0';
        return STATUS_NOT_FOUND;
    }
}

// Call this once at program startup to initialize the DLL
void search_file_init(void) {
    init_filesearch_dll();
}

// Call this once at program shutdown to clean up
void search_file_cleanup(void) {
    cleanup_filesearch_dll();
}