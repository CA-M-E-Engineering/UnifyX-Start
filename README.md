# UnifyX-start

[🇺🇸 English](README.md) | [🇨🇳 简体中文](README_zh.md)

UnifyX-start is a lightweight, cross-platform browser launcher specifically designed to run any Web application (like Vue/React frontend projects) in a pure, full-screen Kiosk mode, providing enterprise-grade high availability and monitoring.

By controlling the natively installed Chrome/Chromium engine under the hood, it delivers a truly borderless, immersive experience. It also features anti-interference mechanisms, OS shortcut blocking, and auto-recovery from crashes.

## Key Features ✨

- **Immersive Kiosk Mode**: Seamlessly displays web pages in full-screen as a standalone App (`--app`) by invoking the local Chrome engine, eliminating regular browser elements like the bookmarks bar, address bar, and security warning bubbles.
- **Sandboxed Environment**: Automatically isolates Kiosk data from daily user profiles, maintaining an independent runtime directory (e.g., Cache, Cookies, extension configurations) to completely avoid interference.
- **System-Level Anti-Interference**:
  - Dynamically injects a dedicated anti-interference Chrome extension: blocks right-clicks, long presses, swipe-to-go-back, pull-to-refresh, long-press Esc, and multi-touch zooming.
  - Supports optional disabling of OS-level shortcuts (e.g., switching desktops, task view, etc., specifically under Linux/GNOME environments).
- **Process Watchdog**: Automatically monitors app exits caused by crashes or unexpected closures. The watchdog can automatically restart the application a few seconds after an exit.
- **Stale Lock Cleanup**: Intelligently cleans up Chrome LevelDB lock files and session recovery prompts ("Chrome didn't shut down correctly") caused by abnormal exits, preventing browser recovery bubbles on the next launch.

## Prerequisites 🛠️

- **Golang**: `1.22+` (Required to compile this project)
- **Chrome / Chromium**: `google-chrome` or `chromium-browser` must be pre-installed on the host system. The application automatically locates common installation paths.

## Run and Build 🚀

### 1. Install Dependencies & Configure Environment

```bash
# Get dependencies
go mod tidy

# Copy the configuration template
cp .env.example .env
```

### 2. Run Locally

```bash
# Starts using the .env configuration by default
go run main.go

# Force a target URL via command line flags (highest priority)
go run main.go --url=http://127.0.0.1:3000
```

### 3. Build Executable

Build a single executable binary for cross-platform deployment:

```bash
# Build for MacOS / Linux
go build -o unifyx_launcher main.go

# Build for Windows
GOOS=windows GOARCH=amd64 go build -o unifyx_launcher.exe main.go
```

## Environment Variables ⚙️

Refer to the `.env.example` file. Below are the supported environment variables for `.env`:

| Variable Name | Default Value | Description |
| ------------- | ------------- | ----------- |
| `LAUNCH_URL` | `https://www.google.com` | The target URL to open and render in the application. |
| `ENABLE_WATCHDOG` | `false` | When set to `true`, the watchdog will automatically restart the browser process 3 seconds after it closes or crashes. |
| `DISABLE_OS_SHORTCUTS` | `false` | When set to `true`, attempts to hijack and disable OS-level (e.g., GNOME) window-switching shortcuts, preventing the user from switching out of the Kiosk app. |
| `USER_DATA_DIR` | `./app_profile` | Defines the sandbox directory for Chrome caching. A relative path is recommended for a completely independent cache. |

## Technical Details

- The project entirely manages the system's Chrome process lifecycle using Go's `os/exec`.
- **Anti-Interference Mechanism (Injected Extension)**: Upon every physical start of the Kiosk window, the Go program extracts the embedded `content.js` and `manifest.json` files into the system's temporary directory and injects them via Chrome's `--load-extension` flag. Temporary files are automatically cleaned up when the process closes.
- Deadlocks or cache states (LevelDB deadlock status) caused by any abnormal app exit will be automatically reset and cleaned during the initialization phase of the next lifecycle, completely eliminating "Restore Pages" safety popups.
