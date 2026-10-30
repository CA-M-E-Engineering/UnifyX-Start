package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/17isme/unifyx-start/internal/launcher"
	"github.com/17isme/unifyx-start/internal/platform"
	"github.com/joho/godotenv"
)

func main() {
	if err := run(); err != nil {
		log.Printf("启动器失败: %v", err)
		os.Exit(1)
	}
}

func run() error {
	urlFlag := flag.String("url", "", "Target URL to open (e.g. http://127.0.0.1:3000)")
	flag.Parse()
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("读取 .env: %w", err)
	}
	url := *urlFlag
	if url == "" {
		url = os.Getenv("LAUNCH_URL")
	}
	if url == "" {
		url = "https://www.google.com"
	}
	watchdog, err := envBool("ENABLE_WATCHDOG")
	if err != nil {
		return err
	}
	lockdown, err := envBool("DISABLE_OS_SHORTCUTS")
	if err != nil {
		return err
	}
	nativeFullscreen, err := envBool("NATIVE_FULLSCREEN_ONLY")
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	app, err := launcher.New(ctx, os.Getenv("BROWSER_PATH"), os.Getenv("USER_DATA_DIR"), nativeFullscreen)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return err
	}
	defer app.Close()
	if lockdown && ctx.Err() == nil {
		platform.LockdownOS()
	}
	err = supervise(ctx, func(ctx context.Context) error { return app.Run(ctx, url) }, watchdog)
	log.Println("启动器已停止。")
	return err
}

func envBool(name string) (bool, error) {
	value := os.Getenv(name)
	if value == "" {
		return false, nil
	}
	result, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s 必须为 true 或 false: %w", name, err)
	}
	return result, nil
}

// supervise 只监控浏览器主进程退出；页面健康检查不在此循环中。
func supervise(ctx context.Context, launch func(context.Context) error, watchdog bool) error {
	var delay time.Duration
	for ctx.Err() == nil {
		started := time.Now()
		err := launch(ctx)
		if ctx.Err() != nil {
			return nil
		}
		if !watchdog {
			return err
		}
		if err != nil {
			log.Printf("应用运行失败: %v", err)
		}
		delay = restartDelay(delay, time.Since(started))
		log.Printf("应用窗口已退出，%s 后重新启动...", delay)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
	}
	return nil
}

func restartDelay(previous, uptime time.Duration) time.Duration {
	if previous == 0 || uptime >= time.Minute {
		return 3 * time.Second
	}
	return min(previous*2, 30*time.Second)
}
