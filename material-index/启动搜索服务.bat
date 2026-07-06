@echo off
cd /d "%~dp0search-server"
echo Starting Aurora Search Service...
echo Open http://localhost:8927 in browser
echo.
echo Press Ctrl+C to stop
echo.
search-server.exe
pause
