#include "find_user.hpp"

#include <algorithm>
#include <cctype>
#include <cstdlib>
#include <cstring>
#include <filesystem>
#include <string>
#include <string_view>
#include <vector>

#ifdef _WIN32
#ifndef NOMINMAX
#define NOMINMAX
#endif
#include <windows.h>
#include <shlobj.h>
#endif

namespace fs = std::filesystem;

namespace {

std::string to_lower(std::string s) {
    std::transform(s.begin(), s.end(), s.begin(), [](unsigned char c) {
        return static_cast<char>(std::tolower(c));
    });
    return s;
}

bool contains_ci(std::string_view hay, std::string_view needle_lower) {
    const std::string h = to_lower(std::string(hay));
    return h.find(needle_lower) != std::string::npos;
}

bool is_skipped_path(const fs::path& p) {
    const std::string s = to_lower(p.generic_string());
    static const char* blocked[] = {
        "/windows/", "\\windows\\",
        "/program files", "\\program files",
        "/programdata", "\\programdata",
        "/appdata/", "\\appdata\\",
        "$recycle.bin",
        "system volume information",
        "/windowsapps/", "\\windowsapps\\",
        "/winsxs/", "\\winsxs\\",
        "/recovery/", "\\recovery\\",
        "/proc/", "/sys/", "/dev/", "/run/",
        "/usr/lib", "/usr/share",
        "/.git/",
    };
    for (const char* b : blocked) {
        if (s.find(b) != std::string::npos) {
            return true;
        }
    }
    // Skip the WayerPC app folder itself
    if (s.find("/wayerpc/") != std::string::npos || s.find("\\wayerpc\\") != std::string::npos) {
        return true;
    }
    return false;
}

int levenshtein(const std::string& a, const std::string& b) {
    const size_t n = a.size();
    const size_t m = b.size();
    if (n == 0) {
        return static_cast<int>(m);
    }
    if (m == 0) {
        return static_cast<int>(n);
    }
    std::vector<int> prev(m + 1);
    std::vector<int> cur(m + 1);
    for (size_t j = 0; j <= m; ++j) {
        prev[j] = static_cast<int>(j);
    }
    for (size_t i = 1; i <= n; ++i) {
        cur[0] = static_cast<int>(i);
        for (size_t j = 1; j <= m; ++j) {
            const int cost = a[i - 1] == b[j - 1] ? 0 : 1;
            cur[j] = std::min({prev[j] + 1, cur[j - 1] + 1, prev[j - 1] + cost});
        }
        prev.swap(cur);
    }
    return prev[m];
}

void add_root(std::vector<fs::path>& roots, const fs::path& p) {
    std::error_code ec;
    if (p.empty()) {
        return;
    }
    if (fs::is_directory(p, ec)) {
        roots.push_back(p);
    }
}

std::vector<fs::path> collect_roots() {
    std::vector<fs::path> roots;
#ifdef _WIN32
    wchar_t buf[MAX_PATH];
    const KNOWNFOLDERID folders[] = {
        FOLDERID_Documents, FOLDERID_Downloads, FOLDERID_Desktop,
        FOLDERID_Pictures, FOLDERID_Videos, FOLDERID_Music, FOLDERID_Public
    };
    for (const auto& id : folders) {
        PWSTR wpath = nullptr;
        if (SUCCEEDED(SHGetKnownFolderPath(id, 0, nullptr, &wpath)) && wpath) {
            add_root(roots, fs::path(wpath));
            CoTaskMemFree(wpath);
        }
    }
    // Extra drives (skip the Windows system drive's root — we already have user folders)
    wchar_t win_dir[MAX_PATH];
    GetWindowsDirectoryW(win_dir, MAX_PATH);
    const wchar_t sys_letter = win_dir[0];
    const DWORD mask = GetLogicalDrives();
    for (int i = 0; i < 26; ++i) {
        if ((mask & (1u << i)) == 0) {
            continue;
        }
        const wchar_t letter = static_cast<wchar_t>(L'A' + i);
        if (letter == sys_letter) {
            continue;
        }
        wchar_t root[] = {letter, L':', L'\\', 0};
        const UINT type = GetDriveTypeW(root);
        if (type == DRIVE_FIXED || type == DRIVE_REMOVABLE || type == DRIVE_RAMDISK) {
            add_root(roots, fs::path(root));
        }
    }
    (void)buf;
#else
    const char* home = std::getenv("HOME");
    if (home) {
        const fs::path h(home);
        add_root(roots, h / "Documents");
        add_root(roots, h / "Downloads");
        add_root(roots, h / "Desktop");
        add_root(roots, h / "Pictures");
        add_root(roots, h / "Videos");
        add_root(roots, h / "Music");
        add_root(roots, h);
    }
    add_root(roots, fs::path("/mnt"));
    add_root(roots, fs::path("/media"));
#endif
    return roots;
}

void append_match(std::string& out, const fs::path& p, size_t max_len, int& count, int cap) {
    if (count >= cap) {
        return;
    }
    const std::string line = p.filename().string() + "\t" + p.string() + "\n";
    if (out.size() + line.size() + 1 >= max_len) {
        return;
    }
    out += line;
    ++count;
}

} // namespace

extern "C" {

int name_similarity_percent(const char* a, const char* b) {
    if (!a || !b) {
        return 0;
    }
    const std::string la = to_lower(a);
    const std::string lb = to_lower(b);
    if (la.empty() && lb.empty()) {
        return 100;
    }
    const int dist = levenshtein(la, lb);
    const int longest = static_cast<int>(std::max(la.size(), lb.size()));
    if (longest == 0) {
        return 100;
    }
    const int pct = 100 - (dist * 100) / longest;
    return pct < 0 ? 0 : pct;
}

int search_user_storage(const char* query, char* out_buf, size_t max_len) {
    if (!query || !out_buf || max_len < 2) {
        return -1;
    }
    out_buf[0] = '\0';
    const std::string q = to_lower(query);
    if (q.empty()) {
        return 0;
    }

    constexpr int kCap = 250;
    int count = 0;
    std::string assembled;
    assembled.reserve(std::min(max_len, static_cast<size_t>(1 << 16)));

    const auto roots = collect_roots();
    std::error_code ec;
    const auto opts = fs::directory_options::skip_permission_denied;

    for (const auto& root : roots) {
        if (count >= kCap) {
            break;
        }
        for (auto it = fs::recursive_directory_iterator(root, opts, ec);
             it != fs::recursive_directory_iterator(); ++it) {
            if (ec) {
                ec.clear();
                continue;
            }
            const fs::path p = it->path();
            if (is_skipped_path(p)) {
                if (it->is_directory(ec)) {
                    it.disable_recursion_pending();
                }
                continue;
            }
            const std::string name = p.filename().string();
            if (!contains_ci(name, q)) {
                continue;
            }
            if (it->is_regular_file(ec)) {
                append_match(assembled, p, max_len, count, kCap);
            } else if (it->is_directory(ec)) {
                // Folder name matched: include files directly under it (one level)
                std::error_code ec2;
                for (const auto& child : fs::directory_iterator(p, opts, ec2)) {
                    if (child.is_regular_file(ec2)) {
                        append_match(assembled, child.path(), max_len, count, kCap);
                    }
                    if (count >= kCap) {
                        break;
                    }
                }
            }
        }
    }

    if (assembled.size() >= max_len) {
        assembled.resize(max_len - 1);
    }
    std::memcpy(out_buf, assembled.c_str(), assembled.size() + 1);
    return count;
}

}
