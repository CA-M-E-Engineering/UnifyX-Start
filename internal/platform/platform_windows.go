package platform

import "log"

func LockdownOS() {
	log.Println("Windows 系统：无法直接通过简单的命令行屏蔽所有的系统热键。")
	log.Println("建议：")
	log.Println("1. 编译时加上 -ldflags=\"-H windowsgui\" 隐藏后台黑色命令行窗口")
	log.Println("2. 将此程序在 Windows 的“受限访问 (Assigned Access)”中配置为 Kiosk 应用")
	log.Println("3. 或通过组策略 / 注册表禁用常用的功能键组合 (如 Alt+Tab 等)")
}
