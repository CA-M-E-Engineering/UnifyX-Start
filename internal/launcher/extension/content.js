/**
 * UnifyX Kiosk Guard - Content Script
 *
 * 此脚本通过 Chrome 扩展的 content_scripts 机制注入，
 * 在 document_start 阶段执行，确保在页面任何代码之前生效。
 * 与 chromedp 的 Evaluate 注入不同，此脚本：
 *   - 每次导航/刷新自动重新注入（不会丢失）
 *   - 覆盖所有 iframe（all_frames: true）
 *   - 不依赖任何外部连接
 */

// === 1. 劫持 Fullscreen API ===
// kiosk 模式下已经是全屏，网页再调 requestFullscreen() 会导致
// Chrome 弹出 "To exit full screen, press Esc" 提示气泡
(function () {
  // 将 requestFullscreen 替换为空操作（返回 resolved Promise 假装成功）
  Element.prototype.requestFullscreen = function () {
    return Promise.resolve();
  };
  if (Element.prototype.webkitRequestFullscreen) {
    Element.prototype.webkitRequestFullscreen = function () {
      return Promise.resolve();
    };
  }
  if (Element.prototype.webkitRequestFullScreen) {
    Element.prototype.webkitRequestFullScreen = function () {
      return Promise.resolve();
    };
  }

  // 伪装 document 的全屏状态（让网页认为已处于全屏）
  Object.defineProperty(document, "fullscreenElement", {
    get: () => document.documentElement,
    configurable: true,
  });
  Object.defineProperty(document, "webkitFullscreenElement", {
    get: () => document.documentElement,
    configurable: true,
  });
  Object.defineProperty(document, "fullscreen", {
    get: () => true,
    configurable: true,
  });
  Object.defineProperty(document, "webkitIsFullScreen", {
    get: () => true,
    configurable: true,
  });

  // exitFullscreen 也变成空操作
  document.exitFullscreen = function () {
    return Promise.resolve();
  };
  if (document.webkitExitFullscreen) {
    document.webkitExitFullscreen = function () {
      return Promise.resolve();
    };
  }
})();

// === 2. 注入 CSS 禁用 overscroll（防止下拉刷新/退出全屏） ===
(function () {
  const style = document.createElement("style");
  style.id = "__kiosk_overscroll_lock";
  style.textContent = [
    "html, body, * { overscroll-behavior: none !important; }",
    "* { cursor: none !important; }", // 触屏场景隐藏光标
  ].join("\n");
  (document.head || document.documentElement).appendChild(style);
})();

// === 3. 拦截触摸事件，阻止边缘手势（左侧、顶部等） ===
(function () {
  let startX = 0;
  let startY = 0;
  document.addEventListener(
    "touchstart",
    function (e) {
      startX = e.touches[0].clientX;
      startY = e.touches[0].clientY;
    },
    { passive: false }
  );

  document.addEventListener(
    "touchmove",
    function (e) {
      const currentX = e.touches[0].clientX;
      const currentY = e.touches[0].clientY;
      
      // 3.1 如果从屏幕顶部 (前5%区域) 开始向下滑，直接拦截气泡提醒或下拉
      if (startY < window.innerHeight * 0.05 && currentY > startY) {
        e.preventDefault();
        e.stopPropagation();
        return false;
      }
      
      // 3.2 拦截从屏幕左侧 (前10%区域) 向右滑，防止触发“后退”历史
      if (startX < window.innerWidth * 0.1 && currentX > startX) {
        e.preventDefault();
        e.stopPropagation();
        return false;
      }
      
      // 3.3 拦截从屏幕右侧 (后10%区域) 向左滑，防止触发“前进”历史
      if (startX > window.innerWidth * 0.9 && currentX < startX) {
        e.preventDefault();
        e.stopPropagation();
        return false;
      }
      
      // 3.4 页面在顶部且用户向下拉时也拦截（防止原生的 pull-to-refresh）
      if (window.scrollY === 0 && currentY > startY) {
        e.preventDefault();
      }
    },
    { passive: false }
  );
})();

// === 4. 键盘快捷键屏蔽 ===
window.addEventListener("keydown", (e) => {
  if (
    e.key === "F5" ||
    (e.ctrlKey && e.key === "r") ||
    e.key === "F12" ||
    e.key === "F11" ||
    (e.ctrlKey && e.shiftKey && e.key === "I") ||
    e.key === "Escape"
  ) {
    e.preventDefault();
  }
});

// === 5. 屏蔽右键菜单 ===
window.addEventListener("contextmenu", (e) => e.preventDefault());
