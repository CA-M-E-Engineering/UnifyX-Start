package launcher

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func browserVersion(ctx context.Context, path string) (string, error) {
	// Windows 的 chrome.exe --version 可能启动 GUI，改为只读取文件元数据。
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command",
		`$v = (Get-Item -LiteralPath $env:UNIFYX_BROWSER_BINARY).VersionInfo; [Console]::Write($v.ProductName + ' ' + $v.ProductVersion)`)
	cmd.Env = append(os.Environ(), "UNIFYX_BROWSER_BINARY="+path)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func configureProcess(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		kill := exec.CommandContext(ctx, "taskkill.exe", "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F")
		kill.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if err := kill.Run(); err != nil {
			return cmd.Process.Kill()
		}
		return nil
	}
}

func cleanupProcess(cmd *exec.Cmd) {}
