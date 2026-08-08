cmake_minimum_required(VERSION 3.10)

project(FileNode)

# Tell CMake to look inside the 'cmake' folder for modules
list(APPEND CMAKE_MODULE_PATH "${CMAKE_CURRENT_SOURCE_DIR}/cmake")

# Pull in your strict compiler flags template
include(CompilerFlags)

#generate the comipile_commands.json
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

#for filesearch.dll
add_subdirectory(c)


# Define your application
add_executable(my_app main.cpp)

# Liquid Gold: This single line injects all C++23 rules, warnings, 
# and the GCC static analyzer into your executable!
target_link_libraries(my_app PRIVATE project_warnings)
