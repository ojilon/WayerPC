// Frameless shell: 44px title bar (drag region + real window controls),
// sidebar (208 <-> 56px, Ctrl+B, persisted), hash router, and the global
// transfer strip. Views are mounted into <main class="view">.
import { api, isWails, win, type Ctx } from "./api";
import { esc } from "./dom";
import { icon } from "./icons";
import { currentRoute, navigate, type Route } from "./nav";
import { initState, linkState, listen, store } from "./state";
import { mountTransferStrip } from "./strip";
import { mountConsole } from "./views/console";
import { mountFiles } from "./views/files";
import { mountFind } from "./views/find";
import { mountImport } from "./views/import";
import { mountSettings } from "./views/settings";
import { mountSync } from "./views/sync";

const NAV: { route: Route; label: string; icon: string; match: Route[] }[] = [
  { route: "console", label: "Console", icon: "console", match: ["console"] },
  { route: "files", label: "Files", icon: "files", match: ["files"] },
  { route: "find", label: "Find", icon: "find", match: ["find", "find/phone", "find/requests", "find/rules"] },
  { route: "sync", label: "Sync", icon: "sync", match: ["sync"] },
  { route: "import", label: "Import", icon: "import", match: ["import"] },
  { route: "settings", label: "Settings", icon: "settings", match: ["settings"] },
];

const ctx: Ctx = { toast, confirm };

let unmount: () => void = () => {};

function toast(msg: string): void {
  const host = document.getElementById("toasts");
  if (!host) return;
  const el = document.createElement("div");
  el.className = "toast";
  el.textContent = msg;
  host.appendChild(el);
  window.setTimeout(() => el.classList.add("show"));
  window.setTimeout(() => {
    el.classList.remove("show");
    window.setTimeout(() => el.remove(), 200);
  }, 3200);
}

function confirm(msg: string): Promise<boolean> {
  return new Promise((resolve) => {
    const overlay = document.getElementById("confirm-overlay");
    const text = document.getElementById("confirm-text");
    if (!overlay || !text) {
      resolve(window.confirm(msg));
      return;
    }
    text.textContent = msg;
    overlay.classList.remove("hidden");
    const done = (v: boolean) => {
      overlay.classList.add("hidden");
      resolve(v);
    };
    const yes = document.getElementById("confirm-yes");
    const no = document.getElementById("confirm-no");
    if (yes) yes.onclick = () => done(true);
    if (no) no.onclick = () => done(false);
  });
}

function shell(): void {
  const app = document.getElementById("app")!;
  app.innerHTML = `
    <header class="titlebar" data-wails-drag>
      <span class="brand">WayerPC</span>
      <span id="tb-ver" class="verchip">v0.0.0</span>
      ${"" /* browsers have no frameless bar; flag the preview below the chip */}
      <span class="tb-spacer"></span>
      <span id="tb-status" class="conn"><span class="dot amber"></span><span class="conn-label">Starting…</span></span>
      <div class="wincontrols no-drag">
        <button id="tb-min" title="Minimise" aria-label="Minimise">&#8211;</button>
        <button id="tb-max" title="Maximise" aria-label="Maximise">&#9723;</button>
        <button id="tb-close" class="close" title="Close" aria-label="Close">&#10005;</button>
      </div>
    </header>
    <div class="body">
      <aside class="sidebar">
        <nav>
          ${NAV.map(
            (n) => `<button class="navitem" data-route="${n.route}" data-tip="${esc(n.label)}" aria-label="${esc(n.label)}">
              ${icon(n.icon, 16)}<span class="label">${esc(n.label)}</span></button>`,
          ).join("")}
        </nav>
        <div class="foot">
          <button id="nav-quit" class="navitem danger" data-tip="Exit" aria-label="Exit">
            ${icon("exit", 16)}<span class="label">Exit</span>
          </button>
          <button id="nav-collapse" class="collapse" data-tip="Expand sidebar" aria-label="Collapse sidebar">
            ${icon("chevronLeft", 14)}<span class="label">Collapse sidebar</span>
          </button>
        </div>
      </aside>
      <main id="view" class="view"></main>
    </div>
    <div id="strip-host"></div>
    <div id="toasts" class="toasts"></div>
    <div id="confirm-overlay" class="overlay hidden">
      <div class="dialog" role="dialog" aria-modal="true">
        <p id="confirm-text"></p>
        <div class="row">
          <button id="confirm-no" class="link muted">Cancel</button>
          <button id="confirm-yes" class="link">Confirm</button>
        </div>
      </div>
    </div>`;

  if (!isWails()) {
    const chip = document.getElementById("tb-ver");
    chip?.insertAdjacentHTML("afterend", '<span class="verchip" title="Vite dev server: no Go backend">preview</span>');
  }
  mountTransferStrip(document.getElementById("strip-host")!);
}

function paintTitleStatus(): void {
  const host = document.getElementById("tb-status");
  if (!host) return;
  const st = linkState(store.status);
  const dot = host.querySelector(".dot")!;
  dot.className = `dot ${st.tone}`;
  const label = host.querySelector(".conn-label")!;
  label.textContent = st.label;
}

function paintVersion(): void {
  const chip = document.getElementById("tb-ver");
  if (chip && store.versionInfo) {
    chip.textContent = `v${store.versionInfo.version}`;
    chip.title = `Protocol v${store.versionInfo.protocolVersion}`;
  }
}

function paintNav(route: Route): void {
  document.querySelectorAll<HTMLElement>(".navitem[data-route]").forEach((el) => {
    const nav = NAV.find((n) => n.route === el.dataset.route);
    el.classList.toggle("on", !!nav && nav.match.includes(route));
  });
}

function render(): void {
  const route = currentRoute();
  const el = document.getElementById("view");
  if (!el) return;
  unmount();
  paintNav(route);
  el.innerHTML = "";
  el.scrollTop = 0;
  switch (route) {
    case "files":
      unmount = mountFiles(el, ctx);
      break;
    case "find":
    case "find/phone":
    case "find/requests":
    case "find/rules":
      unmount = mountFind(el, ctx, route);
      break;
    case "sync":
      unmount = mountSync(el, ctx);
      break;
    case "import":
      unmount = mountImport(el, ctx);
      break;
    case "settings":
      unmount = mountSettings(el, ctx);
      break;
    case "console":
    default:
      unmount = mountConsole(el, ctx);
      break;
  }
}

function setCollapsed(collapsed: boolean, persist: boolean): void {
  document.body.classList.toggle("sidebar-collapsed", collapsed);
  const btn = document.getElementById("nav-collapse");
  if (btn) {
    btn.dataset.tip = collapsed ? "Expand sidebar" : "Collapse sidebar";
    btn.setAttribute("aria-label", collapsed ? "Expand sidebar" : "Collapse sidebar");
    btn.innerHTML = `${icon(collapsed ? "chevronRight" : "chevronLeft", 14)}<span class="label">Collapse sidebar</span>`;
  }
  if (persist) void api.setSetting("ui_sidebar_collapsed", collapsed ? "1" : "0").catch(() => {});
}

async function boot(): Promise<void> {
  shell();
  await initState();
  paintVersion();
  paintTitleStatus();

  listen(() => {
    paintTitleStatus();
    paintVersion();
  });

  // Sidebar collapse state is a persisted setting (Ctrl+B toggles it).
  let collapsed = false;
  try {
    const saved = await api.getSetting("ui_sidebar_collapsed");
    collapsed = saved === "1";
  } catch {
    collapsed = false;
  }
  setCollapsed(collapsed, false);

  document.querySelectorAll<HTMLElement>(".navitem[data-route]").forEach((el) => {
    el.addEventListener("click", () => navigate(el.dataset.route as Route));
  });
  document.getElementById("nav-quit")!.addEventListener("click", () => win.quit());
  document.getElementById("nav-collapse")!.addEventListener("click", () => {
    collapsed = !document.body.classList.contains("sidebar-collapsed");
    setCollapsed(collapsed, true);
  });
  document.getElementById("tb-min")!.addEventListener("click", () => win.min());
  document.getElementById("tb-max")!.addEventListener("click", () => win.toggleMax());
  document.getElementById("tb-close")!.addEventListener("click", () => win.quit());

  document.addEventListener("keydown", (ev) => {
    if (ev.ctrlKey && (ev.key === "b" || ev.key === "B")) {
      ev.preventDefault();
      collapsed = !document.body.classList.contains("sidebar-collapsed");
      setCollapsed(collapsed, true);
      return;
    }
    if (ev.ctrlKey && /^[1-6]$/.test(ev.key)) {
      ev.preventDefault();
      const target = NAV[Number(ev.key) - 1];
      if (target) navigate(target.route);
    }
  });

  window.addEventListener("hashchange", render);
  render();
}

document.readyState === "loading" ? document.addEventListener("DOMContentLoaded", () => void boot()) : void boot();
