@echo off
setlocal

set "EZ64=%~dp0EZFRD64.dll"
set "EZ32=%~dp0EZFRD.dll"
set "EZCLSID=HKCR\CLSID\{2D815FAD-11FF-0847-9310-F8608879882B}"

fltmc >nul 2>&1
if errorlevel 1 (
    echo ERROR: Run this script from an Administrator Command Prompt.
    pause
    exit /b 1
)

if not exist "%EZ64%" (
    echo ERROR: EZFRD64.dll is not beside this script.
    pause
    exit /b 2
)
if not exist "%EZ32%" (
    echo ERROR: EZFRD.dll is not beside this script.
    pause
    exit /b 2
)

echo Disabling the legacy 64-bit in-process FFB driver...
"%SystemRoot%\System32\regsvr32.exe" /u /s "%EZ64%"

echo Disabling the legacy 32-bit in-process FFB driver...
"%SystemRoot%\SysWOW64\regsvr32.exe" /u /s "%EZ32%"

reg query "%EZCLSID%" /reg:64 >nul 2>&1
if not errorlevel 1 goto failed64
reg query "%EZCLSID%" /reg:32 >nul 2>&1
if not errorlevel 1 goto failed32

echo.
echo EZFRD is isolated from games.
echo The PXN joystick remains available as a normal input device.
echo Use PXN2119FFBBridge.exe service mode for vibration.
echo Run register-ezfrd.cmd as Administrator to restore the original driver.
pause
exit /b 0

:failed64
echo ERROR: The 64-bit EZFRD COM class is still registered.
reg query "%EZCLSID%" /reg:64
pause
exit /b 3

:failed32
echo ERROR: The 32-bit EZFRD COM class is still registered.
reg query "%EZCLSID%" /reg:32
pause
exit /b 3
