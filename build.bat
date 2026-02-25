@echo off
setlocal

:: Check for Visual Studio
where cl >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo Visual Studio compiler not found. Please run this from a Visual Studio Developer Command Prompt.
    exit /b 1
)

:: Set CGO flags for MSVC
set CGO_ENABLED=1
set CC=cl
set CXX=cl

:: Check for Vulkan SDK
if not defined VULKAN_SDK (
    echo Vulkan SDK not found. Please install the Vulkan SDK from https://vulkan.lunarg.com/
    echo Make sure to set VULKAN_SDK environment variable.
    exit /b 1
)

echo Building with Vulkan SDK: %VULKAN_SDK%

cd /d "%~dp0"
go build -v ./...

if %ERRORLEVEL% equ 0 (
    echo Build successful!
) else (
    echo Build failed!
)

endlocal
