// 在隔离环境中注册 DOM 事件；页面 API 钩子由 fullscreen.js 负责。
(() => {
  const edge = 24;
  const threshold = 8;
  let start = null;

  // composedPath 同时覆盖普通 DOM 和 Shadow DOM 中的内部滚动容器。
  function canScroll(path, delta, horizontal) {
    for (const element of path) {
      if (!(element instanceof Element)) continue;
      const style = getComputedStyle(element);
      const overflow = horizontal ? style.overflowX : style.overflowY;
      if (element !== document.scrollingElement && !/^(auto|scroll|overlay)$/.test(overflow)) continue;
      const position = horizontal ? element.scrollLeft : element.scrollTop;
      const maximum = horizontal
        ? element.scrollWidth - element.clientWidth
        : element.scrollHeight - element.clientHeight;
      if (maximum > 0 && (delta > 0 ? position > 0 : position < maximum - 1)) return true;
    }
    return false;
  }

  document.addEventListener("touchstart", (event) => {
    start = event.touches.length === 1 ? {
      x: event.touches[0].clientX,
      y: event.touches[0].clientY,
      path: event.composedPath(),
    } : null;
  }, { passive: true, capture: true });

  document.addEventListener("touchmove", (event) => {
    if (event.touches.length > 1) {
      if (event.cancelable) event.preventDefault();
      return;
    }
    if (!start || event.touches.length === 0) return;
    const dx = event.touches[0].clientX - start.x;
    const dy = event.touches[0].clientY - start.y;
    if (Math.max(Math.abs(dx), Math.abs(dy)) < threshold) return;
    const horizontal = Math.abs(dx) > Math.abs(dy);
    if (canScroll(start.path, horizontal ? dx : dy, horizontal)) return;
    const edgeGesture = horizontal
      ? (start.x < edge && dx > 0) || (start.x > innerWidth - edge && dx < 0)
      : start.y < edge && dy > 0;
    if (edgeGesture && event.cancelable) event.preventDefault();
  }, { passive: false, capture: true });

  for (const type of ["touchend", "touchcancel"]) {
    document.addEventListener(type, () => { start = null; }, { passive: true, capture: true });
  }

  window.addEventListener("keydown", (event) => {
    const key = event.key.toLowerCase();
    const modifier = event.ctrlKey || event.metaKey;
    if (["f5", "f11", "f12", "escape"].includes(key) ||
        (modifier && key === "r") ||
        (modifier && event.shiftKey && ["i", "j", "c"].includes(key))) {
      event.preventDefault();
    }
  }, { capture: true });

  window.addEventListener("contextmenu", (event) => event.preventDefault(), { capture: true });
  window.addEventListener("wheel", (event) => {
    if (event.ctrlKey) event.preventDefault();
  }, { passive: false, capture: true });
})();
