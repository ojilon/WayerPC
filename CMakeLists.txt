cmake_minimum_required(VERSION 3.25)
project(WayerPCNative LANGUAGES C CXX)

list(APPEND CMAKE_MODULE_PATH "${CMAKE_CURRENT_SOURCE_DIR}/backend/cmake")
include(CompilerFlags)

# ---------------------------------------------------------------------------
# Shared output: all native libs land in build/native/
# ---------------------------------------------------------------------------
set(CMAKE_RUNTIME_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/native")
set(CMAKE_LIBRARY_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/native")
set(CMAKE_ARCHIVE_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/native")

function(wayer_shared_lib target)
    set_target_properties(${target} PROPERTIES
        PREFIX "lib"
        C_STANDARD 11
        CXX_STANDARD 23
        CXX_STANDARD_REQUIRED ON
        CXX_EXTENSIONS OFF
        WINDOWS_EXPORT_ALL_SYMBOLS ON
    )
    if(WIN32)
        target_compile_definitions(${target} PRIVATE _WIN32_WINNT=0x0A00 UNICODE _UNICODE)
    endif()
    target_link_libraries(${target} PRIVATE project_warnings)
endfunction()

# ---------------------------------------------------------------------------
# libfilesearch — exact recursive filename search
# ---------------------------------------------------------------------------
add_library(filesearch SHARED
    backend/filesearch/find_exact.cpp
)
target_include_directories(filesearch PUBLIC backend/filesearch)
wayer_shared_lib(filesearch)

# ---------------------------------------------------------------------------
# libusersearch — user-storage search (skips system + app data)
# ---------------------------------------------------------------------------
add_library(usersearch SHARED
    backend/usersearch/find_user.cpp
)
target_include_directories(usersearch PUBLIC backend/usersearch)
wayer_shared_lib(usersearch)
if(WIN32)
    target_link_libraries(usersearch PRIVATE shell32 ole32 uuid)
endif()

# ---------------------------------------------------------------------------
# libimportcatalog — optional native SQLite catalog
# Drop sqlite3.c + sqlite3.h into backend/third_party/sqlite/ then reconfigure.
# ---------------------------------------------------------------------------
set(SQLITE_DIR "${CMAKE_CURRENT_SOURCE_DIR}/backend/third_party/sqlite")
if(EXISTS "${SQLITE_DIR}/sqlite3.c" AND EXISTS "${SQLITE_DIR}/sqlite3.h")
    message(STATUS "SQLite amalgamation found — building libimportcatalog")
    add_library(importcatalog SHARED
        backend/native/catalog_sqlite.cpp
        "${SQLITE_DIR}/sqlite3.c"
    )
    target_include_directories(importcatalog PUBLIC
        backend/native
        "${SQLITE_DIR}"
    )
    target_compile_definitions(importcatalog PRIVATE
        SQLITE_THREADSAFE=1
        SQLITE_OMIT_LOAD_EXTENSION
        WAYER_HAS_SQLITE=1
    )
    # sqlite3.c is C; don't apply C++ warning-as-error to it
    set_source_files_properties("${SQLITE_DIR}/sqlite3.c" PROPERTIES LANGUAGE C)
    wayer_shared_lib(importcatalog)
else()
    message(STATUS "SQLite amalgamation not found at ${SQLITE_DIR}")
    message(STATUS "  Python sqlite3 still powers the import catalog.")
    message(STATUS "  To also build libimportcatalog, see backend/third_party/sqlite/README.md")
endif()

# Stub so `cmake --build` has an executable target if needed
add_executable(wayer_native_stub backend/main.cpp)
target_link_libraries(wayer_native_stub PRIVATE project_warnings)
