// Kiosk 本身已全屏。此钩子必须在 MAIN world 中运行，才能覆盖网页调用。
(() => {
  const noop = () => Promise.resolve();
  for (const name of ["requestFullscreen", "webkitRequestFullscreen", "webkitRequestFullScreen"]) {
    if (name in Element.prototype) Element.prototype[name] = noop;
  }
  for (const name of ["exitFullscreen", "webkitExitFullscreen"]) {
    if (name in document) document[name] = noop;
  }
  for (const name of ["fullscreenElement", "webkitFullscreenElement"]) {
    Object.defineProperty(document, name, {
      get: () => document.documentElement,
      configurable: true,
    });
  }
  for (const name of ["fullscreen", "webkitIsFullScreen"]) {
    Object.defineProperty(document, name, { get: () => true, configurable: true });
  }
})();
