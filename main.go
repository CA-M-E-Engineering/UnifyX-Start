package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/17isme/unifyx-start/internal/launcher"
	"github.com/17isme/unifyx-start/internal/platform"
	"github.com/joho/godotenv"
)

func main() {
	// 解析命令行参数
	urlFlag := flag.String("url", "", "Target URL to open (e.g. http://127.0.0.1:3000)")
	flag.Parse()

	err := godotenv.Load()
	if err != nil {
		log.Println("提示: 未找到.env文件或解析失败，将使用默认配置")
	}

	// 优先级：命令行参数 > 环境变量 (.env) > 默认值
	url := *urlFlag
	if url == "" {
		url = os.Getenv("LAUNCH_URL")
	}
	if url == "" {
		url = "https://www.google.com"
	}

	disableShortcuts := os.Getenv("DISABLE_OS_SHORTCUTS") == "true"
	if disableShortcuts {
		platform.LockdownOS()
	}

	enableWatchdog := os.Getenv("ENABLE_WATCHDOG") == "true"

	log.Printf("准备启动应用窗口, 目标地址: %s\n", url)
	launcher.RunApp(url)

	if enableWatchdog {
		for {
			log.Println("应用窗口已退出，3秒后自动重新拉起...")
			time.Sleep(3 * time.Second)
			log.Printf("重新启动应用窗口, 目标地址: %s\n", url)
			launcher.RunApp(url)
		}
	}

	log.Println("应用窗口已退出，启动器已停止。")
}
