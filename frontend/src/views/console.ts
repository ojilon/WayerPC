// Console view: server status hero, network stats, live log with
// filter/pause/autoscroll, start-stop controls.
import { api, fmtMB, onEvent, type Ctx, type Status } from "../api";

export function mountConsole(root: HTMLElement, ctx: Ctx): () => void {
  root.innerHTML = `
    <section class="page">
      <header class="page-head">
        <div>
          <h2>Console</h2>
          <p class="muted">TCP link to the Android client + live activity.</p>
        </div>
        <div class="row">
          <button id="c-start" class="btn primary">Start server</button>
          <button id="c-stop" class="btn ghost">Stop</button>
        </div>
      </header>
      <div id="c-banner" class="banner warn hidden"></div>
      <div class="cards">
        <div class="hero card">
          <span id="c-dot" class="dot off"></span>
          <div>
            <div id="c-state" class="hero-state">LOADING…</div>
            <div id="c-addr" class="muted small"></div>
          </div>
          <div class="hero-right muted small" id="c-uptime">Uptime —</div>
        </div>
        <div class="statgrid">
          <div class="card stat"><span class="muted">Active links</span><b id="c-active">0</b></div>
          <div class="card stat"><span class="muted">Handled</span><b id="c-total">0</b></div>
          <div class="card stat"><span class="muted">Outbound</span><b id="c-sent">0 MB</b></div>
          <div class="card stat"><span class="muted">Inbound</span><b id="c-recv">0 MB</b></div>
        </div>
      </div>
      <div class="card logcard">
        <div class="row between">
          <b>Activity</b>
          <div class="row">
            <input id="c-filter" class="input" placeholder="Filter log…" />
            <button id="c-pause" class="btn ghost sm">Pause</button>
            <button id="c-clear" class="btn ghost sm">Clear</button>
          </div>
        </div>
        <div id="c-log" class="log" aria-live="polite"></div>
      </div>
    </section>`;

  const $ = (id: string) => root.querySelector<HTMLElement>(`#${id}`)!;
  const logEl = $("c-log") as HTMLElement;
  let paused = false;
  let filter = "";

  const append = (line: string) => {
    if (paused) return;
    if (filter && !line.toLowerCase().includes(filter)) return;
    const ts = new Date().toLocaleTimeString();
    const div = document.createElement("div");
    div.className = "logline";
    div.textContent = `[${ts}] ${line}`;
    logEl.appendChild(div);
    while (logEl.children.length > 2000) logEl.firstChild?.remove();
    logEl.scrollTop = logEl.scrollHeight;
  };

  const paint = (st: Status) => {
    const running = st.server?.running ?? false;
    ($("c-dot") as HTMLElement).className = `dot ${running ? "on" : "off"}`;
    $("c-state").textContent = running ? "ONLINE" : "OFFLINE";
    $("c-state").className = `hero-state ${running ? "ok" : "bad"}`;
    $("c-addr").textContent = st.server?.addr ? `Listening on ${st.server.addr}` : "";
    $("c-uptime").textContent = running ? `Uptime ${st.server.uptimeSecs}s` : "Uptime —";
    $("c-active").textContent = String(st.server?.active ?? 0);
    $("c-total").textContent = String(st.server?.total ?? 0);
    $("c-sent").textContent = fmtMB(st.server?.bytesSent ?? 0);
    $("c-recv").textContent = fmtMB(st.server?.bytesReceived ?? 0);
    const banner = $("c-banner") as HTMLElement;
    if (!st.onboarded) {
      banner.classList.remove("hidden");
      banner.textContent = "No data folder yet — open Settings and choose where WayerPC should live.";
    } else {
      banner.classList.add("hidden");
    }
  };

  const offs = [
    onEvent("log", (line) => append(String(line ?? ""))),
    onEvent("stats", (st) => paint(st as Status)),
    onEvent("catalog-changed", () => api.getStatus().then(paint).catch(() => {})),
  ];

  ($("c-filter") as HTMLInputElement).addEventListener("input", (e) => {
    filter = (e.target as HTMLInputElement).value.trim().toLowerCase();
  });
  ($("c-pause") as HTMLButtonElement).addEventListener("click", (e) => {
    paused = !paused;
    (e.target as HTMLButtonElement).textContent = paused ? "Resume" : "Pause";
  });
  ($("c-clear") as HTMLButtonElement).addEventListener("click", () => {
    logEl.innerHTML = "";
    api.clearLogs().catch(() => {});
  });
  ($("c-start") as HTMLButtonElement).addEventListener("click", async () => {
    ctx.toast(await api.startServer().catch((e) => String(e)));
    api.getStatus().then(paint).catch(() => {});
  });
  ($("c-stop") as HTMLButtonElement).addEventListener("click", async () => {
    ctx.toast(await api.stopServer().catch((e) => String(e)));
    api.getStatus().then(paint).catch(() => {});
  });

  api.getLogs().then((lines) => lines.forEach(append)).catch(() => {});
  api.getStatus().then(paint).catch(() => {});
  const timer = window.setInterval(() => api.getStatus().then(paint).catch(() => {}), 2000);

  return () => {
    offs.forEach((off) => off());
    window.clearInterval(timer);
  };
}
