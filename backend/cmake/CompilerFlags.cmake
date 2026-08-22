# ============================================================================
# CompilerFlags.cmake - Modern C++23 strict flags (GCC, Clang, MSVC)
# ============================================================================
cmake_minimum_required(VERSION 3.25)

if(NOT TARGET project_warnings)
    add_library(project_warnings INTERFACE)

    target_compile_features(project_warnings INTERFACE cxx_std_23)
    set(CMAKE_CXX_STANDARD 23)
    set(CMAKE_CXX_STANDARD_REQUIRED ON)
    set(CMAKE_CXX_EXTENSIONS OFF)

    if(CMAKE_CXX_COMPILER_ID STREQUAL "GNU")
        target_compile_options(project_warnings INTERFACE
            -Wall
            -Wextra
            -Wpedantic
            -Wshadow
            -Wnon-virtual-dtor
            -Wcast-align
            -Wunused
            -Woverloaded-virtual
            -Wconversion
            -Wsign-conversion
            -Wnull-dereference
            -Wdouble-promotion
            -Wformat=2
        )
        # Analyzer is useful but slow; keep it opt-in via -DWAYER_ANALYZER=ON
        if(WAYER_ANALYZER)
            target_compile_options(project_warnings INTERFACE -fanalyzer)
        endif()
    elseif(CMAKE_CXX_COMPILER_ID MATCHES "Clang")
        target_compile_options(project_warnings INTERFACE
            -Wall -Wextra -Wpedantic -Wshadow -Wconversion
        )
    elseif(MSVC)
        target_compile_options(project_warnings INTERFACE /W4 /permissive- /Zc:__cplusplus)
    endif()
endif()
