# C file search - uses backend/filesearch C++23 DLL at runtime
# The search_file function dynamically loads libfilesearch.dll
# via LoadLibraryA/GetProcAddress for the search_file_c function.
# This library is informational/standalone C implementation fallback.
add_library(filesearch_static STATIC
    file_search.c
)
set_target_properties(filesearch_static PROPERTIES
    OUTPUT_NAME "filesearch"
)
