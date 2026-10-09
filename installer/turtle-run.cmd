@echo off
rem Runs a Turtle script opened from Explorer (double-click, or right-click
rem "Run with Turtle"), and keeps the window open at the end so what it
rem showed can be read. Installed next to turtle.exe by turtle.iss.
title Turtle
rem The script's own folder, so files beside it are found.
cd /d "%~dp1"
"%~dp0turtle.exe" %*
set "code=%errorlevel%"
echo.
if "%code%"=="0" (
    echo Finished. Press any key to close.
) else (
    echo Finished with exit code %code%. Press any key to close.
)
pause >nul
exit /b %code%
