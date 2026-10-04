@echo off

echo.
echo ===== WayerPC Build =====
echo.

if exist release rmdir /s /q release
mkdir release

echo Compiling DLL...

zig build-lib c\file_search.c ^
    -dynamic ^
    -O ReleaseFast ^
    -femit-bin=release\file_search.dll

echo Building Python EXE...


.env\Scripts\python.exe -m PyInstaller ^
    --onefile ^
    --windowed ^
    --name WayerPC ^
    master.py

copy dist\WayerPC.exe release\
copy version.txt release\VERSION.txt

mkdir release\shared

echo.
echo Build Complete.
echo Output: release\
pause