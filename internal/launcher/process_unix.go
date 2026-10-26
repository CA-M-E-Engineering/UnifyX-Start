//go:build linux || darwin

package launcher

import (
	"context"
	"os/exec"
	"strings"
	"syscall"
)

func browserVersion(ctx context.Context, path string) (string, error) {
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	return strings.TrimSpace(string(out)), err
}

func configureProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM) }
}

func cleanupProcess(cmd *exec.Cmd) {
	// 主进程退出或超时后，回收本次启动的进程组中的剩余子进程。
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
