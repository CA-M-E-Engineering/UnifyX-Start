# UnifyX-start

[🇺🇸 English](README.md) | [🇨🇳 简体中文](README_zh.md)

UnifyX-start 是一个轻量级、跨平台的浏览器启动器 (Launcher)，专门用于将任何 Web 应用（如前端 Vue/React 项目）以纯净的全屏 (Kiosk) 模式运行，并提供企业级的高可用监控。

通过底层调用系统安装的 Chrome/Chromium，它实现了真正的无边框沉浸式体验，并包含防拦截、防快捷键干扰和自动崩溃恢复的能力。

## 主要特性 ✨

- **沉浸式 Kiosk 模式**：通过调用本地 Chrome 引擎，以独立 App (`--app`) 的形式无缝全屏展示网页，消除常规浏览器的书签栏、地址栏及安全警告气泡。
- **环境沙盒隔离**：自动分离 Kiosk 数据与日常用户配置，拥有独立的运行时目录（如 Cache、Cookies、扩展配置），彼此绝不干扰。
- **系统级防干扰机制**：
  - 动态释放并加载专属防拦截 Chrome 扩展：阻止用户右键、长按、滑动返回、下拉刷新、长按 Esc 以及多点触控缩放。
  - 支持可选禁用操作系统底层快捷键（如切换桌面、任务视图等系统组合键）。
- **进程看门狗 (Watchdog)**：自动监听由于意外、内存崩溃导致的应用退出，可在退出数秒后自动重新拉起应用。
- **自动处理残留锁定**：智能清理非正常退出导致的 Chrome LevelDB 锁文件或会话提示（"Chrome didn't shut down correctly"），避免弹出恢复气泡。

## 依赖要求 🛠️

- **Golang**: `1.22+` (用于编译本项目)
- **Chrome / Chromium**: 系统环境中需预装 `google-chrome` 或 `chromium-browser`。程序会自动寻找常见安装路径。

## 运行与编译 🚀

### 1. 安装依赖及配置环境

```bash
# 获取依赖
go mod tidy

# 复制配置文件模板
cp .env.example .env
```

### 2. 本地直接运行

```bash
# 默认读取 .env 配置启动
go run main.go

# 也可通过命令行参数强行指定目标 URL (优先级最高)
go run main.go --url=http://127.0.0.1:3000
```

### 3. 构建可执行文件

构建单文件可执行产物，方便在不同平台部署运行：

```bash
# MacOS / Linux 平台构建
go build -o unifyx_launcher main.go

# Windows 平台构建
GOOS=windows GOARCH=amd64 go build -o unifyx_launcher.exe main.go
```

## 环境变量配置 ⚙️

参考 `.env.example` 文件。以下是所有支持的 `.env` 环境变量说明：

| 变量名 | 默认值 | 描述 |
| ------ | ------ | ---- |
| `LAUNCH_URL` | `https://www.google.com` | 应用要打开和渲染的目标网址 |
| `ENABLE_WATCHDOG` | `false` | 设置为 `true` 时，若浏览器进程被用户关闭、或意外崩溃，进程将被看守并于 3 秒后自动重新启动 |
| `DISABLE_OS_SHORTCUTS` | `false` | 设置为 `true` 时，尝试接管并禁用操作系统（如 GNOME）底层的系统切换快捷键，防止用户强行切出应用 |
| `USER_DATA_DIR` | `./app_profile` | 指定 Chrome 运行时的沙盒环境缓存目录路径。推荐直接使用相对路径，用于完全独立缓存 |

## 技术细节说明

- 本项目由 `os/exec` 完全掌管控制系统 Chrome 进程的启停。
- **防干扰机制（注入式扩展）**：在 Kiosk 窗口每次物理启动时，Go 程序会将内嵌在二进制文件中的 `content.js` 和 `manifest.json` 无感导出到系统临时目录，并经由 Chrome 的 `--load-extension` 参数注入，进程关闭时自动清理临时产物。
- 任何原因导致应用异常退出的死锁缓存（LevelDB 死锁状态）都会在下一次应用生命周期的拉起（启动前）阶段被自动重置与清理，从而彻底杜绝“恢复页面”等安全还原弹窗。
