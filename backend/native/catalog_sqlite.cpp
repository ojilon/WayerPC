#include "catalog_sqlite.hpp"

#include "sqlite3.h"

#include <cstring>
#include <filesystem>
#include <mutex>
#include <string>

namespace {

sqlite3* g_db = nullptr;
std::mutex g_mu;

const char* kSchema =
    "CREATE TABLE IF NOT EXISTS imported_files ("
    "  id INTEGER PRIMARY KEY AUTOINCREMENT,"
    "  path TEXT NOT NULL UNIQUE,"
    "  name TEXT NOT NULL,"
    "  size_bytes INTEGER,"
    "  imported_at TEXT NOT NULL DEFAULT (datetime('now'))"
    ");";

int write_rows(sqlite3_stmt* stmt, char* out_buf, size_t max_len) {
    if (!out_buf || max_len < 2) {
        return -1;
    }
    out_buf[0] = '\0';
    std::string assembled;
    int count = 0;
    while (sqlite3_step(stmt) == SQLITE_ROW) {
        const char* name = reinterpret_cast<const char*>(sqlite3_column_text(stmt, 0));
        const char* path = reinterpret_cast<const char*>(sqlite3_column_text(stmt, 1));
        if (!name || !path) {
            continue;
        }
        const std::string line = std::string(name) + "\t" + path + "\n";
        if (assembled.size() + line.size() + 1 >= max_len) {
            break;
        }
        assembled += line;
        ++count;
    }
    std::memcpy(out_buf, assembled.c_str(), assembled.size() + 1);
    return count;
}

} // namespace

extern "C" {

int catalog_open(const char* db_path) {
    std::lock_guard<std::mutex> lock(g_mu);
    if (!db_path) {
        return -1;
    }
    if (g_db) {
        sqlite3_close(g_db);
        g_db = nullptr;
    }
    if (sqlite3_open(db_path, &g_db) != SQLITE_OK) {
        g_db = nullptr;
        return -1;
    }
    char* err = nullptr;
    if (sqlite3_exec(g_db, kSchema, nullptr, nullptr, &err) != SQLITE_OK) {
        sqlite3_free(err);
        sqlite3_close(g_db);
        g_db = nullptr;
        return -1;
    }
    return 0;
}

int catalog_add(const char* path) {
    std::lock_guard<std::mutex> lock(g_mu);
    if (!g_db || !path) {
        return -1;
    }
    namespace fs = std::filesystem;
    const fs::path p(path);
    std::error_code ec;
    if (!fs::is_regular_file(p, ec)) {
        return -2;
    }
    const auto size = fs::file_size(p, ec);
    sqlite3_stmt* stmt = nullptr;
    const char* sql =
        "INSERT OR REPLACE INTO imported_files(path, name, size_bytes) VALUES(?,?,?);";
    if (sqlite3_prepare_v2(g_db, sql, -1, &stmt, nullptr) != SQLITE_OK) {
        return -1;
    }
    const std::string path_s = p.string();
    const std::string name_s = p.filename().string();
    sqlite3_bind_text(stmt, 1, path_s.c_str(), -1, SQLITE_TRANSIENT);
    sqlite3_bind_text(stmt, 2, name_s.c_str(), -1, SQLITE_TRANSIENT);
    sqlite3_bind_int64(stmt, 3, static_cast<sqlite3_int64>(ec ? 0 : size));
    const int rc = sqlite3_step(stmt);
    sqlite3_finalize(stmt);
    return rc == SQLITE_DONE ? 0 : -1;
}

int catalog_remove(const char* path) {
    std::lock_guard<std::mutex> lock(g_mu);
    if (!g_db || !path) {
        return -1;
    }
    sqlite3_stmt* stmt = nullptr;
    if (sqlite3_prepare_v2(g_db, "DELETE FROM imported_files WHERE path = ?;", -1, &stmt, nullptr) != SQLITE_OK) {
        return -1;
    }
    sqlite3_bind_text(stmt, 1, path, -1, SQLITE_TRANSIENT);
    const int rc = sqlite3_step(stmt);
    sqlite3_finalize(stmt);
    return rc == SQLITE_DONE ? 0 : -1;
}

int catalog_list(char* out_buf, size_t max_len) {
    std::lock_guard<std::mutex> lock(g_mu);
    if (!g_db) {
        return -1;
    }
    sqlite3_stmt* stmt = nullptr;
    if (sqlite3_prepare_v2(g_db, "SELECT name, path FROM imported_files ORDER BY name;", -1, &stmt, nullptr) != SQLITE_OK) {
        return -1;
    }
    const int n = write_rows(stmt, out_buf, max_len);
    sqlite3_finalize(stmt);
    return n;
}

int catalog_find(const char* query, int min_percent, char* out_buf, size_t max_len) {
    std::lock_guard<std::mutex> lock(g_mu);
    if (!g_db || !query) {
        return -1;
    }
    (void)min_percent; // Python layer applies the 20% filter with richer scoring
    sqlite3_stmt* stmt = nullptr;
    const char* sql =
        "SELECT name, path FROM imported_files "
        "WHERE name LIKE '%' || ? || '%' COLLATE NOCASE "
        "ORDER BY name LIMIT 200;";
    if (sqlite3_prepare_v2(g_db, sql, -1, &stmt, nullptr) != SQLITE_OK) {
        return -1;
    }
    sqlite3_bind_text(stmt, 1, query, -1, SQLITE_TRANSIENT);
    const int n = write_rows(stmt, out_buf, max_len);
    sqlite3_finalize(stmt);
    return n;
}

void catalog_close(void) {
    std::lock_guard<std::mutex> lock(g_mu);
    if (g_db) {
        sqlite3_close(g_db);
        g_db = nullptr;
    }
}

}
