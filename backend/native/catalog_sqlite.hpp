#pragma once

#include <cstddef>

#ifdef _WIN32
#define WAYER_EXPORT __declspec(dllexport)
#else
#define WAYER_EXPORT
#endif

#ifdef __cplusplus
extern "C" {
#endif

// Native SQLite catalog (optional). Python sqlite3 is the primary interface;
// this DLL talks to the same imports.db file when the amalgamation is built in.

WAYER_EXPORT int catalog_open(const char* db_path);
WAYER_EXPORT int catalog_add(const char* path);
WAYER_EXPORT int catalog_remove(const char* path);
WAYER_EXPORT int catalog_list(char* out_buf, size_t max_len);
WAYER_EXPORT int catalog_find(const char* query, int min_percent, char* out_buf, size_t max_len);
WAYER_EXPORT void catalog_close(void);

#ifdef __cplusplus
}
#endif
