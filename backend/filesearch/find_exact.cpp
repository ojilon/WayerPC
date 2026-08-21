#include "find_exact.hpp"

namespace Backend::Utils::FileFindExact {

    std::expected<fs::path, SearchStatus> search_file(const fs::path& search_dir, const std::string& filename) {
        std::error_code ec;
        
        // Validate directory existence and permissions safely without throwing exceptions
        if (!fs::exists(search_dir, ec) || !fs::is_directory(search_dir, ec)) {
            return std::unexpected(SearchStatus::DirectoryError);
        }

        // Configured to safely bypass folders that throw permission errors
        auto iter_options = fs::directory_options::skip_permission_denied;
        
        for (const auto& entry : fs::recursive_directory_iterator(search_dir, iter_options, ec)) {
            // Handle iteration errors mid-loop (e.g. unreadable nested folder)
            if (ec) {
                continue; 
            }

            if (entry.is_regular_file(ec) && entry.path().filename() == filename) {
                return entry.path();
            }
        }

        return std::unexpected(SearchStatus::FileNotFound);
    }
}

extern "C" {

bool search_file_c(const char* search_dir, const char* filename, char* out_path, size_t max_len) {
    if (!search_dir || !filename || !out_path) {
        return false;
    }

    std::string dir_str(search_dir);
    std::string file_str(filename);

    auto result = search_file(fs::path(dir_str), file_str);

    if (result) {
        // Success - copy the path to out_buffer
        std::string path_str = result.value().string();
        size_t copy_len = path_str.length() < max_len ? path_str.length() : max_len - 1;
        memcpy(out_path, path_str.c_str(), copy_len);
        out_path[copy_len] = '\0';
        return true;
    } else {
        // Failure - set empty string
        out_path[0] = '\0';
        return false;
    }
}

}