#pragma once

#include <cstddef>

#ifdef _WIN32
#define WAYER_EXPORT __declspec(dllexport)
#else
#define WAYER_EXPORT
#endif

extern "C" {
    // Recursively search user storage (Documents, Downloads, Desktop, extra
    // drives). Skips Windows/system directories and AppData.
    //
    // query: file or folder name fragment (case-insensitive)
    // out_buf: filled with "name<TAB>path" lines, newline-separated, UTF-8
    //
    // Returns number of matches written, 0 if none, -1 on argument error.
    WAYER_EXPORT int search_user_storage(const char* query, char* out_buf, size_t max_len);

    // Similarity of two names in [0, 100]. Used for the 20% related-name filter.
    WAYER_EXPORT int name_similarity_percent(const char* a, const char* b);
}
