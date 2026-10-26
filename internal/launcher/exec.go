package launcher

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

//go:embed extension/*
var extFS embed.FS

// App 的浏览器路径和扩展目录在整个看门狗生命周期中复用。
type App struct {
	browser, profile, extension string
}

func New(ctx context.Context, browserPath, userDataDir string) (*App, error) {
	browser, err := findChromeBinary(ctx, browserPath)
	if err != nil {
		return nil, err
	}
	if userDataDir == "" {
		userDataDir = "./app_profile"
	}
	profile, err := filepath.Abs(userDataDir)
	if err != nil {
		return nil, fmt.Errorf("解析用户数据目录: %w", err)
	}
	if err := os.MkdirAll(profile, 0700); err != nil {
		return nil, fmt.Errorf("创建用户数据目录: %w", err)
	}
	extension, err := setupTempExtension()
	if err != nil {
		return nil, fmt.Errorf("准备防护扩展: %w", err)
	}
	log.Printf("渲染引擎: %s", browser)
	return &App{browser: browser, profile: profile, extension: extension}, nil
}

func (app *App) Close() {
	if err := os.RemoveAll(app.extension); err != nil {
		log.Printf("清理临时扩展失败: %v", err)
	}
}

func setupTempExtension() (dir string, err error) {
	dir, err = os.MkdirTemp("", "unifyx_ext_*")
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			os.RemoveAll(dir)
		}
	}()
	entries, err := extFS.ReadDir("extension")
	if err != nil {
		return dir, err
	}
	for _, entry := range entries {
		data, readErr := extFS.ReadFile("extension/" + entry.Name())
		if readErr != nil {
			return dir, readErr
		}
		if err = os.WriteFile(filepath.Join(dir, entry.Name()), data, 0600); err != nil {
			return dir, err
		}
	}
	return dir, nil
}

func buildChromeArgs(url, userDataDir, extensionDir string) []string {
	return []string{
		"--kiosk",
		"--app=" + url,
		"--start-fullscreen",
		"--noerrdialogs",
		"--disable-infobars",
		"--disable-notifications",
		"--disable-session-crashed-bubble",
		"--disable-hang-monitor",
		"--test-type",
		"--disable-features=Translate,ExclusiveAccessBubble,ExclusiveAccessBubbleV2,PointerLockBubble,KeyboardLockPrompt",
		"--use-fake-ui-for-media-stream",
		"--autoplay-policy=no-user-gesture-required",
		"--disable-background-networking",
		"--disable-extensions-except=" + extensionDir,
		"--load-extension=" + extensionDir,
		"--disable-popup-blocking",
		"--disable-prompt-on-repost",
		"--disable-sync",
		"--no-first-run",
		// 避免 Windows Edge 通过兼容层重新拉起自身，导致看门狗误判主进程退出。
		"--edge-skip-compat-layer-relaunch",
		"--no-default-browser-check",
		"--pull-to-refresh-style=0",
		"--overscroll-history-navigation=0",
		"--disk-cache-size=52428800",
		"--user-data-dir=" + userDataDir,
	}
}

// cleanupStaleLocks 按一体机单实例部署约定，保留启动前清理机制。
// Singleton 文件是进程单实例锁，并不是 LevelDB 锁。
func cleanupStaleLocks(userDataDir string) error {
	for _, name := range []string{"SingletonLock", "SingletonSocket", "SingletonCookie"} {
		if err := os.Remove(filepath.Join(userDataDir, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("清理 %s: %w", name, err)
		}
	}
	path := filepath.Join(userDataDir, "Default", "Preferences")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var prefs map[string]json.RawMessage
	if err := json.Unmarshal(data, &prefs); err != nil {
		return fmt.Errorf("读取浏览器 Preferences: %w", err)
	}
	var profile map[string]json.RawMessage
	if len(prefs["profile"]) == 0 {
		return nil
	}
	if err := json.Unmarshal(prefs["profile"], &profile); err != nil {
		return err
	}
	var exitType string
	if err := json.Unmarshal(profile["exit_type"], &exitType); err != nil || exitType != "Crashed" {
		return nil
	}
	profile["exit_type"] = json.RawMessage(`"Normal"`)
	profile["exited_cleanly"] = json.RawMessage(`true`)
	prefs["profile"], err = json.Marshal(profile)
	if err != nil {
		return err
	}
	data, err = json.Marshal(prefs)
	if err != nil {
		return err
	}
	// 在同一目录写完再替换，避免写到一半损坏原 Preferences。
	tmp, err := os.CreateTemp(filepath.Dir(path), ".unifyx-prefs-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	_, writeErr := tmp.Write(data)
	closeErr := tmp.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func (app *App) Run(ctx context.Context, url string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := cleanupStaleLocks(app.profile); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, app.browser, buildChromeArgs(url, app.profile, app.extension)...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	cmd.WaitDelay = 5 * time.Second
	configureProcess(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动浏览器: %w", err)
	}
	log.Printf("应用窗口已启动 (PID: %d): %s", cmd.Process.Pid, url)
	err := cmd.Wait()
	cleanupProcess(cmd)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		return fmt.Errorf("浏览器退出: %w", err)
	}
	return nil
}
