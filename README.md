# UnifyX-start

[🇺🇸 English](README.md) | [🇨🇳 简体中文](README_zh.md)

A small Go launcher for dedicated all-in-one kiosks. It displays a web application in a Chromium-based browser's Kiosk/App mode, prevents accidental touch interactions, and restarts the browser when its main process exits.

## Requirements

- Build: **Go 1.26+**.
- Browser: **Microsoft Edge, Chromium or Chrome for Testing**, using a full desktop build with extension support. The default page-API fullscreen hook requires 111+; `NATIVE_FULLSCREEN_ONLY=true` uses native Kiosk fullscreen and supports Chromium 88+, including the hardware-adapted Chromium 91 tablet build.
- Windows, Linux and macOS are supported. OS shortcut lockdown is implemented only for Linux/GNOME.
- Run one launcher per device with a dedicated `USER_DATA_DIR`.

Regular Google Chrome is unsupported: Chrome 137 removed `--load-extension`. This launcher uses one extension-loading mechanism and checks browser branding/version before starting. An incompatible browser produces an error instead of silently opening an unprotected window. [Official Chrome announcement](https://groups.google.com/a/chromium.org/g/chromium-extensions/c/1-g8EFx2BBY)

## Run and build

```bash
go mod download
cp .env.example .env
go run .
go run . --url=http://127.0.0.1:3000
go build -o unifyx_launcher .
```

Windows PowerShell:

```powershell
Copy-Item .env.example .env
go build -o unifyx_launcher.exe .
.\unifyx_launcher.exe
```

Set `BROWSER_PATH` for your installation:

```dotenv
# Linux
BROWSER_PATH=/usr/bin/chromium
# Microsoft Edge on Windows
# BROWSER_PATH='C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe'
# Unpacked Chrome for Testing on Windows
# BROWSER_PATH='C:\kiosk\chrome-win64\chrome.exe'
# macOS
# BROWSER_PATH='/Applications/Chromium.app/Contents/MacOS/Chromium'
```

When unset, the launcher searches PATH and common installation locations for each platform. An invalid explicit path is an error; it never silently falls back to another browser.

## Configuration

The launcher reads `.env` from its current working directory. Existing environment variables take precedence over `.env`; `--url` has the highest priority. Relative paths also use the current working directory. Configure a fixed working directory for startup services.

| Variable | Default | Description |
| --- | --- | --- |
| `LAUNCH_URL` | `https://www.google.com` | Web application URL |
| `BROWSER_PATH` | Auto-detect | Microsoft Edge / Chromium / Chrome for Testing executable |
| `ENABLE_WATCHDOG` | `false` | Restart after the browser main process exits |
| `NATIVE_FULLSCREEN_ONLY` | `false` | Use native Kiosk fullscreen without modifying page Fullscreen APIs; input/CSS protection remains enabled |
| `DISABLE_OS_SHORTCUTS` | `false` | Apply the existing Linux/GNOME kiosk settings |
| `USER_DATA_DIR` | `./app_profile` | Dedicated browser profile retaining cookies/cache |

## Behavior

- **Extension lifecycle:** Extract once when the launcher initializes, reuse across browser restarts, and remove on normal launcher shutdown. Extraction errors stop startup.
- **Fullscreen:** Native Kiosk/App fullscreen flags are always used. By default, additional page API hooks run in the `MAIN` world; native-only mode omits those hooks for older embedded browsers. Input handlers and declarative CSS stay enabled in both modes and are reapplied on navigation, reload and matching frames.
- **Touch:** Block browser edge navigation, pull-to-refresh and pinch zoom, and hide the cursor. Scrolling inside page containers remains usable. Browser extensions cannot block every OS gesture.
- **Restart policy:** Wait 3 seconds after the first exit. Consecutive short runs back off through 3, 6, 12, 24 and 30 seconds, capped at 30. A run lasting at least one minute resets the delay to 3 seconds. With the watchdog disabled, launch/runtime failures return a nonzero exit status.
- **Monitoring scope:** Only browser main-process exits are monitored. Page hangs, network failures and individual renderer crashes are not detected.
- **Shutdown:** Ctrl+C / SIGTERM cancels restart waits and stops the browser. Linux/macOS send SIGTERM and allow up to five seconds before cleaning up the process group; Windows terminates the browser process tree.
- **Crash recovery:** Under the single-instance deployment assumption, remove stale Singleton files before each browser start. Repair crash markers using JSON parsing and a temporary-file replacement, preserving unrelated preferences. This does not repair LevelDB data.
- **OS lockdown:** Retains the existing GNOME changes to shortcuts, workspaces, hot corners, favorites, screen locking and idle settings. These changes persist after launcher exit.

## Validation

```bash
go test ./...
go vet ./...
```

Optional real-browser integration checks (Node.js 22+, no npm dependencies, isolated temporary profile):

```bash
BROWSER_PATH=/usr/bin/chromium node tests/browser.mjs
UNIFYX_TEST_BROWSER=/usr/bin/chromium go test ./internal/launcher
```

In PowerShell, set `$env:BROWSER_PATH = 'full executable path'` before running `node tests/browser.mjs`. The integration test covers early fullscreen hooks, frames, shortcuts, touch restrictions, real nested scrolling and reinjection after navigation. OS gestures and visible fullscreen behavior still require acceptance testing on the target kiosk.
