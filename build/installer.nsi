; WayerPC Windows installer (NSIS 3, user-level — no admin required).
;
; Built by scripts/build-installer.ps1, which passes:
;   /DAPP_VERSION=0.2.0        (from version.json)
;   (paths below are relative to build/, the script's directory)
;
; Layout produced on the target machine (user picks <chosen> on the
; Directory page — drive root like D:\ or any subfolder like D:\projects\):
;
;   <chosen>\WayerPC\
;   +-- bin\WayerPC.exe        (app; bin\ is added to the user PATH)
;   +-- bin\uninstall.exe
;   +-- data\shared\           (drop folder, served by /ask)
;   +-- data\received\         (phone uploads)
;   +-- data\Data\wayerpc.db  (SQLite catalog, created on first run)
;
; Plus: Start Menu shortcut (Windows search finds "WayerPC"), user-PATH
; entry (terminal: `where WayerPC`, `wayerpc --version`), first-run
; %LOCALAPPDATA%\WayerPC\config.json (never clobbered on reinstall).

!ifndef APP_VERSION
  !error "APP_VERSION is not defined. Build via scripts/build-installer.ps1"
!endif

!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "WinMessages.nsh"
!include "StrFunc.nsh"
${StrRep}

Name "WayerPC ${APP_VERSION}"
OutFile "..\out\WayerPC-${APP_VERSION}-setup.exe"
Unicode True
RequestExecutionLevel user
SetShellVarContext current
InstallDir "$PROGRAMFILES\WayerPC"
InstallDirRegKey HKCU "Software\WayerPC" "InstallDir"
VIProductVersion "${APP_VERSION}.0"
VIAddVersionKey "ProductName" "WayerPC"
VIAddVersionKey "ProductVersion" "${APP_VERSION}"
VIAddVersionKey "FileDescription" "WayerPC — phone to PC transfer"
VIAddVersionKey "LegalCopyright" "MIT"

; ---- pages ----
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "..\LICENSE"
!define MUI_PAGE_CUSTOMFUNCTION_LEAVE dirLeave
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_RUN "$INSTDIR\bin\WayerPC.exe"
!define MUI_FINISHPAGE_RUN_TEXT "Launch WayerPC"
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

; ---- default dir: D:\projects\WayerPC when D: exists ----
Function .onInit
  IfFileExists "D:\*.*" hasD
    StrCpy $INSTDIR "$PROGRAMFILES\WayerPC"
    Goto done
  hasD:
    StrCpy $INSTDIR "D:\projects\WayerPC"
  done:
FunctionEnd

; Guarantee the install ends in a WayerPC folder: if the user picked
; D:\projects\ (or D:\) we append WayerPC; if they already chose a
; WayerPC leaf (any case) we leave it alone.
Function dirLeave
  StrCpy $1 $INSTDIR -7 ; last 7 chars (whole string when shorter)
  ${If} $1 != "WayerPC"
  ${AndIf} $1 != "wayerpc"
  ${AndIf} $1 != "WAYERPC"
    StrCpy $INSTDIR "$INSTDIR\WayerPC"
  ${EndIf}
FunctionEnd

Section "WayerPC application (required)" SecApp
  SectionIn RO
  SetOutPath "$INSTDIR\bin"
  File "bin\WayerPC.exe"

  ; data tree (db file itself is created on first run)
  CreateDirectory "$INSTDIR\data\shared"
  CreateDirectory "$INSTDIR\data\received"
  CreateDirectory "$INSTDIR\data\Data"

  ; install marker with version
  FileOpen $0 "$INSTDIR\data\.installed" w
  FileWrite $0 "${APP_VERSION}"
  FileClose $0

  ; first-run config — only when none exists (reinstalls never clobber
  ; a relocated library)
  SetShellVarContext current
  ReadEnvStr $2 "LOCALAPPDATA"
  ${If} $2 == ""
    ReadEnvStr $2 "USERPROFILE"
    StrCpy $2 "$2\AppData\Local"
  ${EndIf}
  CreateDirectory "$2\WayerPC"
  IfFileExists "$2\WayerPC\config.json" hasConfig
    ; escape backslashes for JSON
    ${StrRep} $3 "$INSTDIR\data" "\" "\\"
    ${StrRep} $4 "$INSTDIR" "\" "\\"
    FileOpen $0 "$2\WayerPC\config.json" w
    FileWrite $0 '{"host": "0.0.0.0", "port": 5000, "data_root": "$3", "install_dir": "$4"}$\r$\n'
    FileClose $0
  hasConfig:

  ; registry (install location + Add/Remove Programs entry)
  WriteRegStr HKCU "Software\WayerPC" "InstallDir" "$INSTDIR"
  WriteRegStr HKCU "Software\WayerPC" "DataDir" "$INSTDIR\data"
  WriteRegStr HKCU "Software\WayerPC" "Version" "${APP_VERSION}"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\WayerPC" \
    "DisplayName" "WayerPC ${APP_VERSION}"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\WayerPC" \
    "DisplayVersion" "${APP_VERSION}"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\WayerPC" \
    "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\WayerPC" \
    "UninstallString" "$INSTDIR\bin\uninstall.exe"
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\WayerPC" \
    "NoModify" 1
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\WayerPC" \
    "NoRepair" 1

  ; Start Menu shortcut -> Windows search finds "WayerPC"
  CreateDirectory "$SMPROGRAMS\WayerPC"
  CreateShortcut "$SMPROGRAMS\WayerPC\WayerPC.lnk" "$INSTDIR\bin\WayerPC.exe" \
    "" "$INSTDIR\bin\WayerPC.exe" 0
  CreateShortcut "$SMPROGRAMS\WayerPC\Uninstall.lnk" "$INSTDIR\bin\uninstall.exe"

  ; user PATH so the terminal finds it (`where WayerPC`)
  EnVar::AddValue "PATH" "$INSTDIR\bin"
  Pop $0
  SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000

  WriteUninstaller "$INSTDIR\bin\uninstall.exe"
SectionEnd

Section "Desktop shortcut" SecDesktop
  CreateShortcut "$DESKTOP\WayerPC.lnk" "$INSTDIR\bin\WayerPC.exe" \
    "" "$INSTDIR\bin\WayerPC.exe" 0
SectionEnd

Section "Uninstall"
  Delete "$INSTDIR\bin\WayerPC.exe"
  Delete "$INSTDIR\bin\uninstall.exe"
  RMDir "$INSTDIR\bin"
  ; data/ is intentionally KEPT (the library outlives the app)
  Delete "$SMPROGRAMS\WayerPC\WayerPC.lnk"
  Delete "$SMPROGRAMS\WayerPC\Uninstall.lnk"
  RMDir "$SMPROGRAMS\WayerPC"
  Delete "$DESKTOP\WayerPC.lnk"
  EnVar::DeleteValue "PATH" "$INSTDIR\bin"
  Pop $0
  SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000
  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\WayerPC"
  DeleteRegKey HKCU "Software\WayerPC"
  RMDir "$INSTDIR"
SectionEnd
