#include "find_exact.hpp"

#include <cstring>
#include <string>

namespace Backend::Utils::FileFindExact {

    std::expected<fs::path, SearchStatus> search_file(const fs::path& search_dir, const std::string& filename) {
        std::error_code ec;

        if (!fs::exists(search_dir, ec) || !fs::is_directory(search_dir, ec)) {
            return std::unexpected(SearchStatus::DirectoryError);
        }

        auto iter_options = fs::directory_options::skip_permission_denied;

        for (const auto& entry : fs::recursive_directory_iterator(search_dir, iter_options, ec)) {
            if (ec) {
                ec.clear();
                continue;
            }
            if (entry.is_regular_file(ec) && entry.path().filename() == filename) {
                return entry.path();
            }
        }

        return std::unexpected(SearchStatus::FileNotFound);
    }
}

namespace {

int copy_path(const std::filesystem::path& path, char* out_path, size_t max_len) {
    if (!out_path || max_len == 0) {
        return -1;
    }
    const std::string path_str = path.string();
    const size_t copy_len = path_str.length() < (max_len - 1) ? path_str.length() : (max_len - 1);
    std::memcpy(out_path, path_str.c_str(), copy_len);
    out_path[copy_len] = '\0';
    return 0;
}

int search_one(const char* dir, const char* filename, char* out_path, size_t max_len) {
    if (!dir || !filename || !out_path) {
        return -1;
    }
    auto result = Backend::Utils::FileFindExact::search_file(
        std::filesystem::path(dir),
        std::string(filename)
    );
    if (result) {
        return copy_path(result.value(), out_path, max_len);
    }
    if (result.error() == Backend::Utils::FileFindExact::SearchStatus::DirectoryError) {
        out_path[0] = '\0';
        return -1;
    }
    out_path[0] = '\0';
    return -2;
}

} // namespace

extern "C" {

int search_file(
    const char* shared_dir,
    const char* filename,
    const char* project_root,
    char* out_path,
    size_t max_len
) {
    if (shared_dir && shared_dir[0] != '\0') {
        const int r = search_one(shared_dir, filename, out_path, max_len);
        if (r == 0) {
            return 0;
        }
        if (r == -1 && (!project_root || project_root[0] == '\0')) {
            return -1;
        }
    }
    if (project_root && project_root[0] != '\0') {
        return search_one(project_root, filename, out_path, max_len);
    }
    if (out_path && max_len > 0) {
        out_path[0] = '\0';
    }
    return -2;
}

bool search_file_c(const char* search_dir, const char* filename, char* out_path, size_t max_len) {
    return search_one(search_dir, filename, out_path, max_len) == 0;
}

}
