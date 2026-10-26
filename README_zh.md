# UnifyX-start

[🇺🇸 English](README.md) | [🇨🇳 简体中文](README_zh.md)

面向一体机的轻量级 Go 启动器，通过基于 Chromium 的浏览器的 Kiosk/App 模式全屏展示 Web 应用，提供触摸防误操作和浏览器主进程退出后的自动重启。

## 运行要求

- 编译：Go **1.26+**。
- 浏览器：**Microsoft Edge、Chromium 或 Chrome for Testing 111+**，需为支持扩展的完整桌面浏览器。
- 支持 Windows、Linux、macOS；系统快捷键锁定仅在 Linux/GNOME 上实际执行。
- 每台设备只运行一个启动器实例，使用专属的 `USER_DATA_DIR`。

正式版 Google Chrome 不适用于本程序：Chrome 137 起移除了 `--load-extension`。本项目统一使用该参数加载内嵌扩展，启动前会检查浏览器品牌和版本，遇到不兼容浏览器明确报错，不会静默降级为无防护窗口。[Chrome 官方说明](https://groups.google.com/a/chromium.org/g/chromium-extensions/c/1-g8EFx2BBY)

## 运行与构建

```bash
go mod download
cp .env.example .env
go run .
# 命令行 URL 优先于环境变量
go run . --url=http://127.0.0.1:3000
go build -o unifyx_launcher .
```

Windows PowerShell：

```powershell
Copy-Item .env.example .env
go build -o unifyx_launcher.exe .
.\unifyx_launcher.exe
```

根据部署位置配置 `BROWSER_PATH`，例如：

```dotenv
# Linux
BROWSER_PATH=/usr/bin/chromium
# Windows Microsoft Edge 示例
# BROWSER_PATH='C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe'
# Windows 解压版 Chrome for Testing 示例
# BROWSER_PATH='C:\kiosk\chrome-win64\chrome.exe'
# macOS
# BROWSER_PATH='/Applications/Chromium.app/Contents/MacOS/Chromium'
```

留空时自动查找 PATH 和各平台常见安装位置。显式路径无效时直接报错，不自动切换其他浏览器。

## 配置

`.env` 从启动器的当前工作目录读取；系统环境变量优先于 `.env`，`--url` 优先级最高。相对路径也相对于当前工作目录，自启动服务应设置固定工作目录。

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `LAUNCH_URL` | `https://www.google.com` | 要展示的 Web 应用地址 |
| `BROWSER_PATH` | 自动查找 | Microsoft Edge / Chromium / Chrome for Testing 可执行文件路径 |
| `ENABLE_WATCHDOG` | `false` | 浏览器主进程退出后自动重启 |
| `DISABLE_OS_SHORTCUTS` | `false` | 应用现有 Linux/GNOME 一体机锁定设置 |
| `USER_DATA_DIR` | `./app_profile` | 专属浏览器用户数据目录，保留 Cookie 和缓存 |

## 行为说明

- **扩展**：启动器初始化时释放一次，浏览器重启时复用，启动器正常退出时清理；释放失败则启动失败。
- **全屏**：全屏 API 钩子在页面 `MAIN` 环境执行；触摸和键盘事件在扩展隔离环境处理；样式由扩展直接注入。刷新、导航和匹配的 iframe 会重新加载防护。
- **触摸**：屏蔽浏览器边缘导航、下拉刷新和双指缩放，隐藏光标；保留内部列表的正常滚动。浏览器扩展不能屏蔽操作系统自身的全部手势。
- **重启**：首次退出等待 3 秒；连续快速退出按 3、6、12、24、30 秒退避，最多 30 秒。连续运行满 1 分钟后恢复为 3 秒。关闭看门狗时，启动或运行失败会返回非零退出码。
- **监控范围**：看门狗监控浏览器主进程退出，不检测页面卡死、网络断开或单独的渲染进程崩溃。
- **停止**：Ctrl+C / SIGTERM 取消重启等待并停止浏览器。Linux/macOS 先发送 SIGTERM，最多等待 5 秒后回收进程组；Windows 结束本次浏览器进程树。
- **异常恢复**：按单实例部署约定，在每次启动浏览器前清理 Singleton 残留锁；通过 JSON 解析、临时文件替换修复崩溃标记，不修改其他配置。这不是 LevelDB 数据修复。
- **系统锁定**：保留原有 GNOME 快捷键、工作区、热角、收藏应用、锁屏和空闲设置的修改范围。设置会持久保存，退出启动器不会恢复。

## 验证

```bash
go test ./...
go vet ./...
```

可选的真实浏览器集成验证（Node.js 22+，无 npm 依赖，使用独立临时配置目录）：

```bash
BROWSER_PATH=/usr/bin/chromium node tests/browser.mjs
UNIFYX_TEST_BROWSER=/usr/bin/chromium go test ./internal/launcher
```

PowerShell 使用 `$env:BROWSER_PATH = '浏览器完整路径'` 设置变量后运行 `node tests/browser.mjs`。集成验证覆盖全屏钩子、iframe、快捷键、触摸限制、内部列表实际滚动和导航后重新注入；真实一体机上的系统手势与全屏显示仍需设备验收。