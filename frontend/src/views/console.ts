// Console (concept 01): status line with the address a phone can dial, four
// hairline-separated stats, and the structured activity table.
//
// Performance notes: `stats` arrives every second, so status values are
// patched node-by-node and the activity table only appends the rows that are
// actually new. A filter change (rare) rebuilds.
import { api, type ActivityRow, type Ctx } from "../api";
import { activityLabel, esc, fmtBytes, fmtClock, fmtUptime, q } from "../dom";
import { icon } from "../icons";
import { clearActivity, phoneAddr, refreshStatus, store, listen } from "../state";

type Tone = "st-green" | "st-red" | "st-amber" | "st-muted";

export function mountConsole(root: HTMLElement, _ctx: Ctx): () => void {
  root.innerHTML = `
    <section class="page">
      <header class="page-head">
        <div>
          <h1 class="screen">Console</h1>
          <p class="screen-sub">Server activity and transfer overview</p>
        </div>
      </header>

      <div class="statusline">
        <div class="left">
          <span id="c-dot" class="dot amber"></span>
          <span id="c-state" class="muted">Starting…</span>
          <button id="c-toggle" class="link" hidden></button>
        </div>
        <div class="left">
          <span id="c-uptime" class="uptime">UPTIME 00:00:00</span>
        </div>
      </div>

      <div class="stats">
        <div class="stat">
          <div class="k">Active links</div>
          <div class="v" id="c-active">0</div>
          <div class="s" id="c-active-sub">&nbsp;</div>
        </div>
        <div class="stat">
          <div class="k">Handled</div>
          <div class="v" id="c-total">0</div>
          <div class="s" id="c-total-sub">&nbsp;</div>
        </div>
        <div class="stat">
          <div class="k">Outbound</div>
          <div class="v" id="c-sent">0 B</div>
          <div class="s" id="c-sent-sub">0 files</div>
        </div>
        <div class="stat">
          <div class="k">Inbound</div>
          <div class="v" id="c-recv">0 B</div>
          <div class="s" id="c-recv-sub">0 files</div>
        </div>
      </div>

      <span class="section-label">Activity log</span>
      <div class="listhead">
        <div class="tabletools">
          <label class="field sm" style="width:230px">
            ${icon("find", 13)}
            <input id="c-filter" type="search" placeholder="Filter activity" aria-label="Filter activity" />
          </label>
          <button id="c-pause" class="link">Pause</button>
          <button id="c-clear" class="link">Clear</button>
        </div>
        <div class="small muted" id="c-count">0 events</div>
      </div>
      <table class="table">
        <thead>
          <tr><th>Time</th><th>Event</th><th>Detail</th><th style="text-align:right">Status</th></tr>
        </thead>
        <tbody id="c-rows"></tbody>
      </table>
      <div id="c-empty" class="state hidden">
        <span class="rule"></span>
        <div class="title">No activity yet</div>
        <div class="body">Server events and file transfers appear here as they happen.</div>
      </div>
    </section>`;

  const $ = <T extends HTMLElement = HTMLElement>(id: string) => q<T>(root, `#${id}`);
  const rows = $("c-rows");
  const filter = $<HTMLInputElement>("c-filter");
  const pauseBtn = $("c-pause");
  const empty = $("c-empty");
  const countEl = $("c-count");

  let paused = false;
  let pausedFrom = 0;
  let rendered = 0;
  let filterText = "";
  let lastStatusKey = "";

  const tone = (status: string): Tone => {
    const s = (status || "").toLowerCase();
    if (s.includes("fail") || s.includes("error") || s.includes("not found") || s.includes("stopped") || s.includes("denied"))
      return "st-red";
    if (s.includes("pending") || s.includes("wait") || s.includes("cancel") || s.includes("disconnect")) return "st-amber";
    if (s === "" || s === "—") return "st-muted";
    return "st-green";
  };

  const matches = (row: ActivityRow): boolean => {
    if (!filterText) return true;
    const haystack = `${activityLabel(row.kind)} ${row.detail} ${row.status} ${row.device ?? ""}`.toLowerCase();
    return haystack.includes(filterText);
  };

  const rowHtml = (row: ActivityRow): string => `
    <tr>
      <td class="time">${esc(fmtClock(row.ts))}</td>
      <td class="event">${esc(activityLabel(row.kind))}</td>
      <td class="detail">${esc(row.detail)}${row.device ? ` <span class="faint">· ${esc(row.device)}</span>` : ""}</td>
      <td class="status ${tone(row.status)}">${esc(row.status || "—")}</td>
    </tr>`;

  const rebuild = (): void => {
    const all = store.activity.filter(matches);
    rows.innerHTML = all.map(rowHtml).join("");
    rendered = store.activity.length;
    countEl.textContent = `${all.length} event${all.length === 1 ? "" : "s"}`;
    empty.classList.toggle("hidden", all.length > 0);
  };

  const appendNew = (): void => {
    const total = store.activity.length;
    if (total <= rendered) return; // nothing new (stats ticks land here)
    const fresh = store.activity.slice(rendered);
    rendered = total;
    const visible = fresh.filter(matches);
    if (visible.length > 0) {
      rows.insertAdjacentHTML("beforeend", visible.map(rowHtml).join(""));
      empty.classList.add("hidden");
    }
    countEl.textContent = `${rows.children.length} event${rows.children.length === 1 ? "" : "s"}`;
  };

  const paintStats = (): void => {
    const st = store.status;
    const srv = st?.server;
    const dot = $("c-dot");
    const state = $("c-state");
    const toggle = $<HTMLButtonElement>("c-toggle");
    if (!srv) {
      state.textContent = "Waiting for the backend…";
      return;
    }
    const addr = phoneAddr(st ?? null);
    if (srv.running) {
      dot.className = "dot green";
      state.innerHTML = `Listening on <span class="mono">${esc(addr)}</span>`;
      toggle.hidden = false;
      toggle.textContent = "Stop server";
    } else {
      dot.className = "dot red";
      state.textContent = "Server stopped";
      toggle.hidden = false;
      toggle.textContent = "Start server";
    }
    $("c-uptime").textContent = `UPTIME ${fmtUptime(srv.uptimeSecs)}`;
    $("c-active").textContent = String(srv.active);
    $("c-active-sub").innerHTML = srv.active > 0 ? "phone link open" : "&nbsp;";
    $("c-total").textContent = String(srv.total);
    $("c-total-sub").innerHTML = srv.total > 0 ? "connections handled" : "&nbsp;";
    $("c-sent").textContent = fmtBytes(srv.bytesSent);
    $("c-sent-sub").textContent = `${srv.filesSent ?? 0} file${(srv.filesSent ?? 0) === 1 ? "" : "s"}`;
    $("c-recv").textContent = fmtBytes(srv.bytesReceived);
    $("c-recv-sub").textContent = `${srv.filesReceived ?? 0} file${(srv.filesReceived ?? 0) === 1 ? "" : "s"}`;
  };

  // Stats land once a second: repaint only when a displayed value changed.
  const paint = (): void => {
    const srv = store.status?.server;
    const key = srv
      ? `${srv.running}|${srv.active}|${srv.total}|${srv.bytesSent}|${srv.bytesReceived}|${srv.uptimeSecs}|${store.status?.lanAddr}`
      : "none";
    if (key !== lastStatusKey) {
      lastStatusKey = key;
      paintStats();
    }
    if (paused) {
      const pending = store.activity.length - pausedFrom;
      pauseBtn.textContent = pending > 0 ? `Resume (${pending} new)` : "Resume";
      return;
    }
    appendNew();
  };

  filter.addEventListener("input", () => {
    filterText = filter.value.trim().toLowerCase();
    rebuild();
  });
  pauseBtn.addEventListener("click", () => {
    paused = !paused;
    if (paused) {
      pausedFrom = store.activity.length;
      pauseBtn.textContent = "Resume";
    } else {
      pauseBtn.textContent = "Pause";
      rebuild();
    }
  });
  $("c-clear").addEventListener("click", () => {
    void clearActivity().then(() => rebuild());
  });
  $("c-toggle").addEventListener("click", async () => {
    const running = store.status?.server?.running;
    if (running) {
      await api.stopServer().catch(() => {});
    } else {
      await api.startServer().catch(() => {});
    }
    await refreshStatus();
  });

  const off = listen(paint);
  rebuild();
  paintStats();

  return () => {
    off();
  };
}
