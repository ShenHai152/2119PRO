@echo off
setlocal

set "EZ64=%~dp0EZFRD64.dll"
set "EZ32=%~dp0EZFRD.dll"
set "EZCLSID=HKCR\CLSID\{2D815FAD-11FF-0847-9310-F8608879882B}\InProcServer32"

fltmc >nul 2>&1
if errorlevel 1 (
    echo ERROR: This script must be run from an Administrator Command Prompt.
    echo Right-click Command Prompt, choose "Run as administrator", then run it again.
    pause
    exit /b 1
)

if not exist "%EZ64%" (
    echo ERROR: EZFRD64.dll is not in %~dp0
    pause
    exit /b 2
)

if not exist "%EZ32%" (
    echo ERROR: EZFRD.dll is not in %~dp0
    pause
    exit /b 2
)

echo Registering 64-bit EZFRD...
"%SystemRoot%\System32\regsvr32.exe" /s "%EZ64%"

echo Registering 32-bit EZFRD...
"%SystemRoot%\SysWOW64\regsvr32.exe" /s "%EZ32%"

echo Verifying 64-bit COM registration...
reg query "%EZCLSID%" /ve /reg:64 | "%SystemRoot%\System32\findstr.exe" /I /L /C:"%EZ64%" >nul
if errorlevel 1 goto failed64

echo Verifying 32-bit COM registration...
reg query "%EZCLSID%" /ve /reg:32 | "%SystemRoot%\System32\findstr.exe" /I /L /C:"%EZ32%" >nul
if errorlevel 1 goto failed32

echo.
echo EZFRD registration verified successfully.
pause
exit /b 0

:failed64
echo.
echo ERROR: 64-bit EZFRD registration does not point to:
echo   %EZ64%
reg query "%EZCLSID%" /ve /reg:64
pause
exit /b 3

:failed32
echo.
echo ERROR: 32-bit EZFRD registration does not point to:
echo   %EZ32%
reg query "%EZCLSID%" /ve /reg:32
pause
exit /b 3
