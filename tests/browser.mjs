// Optional integration test: BROWSER_PATH must point to Edge / Chromium / Chrome for Testing.
// Node.js 22+; no npm dependencies. Uses an isolated temporary browser profile.
import assert from "node:assert/strict";
import { spawn, execFile } from "node:child_process";
import { mkdtemp, readFile, rm } from "node:fs/promises";
import { createServer } from "node:http";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { promisify } from "node:util";

assert.ok(process.env.BROWSER_PATH, "Set BROWSER_PATH to Edge / Chromium / Chrome for Testing");
const extension = resolve(dirname(fileURLToPath(import.meta.url)), "../internal/launcher/extension");
const prefix = join(tmpdir(), "unifyx-browser-test-");
const profile = await mkdtemp(prefix);
const sleep = (ms) => new Promise((done) => setTimeout(done, ms));
async function until(check) {
  const deadline = Date.now() + 15000;
  while (Date.now() < deadline) {
    const result = await check();
    if (result) return result;
    await sleep(50);
  }
  throw new Error("Timed out waiting for browser");
}

const server = createServer((req, res) => {
  res.writeHead(200, { "Content-Type": "text/html; charset=utf-8" });
  res.end(`<!doctype html><html><head><script>
    window.earlyGuard = document.fullscreenElement === document.documentElement;
    window.scriptErrors = [];
    window.addEventListener('error', e => scriptErrors.push(e.message));
  </script><style>
    html,body { margin:0; height:100%; overflow:hidden; }
    #list { position:absolute; left:60px; top:60px; width:300px; height:250px; overflow:auto; }
    #items { height:1500px; background:linear-gradient(white,blue); }
    iframe { position:absolute; left:500px; top:60px; width:200px; height:100px; }
  </style></head><body><div id="list"><div id="items">Scrollable list</div></div>
  <iframe srcdoc="<p>frame</p>"></iframe></body></html>`);
});
await new Promise((done) => server.listen(0, "127.0.0.1", done));
const url = `http://127.0.0.1:${server.address().port}/`;
const browser = spawn(process.env.BROWSER_PATH, [
  "--headless=new", "--remote-debugging-port=0", `--user-data-dir=${profile}`,
  `--disable-extensions-except=${extension}`, `--load-extension=${extension}`,
  "--no-first-run", "--no-default-browser-check", "--disable-background-networking",
  "--edge-skip-compat-layer-relaunch",
  "--disable-sync", "--window-size=1000,800", "about:blank",
], { windowsHide: true, stdio: ["ignore", "ignore", "pipe"] });
let stderr = "";
browser.stderr.on("data", (data) => { stderr = (stderr + data).slice(-4000); });
const exited = new Promise((done) => {
  browser.once("exit", done);
  browser.once("error", done);
});
let socket;
let send;
try {
  const endpoint = await until(async () => {
    if (browser.exitCode !== null) throw new Error(`Browser exited (${browser.exitCode}): ${stderr}`);
    try { return await readFile(join(profile, "DevToolsActivePort"), "utf8"); }
    catch (error) { if (error.code !== "ENOENT") throw error; }
  });
  const [port, route] = endpoint.trim().split(/\r?\n/);
  socket = new WebSocket(`ws://127.0.0.1:${port}${route}`);
  await new Promise((done, reject) => {
    socket.addEventListener("open", done, { once: true });
    socket.addEventListener("error", reject, { once: true });
  });
  let nextID = 0;
  const pending = new Map();
  socket.addEventListener("message", (event) => {
    const message = JSON.parse(event.data);
    const request = pending.get(message.id);
    if (!request) return;
    pending.delete(message.id);
    clearTimeout(request.timer);
    if (message.error) request.reject(new Error(JSON.stringify(message.error)));
    else request.resolve(message.result);
  });
  send = (method, params = {}, sessionId) => new Promise((resolve, reject) => {
    const id = ++nextID;
    const timer = setTimeout(() => { pending.delete(id); reject(new Error(`CDP timeout: ${method}`)); }, 5000);
    pending.set(id, { resolve, reject, timer });
    socket.send(JSON.stringify({ id, method, params, sessionId }));
  });
  const { targetId } = await send("Target.createTarget", { url });
  const { sessionId } = await send("Target.attachToTarget", { targetId, flatten: true });
  const evaluate = async (expression) => {
    const result = await send("Runtime.evaluate", { expression, returnByValue: true, awaitPromise: true }, sessionId);
    assert.equal(result.exceptionDetails, undefined, JSON.stringify(result.exceptionDetails));
    return result.result.value;
  };
  const ready = () => until(() => evaluate("document.readyState === 'complete' && !!document.querySelector('iframe')?.contentDocument?.body"));
  await ready();
  assert.equal(await evaluate("earlyGuard"), true, "hook must run before the first page script");
  assert.equal(await evaluate("document.body.requestFullscreen().then(() => document.fullscreenElement === document.documentElement)"), true);
  assert.equal(await evaluate("document.querySelector('iframe').contentDocument.fullscreenElement === document.querySelector('iframe').contentDocument.documentElement"), true, "about:srcdoc iframe guard");
  assert.equal(await evaluate("getComputedStyle(document.documentElement).overscrollBehaviorY"), "none");
  assert.equal(await evaluate("getComputedStyle(document.body).cursor"), "none");
  assert.equal(await evaluate("!document.body.dispatchEvent(new MouseEvent('contextmenu', {bubbles:true, cancelable:true}))"), true);
  assert.equal(await evaluate("!document.body.dispatchEvent(new KeyboardEvent('keydown', {key:'R', ctrlKey:true, shiftKey:true, bubbles:true, cancelable:true}))"), true);

  const touchResults = await evaluate(`(() => {
    const list = document.getElementById('list');
    function gesture(target, x, y, dx, dy, count=1) {
      const touches = (move) => Array.from({length:count}, (_,i) => new Touch({identifier:i, target, clientX:x+i*20+(move?dx:0), clientY:y+(move?dy:0)}));
      target.dispatchEvent(new TouchEvent('touchstart', {touches:touches(false), bubbles:true, cancelable:true, composed:true}));
      const move = new TouchEvent('touchmove', {touches:touches(true), bubbles:true, cancelable:true, composed:true});
      target.dispatchEvent(move);
      target.dispatchEvent(new TouchEvent('touchend', {touches:[], bubbles:true}));
      return move.defaultPrevented;
    }
    list.scrollTop = 200;
    return {
      nestedScrollBlocked: gesture(list, 100, 100, 0, 40),
      scrollAtEdgeBlocked: gesture(list, 100, 5, 0, 40),
      backGestureBlocked: gesture(document.body, 2, 400, 60, 1),
      verticalAtSideBlocked: gesture(document.body, 2, 400, 1, 60),
      pinchBlocked: gesture(document.body, 400, 400, 20, 20, 2),
      windowScrollY: window.scrollY
    };
  })()`);
  assert.deepEqual(touchResults, {
    nestedScrollBlocked: false, scrollAtEdgeBlocked: false, backGestureBlocked: true,
    verticalAtSideBlocked: false, pinchBlocked: true, windowScrollY: 0,
  });

  // Actual browser scrolling, beyond checking synthetic event cancellation.
  await send("Emulation.setTouchEmulationEnabled", { enabled: true, maxTouchPoints: 5 }, sessionId);
  await evaluate("document.getElementById('list').scrollTop = 300");
  for (let step = 0; step <= 6; step++) {
    await send("Input.dispatchTouchEvent", {
      type: step === 0 ? "touchStart" : "touchMove",
      touchPoints: [{ x: 180, y: 130 + step * 15 }],
    }, sessionId);
    await sleep(30);
  }
  await send("Input.dispatchTouchEvent", { type: "touchEnd", touchPoints: [] }, sessionId);
  await until(() => evaluate("document.getElementById('list').scrollTop < 280"));
  assert.deepEqual(await evaluate("scriptErrors"), []);

  await send("Page.navigate", { url: url + "?navigation=2" }, sessionId);
  await until(() => evaluate("location.search === '?navigation=2' && document.readyState === 'complete'"));
  assert.equal(await evaluate("earlyGuard"), true, "guard must survive navigation");
  console.log("PASS: early MAIN hook, iframe, CSS, shortcuts, edge/pinch guard, nested scrolling, navigation");
} finally {
  if (send && socket?.readyState === WebSocket.OPEN) {
    try { await send("Browser.close"); } catch {}
    socket.close();
  }
  if (browser.exitCode === null) {
    await Promise.race([exited, sleep(1500)]);
    if (browser.exitCode === null && browser.pid) {
      if (process.platform === "win32") {
        await promisify(execFile)("taskkill.exe", ["/PID", String(browser.pid), "/T", "/F"], { windowsHide: true }).catch(() => {});
      } else browser.kill("SIGKILL");
    }
  }
  await exited;
  server.closeAllConnections();
  await new Promise((done) => server.close(done));
  assert.ok(resolve(profile).startsWith(resolve(prefix)), "cleanup must stay inside the temporary test directory");
  await rm(profile, { recursive: true, force: true, maxRetries: 5, retryDelay: 200 });
}
