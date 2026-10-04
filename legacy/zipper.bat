@echo off

powershell Compress-Archive ^
    -Path release\* ^
    -DestinationPath WayerPC-v0.1.0-alpha.zip ^
    -Force

pause