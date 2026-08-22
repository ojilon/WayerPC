#pragma once

#include <cstddef>
#include <expected>
#include <filesystem>
#include <string>

namespace Backend::Utils::FileFindExact {
    namespace fs = std::filesystem;

    enum class SearchStatus {
        DirectoryError,
        FileNotFound
    };

    std::expected<fs::path, SearchStatus> search_file(const fs::path& search_dir, const std::string& filename);
}

#ifdef _WIN32
#define BACKEND_FILESEARCH_EXPORT __declspec(dllexport)
#else
#define BACKEND_FILESEARCH_EXPORT
#endif

extern "C" {
    // Matches the Python ctypes binding:
    // search_file(shared, filename, project_root, out_buf, max_len) -> int
    //  0 found, -1 directory error, -2 not found
    BACKEND_FILESEARCH_EXPORT int search_file(
        const char* shared_dir,
        const char* filename,
        const char* project_root,
        char* out_path,
        size_t max_len
    );

    BACKEND_FILESEARCH_EXPORT bool search_file_c(
        const char* search_dir,
        const char* filename,
        char* out_path,
        size_t max_len
    );
}
