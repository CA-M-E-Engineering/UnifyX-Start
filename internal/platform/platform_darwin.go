package platform

import "log"

func LockdownOS() {
	log.Println("macOS 系统：无简单命令可以彻底屏蔽 Command+Tab 等热键。")
	log.Println("建议使用者在 Mac 上的“系统设置” - “辅助功能”中手动开启相关限制，或者使用屏幕时间进行锁定功能测试。")
}
