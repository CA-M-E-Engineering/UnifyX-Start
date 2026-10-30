package launcher

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const browserRequirement = "请使用 Microsoft Edge、Chromium 或 Chrome for Testing；原生全屏模式需支持 Manifest V3 (88+)，网页全屏 API 钩子模式需 111+；正式版 Google Chrome 不支持本程序的扩展加载方式"

func findChromeBinary(ctx context.Context, configured string, nativeFullscreen bool) (string, error) {
	candidates := browserCandidates(runtime.GOOS)
	if configured != "" {
		candidates = []string{configured}
	}
	var failures []string
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		path, err := exec.LookPath(candidate)
		if err != nil {
			if configured != "" {
				return "", fmt.Errorf("BROWSER_PATH 无效: %w", err)
			}
			continue
		}
		path, err = filepath.Abs(path)
		if err != nil {
			return "", err
		}
		probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		version, err := browserVersion(probeCtx, path)
		cancel()
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if err == nil {
			err = validateBrowserVersion(version, nativeFullscreen)
		}
		if err == nil {
			return path, nil
		}
		failures = append(failures, fmt.Sprintf("%s: %v", path, err))
	}
	return "", fmt.Errorf("未找到兼容浏览器。%s。%s", browserRequirement, strings.Join(failures, "; "))
}

var compatibleVersion = regexp.MustCompile(`^(?:Microsoft Edge|Chromium|Google Chrome for Testing)\s+(\d+)\.`)

func validateBrowserVersion(version string, nativeFullscreen bool) error {
	match := compatibleVersion.FindStringSubmatch(strings.TrimSpace(version))
	if match == nil {
		return fmt.Errorf("不支持的浏览器: %q", strings.TrimSpace(version))
	}
	major, err := strconv.Atoi(match[1])
	minimum := 111
	if nativeFullscreen {
		minimum = 88
	}
	if err != nil || major < minimum {
		return fmt.Errorf("浏览器版本过旧: %q，当前模式需要 %d+", version, minimum)
	}
	return nil
}

func browserCandidates(goos string) []string {
	candidates := []string{"chromium", "chromium-browser", "chrome-for-testing"}
	switch goos {
	case "windows":
		for _, root := range []string{os.Getenv("LOCALAPPDATA"), os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)")} {
			if root != "" {
				candidates = append(candidates,
					filepath.Join(root, "Microsoft", "Edge", "Application", "msedge.exe"),
					filepath.Join(root, "Chromium", "Application", "chrome.exe"),
					filepath.Join(root, "Google", "Chrome for Testing", "Application", "chrome.exe"),
					filepath.Join(root, "Google", "Chrome", "Application", "chrome.exe"))
			}
		}
		return append(candidates, "msedge.exe", "chrome.exe")
	case "darwin":
		roots := []string{"/Applications"}
		if home, err := os.UserHomeDir(); err == nil {
			roots = append(roots, filepath.Join(home, "Applications"))
		}
		for _, root := range roots {
			for _, name := range []string{"Microsoft Edge", "Chromium", "Google Chrome for Testing", "Google Chrome"} {
				candidates = append(candidates, filepath.Join(root, name+".app", "Contents", "MacOS", name))
			}
		}
	default:
		candidates = append(candidates, "microsoft-edge", "microsoft-edge-stable", "/usr/bin/microsoft-edge", "/usr/bin/microsoft-edge-stable", "/usr/bin/chromium", "/usr/bin/chromium-browser", "/snap/bin/chromium")
	}
	return append(candidates, "google-chrome-stable", "google-chrome")
}
