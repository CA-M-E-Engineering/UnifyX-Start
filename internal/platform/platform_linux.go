package platform

import (
	"log"
	"os/exec"
)

func LockdownOS() {
	log.Println("Linux/Ubuntu 检测到：执行系统级快捷键和工作区切换锁定...")
	
	// 忽略执行可能抛出的错误（因为不一定是 GNOME 桌面环境）
	// 禁用左右工作区切换快捷键
	exec.Command("gsettings", "set", "org.gnome.desktop.wm.keybindings", "switch-to-workspace-left", "[]").Run()
	exec.Command("gsettings", "set", "org.gnome.desktop.wm.keybindings", "switch-to-workspace-right", "[]").Run()
	
	// 禁用上下工作区切换（触屏从顶部/底部边缘滑动触发）
	exec.Command("gsettings", "set", "org.gnome.desktop.wm.keybindings", "switch-to-workspace-up", "[]").Run()
	exec.Command("gsettings", "set", "org.gnome.desktop.wm.keybindings", "switch-to-workspace-down", "[]").Run()

	// 禁用 Alt+Tab 切换窗口等
	exec.Command("gsettings", "set", "org.gnome.desktop.wm.keybindings", "switch-applications", "[]").Run()
	exec.Command("gsettings", "set", "org.gnome.desktop.wm.keybindings", "switch-windows", "[]").Run()
	
	// 禁用截图 shortcuts
	exec.Command("gsettings", "set", "org.gnome.settings-daemon.plugins.media-keys", "screenshot", "[]").Run()

	// === 关键：禁用 GNOME 边缘手势和热角 ===
	// 禁用热角（左上角触发 Activities）
	exec.Command("gsettings", "set", "org.gnome.desktop.interface", "enable-hot-corners", "false").Run()
	// 禁用 Super/Overlay 键（防止触发 Activities Overview）
	exec.Command("gsettings", "set", "org.gnome.mutter", "overlay-key", "").Run()
	// 禁用窗口拖拽到边缘自动平铺
	exec.Command("gsettings", "set", "org.gnome.mutter", "edge-tiling", "false").Run()
	// 禁用动态工作区（固定为1个工作区，防止手势切换）
	exec.Command("gsettings", "set", "org.gnome.mutter", "dynamic-workspaces", "false").Run()
	exec.Command("gsettings", "set", "org.gnome.desktop.wm.preferences", "num-workspaces", "1").Run()
	// 禁用可被手势触发的 Activities 按钮
	exec.Command("gsettings", "set", "org.gnome.shell", "favorite-apps", "[]").Run()

	// 禁用锁屏（以防触摸误操作锁屏）
	exec.Command("gsettings", "set", "org.gnome.desktop.screensaver", "lock-enabled", "false").Run()
	exec.Command("gsettings", "set", "org.gnome.desktop.session", "idle-delay", "0").Run()

	log.Println("GNOME 系统锁定配置已应用（热角、边缘手势、工作区切换等已禁用）")
}

