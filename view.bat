@echo off
chcp 65001 >nul
title MD Reader - Просмотр Markdown

set "EXE=%~dp0md-reader.exe"

if not exist "%EXE%" (
    echo [ERROR] md-reader.exe не найден
    echo Папка: %~dp0
    pause
    exit /b 1
)

:: --help
if /I "%1"=="--help" goto :help
if /I "%1"=="-h" goto :help
if "%1"=="" goto :menu

:: Запуск с аргументом — открыть файл/папку
"%EXE%" open %*
if %ERRORLEVEL% neq 0 (
    echo.
    pause
)
exit /b

:menu
cls
echo ╔══════════════════════════════════════╗
echo ║        MD Reader — Меню             ║
echo ╚══════════════════════════════════════╝
echo.
echo  1. Просмотр файла
echo  2. Просмотр папки (рекурсивно)
echo  3. Показать оглавление
echo  4. Статистика по папке
echo  5. Версия
echo.
echo  0. Выход
echo.
echo  Использование: перетащи .md файл
echo  на этот батник для быстрого открытия.
echo.
set /p "menu=Выбери [0-5]: "

if "%menu%"=="1" (
    set /p "path=Путь к файлу: "
    "%EXE%" open "%path%"
)
if "%menu%"=="2" (
    set /p "path=Путь к папке: "
    "%EXE%" open "%path%" --recursive
)
if "%menu%"=="3" (
    set /p "path=Путь к файлу/папке: "
    "%EXE%" toc "%path%"
    echo.
    pause
)
if "%menu%"=="4" (
    set /p "path=Путь к папке: "
    "%EXE%" stats "%path%"
    echo.
    pause
)
if "%menu%"=="5" (
    "%EXE%" version
    echo.
    pause
)
exit /b

:help
echo.
echo  MD Reader — просмотр Markdown документов
echo.
echo  Использование:
echo    view.bat                       — меню
echo    view.bat файл.md              — открыть файл
echo    view.bat папка\               — открыть папку
echo    view.bat папка\ --recursive   — рекурсивно
echo.
echo  Быстрый запуск:
echo    Перетащи .md файл на view.bat
echo.
pause
