@echo off
chcp 65001 >nul
"%~dp0md-reader.exe" stats %*
if %ERRORLEVEL% neq 0 echo. & pause
