cmake_minimum_required(VERSION 3.10)

project(FileNode)

#generate the comipile_commands.json
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

#for filesearch.dll
add_subdirectory(c)
