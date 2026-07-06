@echo off
cd /d "%~dp0search-server"
echo Building Aurora Search Service...
go build -o search-server.exe .
if %errorlevel% equ 0 (
    echo Build OK!
    echo Run with: 启动搜索服务.bat
) else (
    echo Build failed. Need Go 1.21+
)
pause
