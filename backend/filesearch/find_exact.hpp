#pragma once

#include <filesystem>
#include <string>
#include <expected>

namespace Backend::Utils::FileFindExact {
    namespace fs = std::filesystem;

    // Scoped enum for explicit error categorization
    enum class SearchStatus {
        DirectoryError,
        FileNotFound
    };

    /**
     * @brief Recursively searches for a file within a directory.
     * @param search_dir The root directory to start the search from.
     * @param filename The exact name of the file to search for (including extension).
     * @return std::expected containing the cross-platform fs::path on success, 
     *         or a SearchStatus error enum on failure.
     */
    std::expected<fs::path, SearchStatus> search_file(const fs::path& search_dir, const std::string& filename);
}

// C-compatible interface for DLL export
#ifdef _WIN32
#define BACKEND_FILESEARCH_EXPORT __declspec(dllexport)
#else
#define BACKEND_FILESEARCH_EXPORT
#endif

extern "C" {
    BACKEND_FILESEARCH_EXPORT bool search_file_c(const char* search_dir, const char* filename, char* out_path, size_t max_len);
}