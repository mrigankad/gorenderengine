@echo off
setlocal

echo Building Render Engine (OpenGL backend)...

set CGO_ENABLED=1

cd /d "%~dp0"

:: -s -w strips debug info; required on Windows with GCC to avoid HVCI issues.
go build -ldflags="-s -w" -v -o triangle_app.exe ./examples/basic_triangle/

if %ERRORLEVEL% equ 0 (
    echo Build successful: triangle_app.exe
) else (
    echo Build failed!
)

endlocal
