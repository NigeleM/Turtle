@echo off
rem Runs a double-clicked .turtle file in its own folder (see turtle.iss).
title Turtle
cd /d "%~dp1"
"%~dp0turtle.exe" %*
exit /b %errorlevel%
