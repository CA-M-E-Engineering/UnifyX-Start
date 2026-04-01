package launcher

import (
	"embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

//go:embed extension/*
var extFS embed.FS

// setupTempExtension 将编译内嵌的扩展文件释放到系统临时目录
func setupTempExtension() (string, error) {
	tmpDir, err := os.MkdirTemp("", "unifyx_ext_*")
	if err != nil {
		return "", err
	}

	// 释放 manifest.json
	manifest, err := extFS.ReadFile("extension/manifest.json")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "manifest.json"), manifest, 0644); err != nil {
		return "", err
	}

	// 释放 content.js
	content, err := extFS.ReadFile("extension/content.js")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "content.js"), content, 0644); err != nil {
		return "", err
	}

	return tmpDir, nil
}

// findChromeBinary 在常见路径中查找系统安装的 Chrome/Chromium 可执行文件
func findChromeBinary() (string, error) {
	candidates := []string{
		"chromium-browser",
		"chromium",
		"google-chrome-stable",
		"google-chrome",
	}
	for _, name := range candidates {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	// 尝试常见的绝对路径
	absPaths := []string{
		"/usr/bin/chromium-browser",
		"/usr/bin/chromium",
		"/usr/bin/google-chrome-stable",
		"/usr/bin/google-chrome",
		"/opt/google/chrome/google-chrome",
	}
	for _, p := range absPaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("未找到 Chrome/Chromium 浏览器，请安装 chromium-browser 或 google-chrome")
}

// buildChromeArgs 组装 Chrome 启动参数
// 使用 --app 模式：Chrome 作为渲染引擎，呈现为独立应用窗口（无地址栏、无标签页）
func buildChromeArgs(url, userDataDir, extensionDir string) []string {
	args := []string{
		// === 应用模式：以独立应用窗口呈现，Chrome 仅作为渲染引擎 ===
		"--kiosk",        // 必须加上 kiosk 才能在 C++ 底层屏蔽“长按 Esc 退出”的安全提示
		"--app=" + url,
		"--start-fullscreen",

		// === 抑制弹窗/气泡/通知 ===
		"--noerrdialogs",
		"--disable-infobars",
		"--disable-notifications",
		"--disable-session-crashed-bubble",
		"--disable-hang-monitor",
		"--test-type",
		"--disable-features=Translate,ExclusiveAccessBubble,ExclusiveAccessBubbleV2,PointerLockBubble,KeyboardLockPrompt",

		// === 媒体与自动播放 ===
		"--use-fake-ui-for-media-stream",
		"--autoplay-policy=no-user-gesture-required",

		// === 禁用不必要的后台功能 ===
		"--disable-background-networking",
		"--disable-extensions-except=" + extensionDir,
		"--load-extension=" + extensionDir,
		"--disable-popup-blocking",
		"--disable-prompt-on-repost",
		"--disable-sync",
		"--no-first-run",
		"--no-default-browser-check",

		// === 下拉刷新与滑动手势 ===
		"--pull-to-refresh-style=0",
		"--overscroll-history-navigation=0",
		"--enable-features=OverscrollHistoryNavigationDisabled",

		// === 磁盘缓存限制 (50MB) ===
		"--disk-cache-size=52428800",

		// === 用户数据目录 ===
		"--user-data-dir=" + userDataDir,
	}
	return args
}

// cleanupStaleLocks 清理上次异常退出残留的锁文件，避免 Chrome 弹出恢复提示
func cleanupStaleLocks(userDataDir string) {
	lockFiles := []string{"SingletonLock", "SingletonSocket", "SingletonCookie"}
	for _, f := range lockFiles {
		fp := filepath.Join(userDataDir, f)
		if err := os.Remove(fp); err == nil {
			log.Printf("已清理残留锁文件: %s\n", fp)
		}
	}
	// 清理 "Chrome didn't shut down correctly" 标记
	defaultPrefs := filepath.Join(userDataDir, "Default", "Preferences")
	if data, err := os.ReadFile(defaultPrefs); err == nil {
		content := string(data)
		if strings.Contains(content, `"exit_type":"Crashed"`) {
			content = strings.Replace(content, `"exit_type":"Crashed"`, `"exit_type":"Normal"`, 1)
			os.WriteFile(defaultPrefs, []byte(content), 0644)
			log.Println("已修复 Chrome 上次崩溃退出标记")
		}
	}
}

// RunApp 启动 Chrome 应用窗口并监控进程
// Chrome 仅作为渲染引擎，呈现的是我们的应用程序窗口
// 此函数会在 Chrome 进程退出后返回（由调用方决定是否重启）
func RunApp(url string) {
	// 1. 解析用户数据目录
	userDataDir := os.Getenv("USER_DATA_DIR")
	if userDataDir == "" {
		userDataDir = "./app_profile"
	}
	absDataDir, _ := filepath.Abs(userDataDir)

	// 2. 释放内嵌的扩展文件到临时目录
	extDir, err := setupTempExtension()
	if err != nil {
		log.Printf("警告: 无法释放防拦截扩展 (%v)，防干扰脚本可能不会生效\n", err)
	} else {
		// 退出时自动清理临时写入的扩展文件
		defer os.RemoveAll(extDir)
		log.Printf("动态防护脚本已释放在: %s\n", extDir)
	}

	// 3. 查找 Chrome 可执行文件
	chromeBin, err := findChromeBinary()
	if err != nil {
		log.Fatalf("致命错误: %v\n", err)
	}
	log.Printf("渲染引擎: %s\n", chromeBin)

	// 4. 清理残留锁文件
	cleanupStaleLocks(absDataDir)

	// 5. 组装启动参数
	args := buildChromeArgs(url, absDataDir, extDir)
	log.Printf("启动应用窗口: %s --app=%s\n", chromeBin, url)

	// 6. 启动 Chrome 进程
	cmd := exec.Command(chromeBin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if err := cmd.Start(); err != nil {
		log.Printf("应用窗口启动失败: %v\n", err)
		return
	}

	log.Printf("应用窗口已启动 (PID: %d)，进入进程监控...\n", cmd.Process.Pid)

	// 7. 阻塞等待进程退出
	if err := cmd.Wait(); err != nil {
		log.Printf("应用进程异常退出: %v\n", err)
	} else {
		log.Println("应用进程正常退出")
	}

	time.Sleep(500 * time.Millisecond)
}
