// Frameless shell: custom titlebar (drag + min/max/close), left nav,
// view router, toasts, and a promise-based confirm dialog.
import { api, isWails, win, type Ctx } from "./api";
import { APP_VERSION } from "./version";
import { mountConsole } from "./views/console";
import { mountImport } from "./views/import";
import { mountLibrary } from "./views/library";
import { mountSettings } from "./views/settings";

type View = "console" | "import" | "files" | "settings";

const ctx: Ctx = { toast, confirm };

function toast(msg: string): void {
  const host = document.getElementById("toasts")!;
  const el = document.createElement("div");
  el.className = "toast";
  el.textContent = msg;
  host.appendChild(el);
  window.setTimeout(() => el.classList.add("show"));
  window.setTimeout(() => {
    el.classList.remove("show");
    window.setTimeout(() => el.remove(), 300);
  }, 3200);
}

function confirm(msg: string): Promise<boolean> {
  return new Promise((resolve) => {
    const overlay = document.getElementById("confirm-overlay")!;
    const text = document.getElementById("confirm-text")!;
    text.textContent = msg;
    overlay.classList.remove("hidden");
    const done = (v: boolean) => {
      overlay.classList.add("hidden");
      resolve(v);
    };
    document.getElementById("confirm-yes")!.onclick = () => done(true);
    document.getElementById("confirm-no")!.onclick = () => done(false);
  });
}

function shell(): void {
  document.getElementById("app")!.innerHTML = `
    <div class="titlebar" data-wails-drag>
      <div class="tb-left">
        <span class="appdot"></span>
        <span class="tb-name">WayerPC</span>
        <span id="tb-ver" class="verchip">v${APP_VERSION}</span>
        ${isWails() ? "" : '<span class="verchip mock">browser preview</span>'}
      </div>
      <div class="tb-right no-drag">
        <button id="tb-min" title="Minimize">_</button>
        <button id="tb-max" title="Maximize">▢</button>
        <button id="tb-close" title="Close">✕</button>
      </div>
    </div>
    <div class="body">
      <nav class="sidebar">
        <div class="brand"><b>WayerPC</b><span class="muted small">Phone ↔ PC transfer</span></div>
        <button data-view="console" class="on">Console</button>
        <button data-view="import">Import</button>
        <button data-view="files">Files</button>
        <button data-view="settings">Settings</button>
        <div class="spacer"></div>
        <button id="nav-quit" class="danger-link">Exit</button>
      </nav>
      <main id="view" class="view"></main>
    </div>
    <div id="toasts" class="toasts"></div>
    <div id="confirm-overlay" class="overlay hidden">
      <div class="dialog">
        <p id="confirm-text"></p>
        <div class="row end">
          <button id="confirm-no" class="btn ghost">Cancel</button>
          <button id="confirm-yes" class="btn primary">Confirm</button>
        </div>
      </div>
    </div>`;
}

function mount(view: View): () => void {
  const el = document.getElementById("view")!;
  el.innerHTML = "";
  document.querySelectorAll<HTMLButtonElement>(".sidebar button[data-view]").forEach((b) => {
    b.classList.toggle("on", b.dataset.view === view);
  });
  switch (view) {
    case "import":
      return mountImport(el, ctx);
    case "files":
      return mountLibrary(el, ctx);
    case "settings":
      return mountSettings(el, ctx);
    case "console":
    default:
      return mountConsole(el, ctx);
  }
}

function boot(): void {
  shell();
  let unmount: () => void = () => {};
  const go = (v: View) => {
    unmount();
    unmount = mount(v);
  };
  document.querySelectorAll<HTMLButtonElement>(".sidebar button[data-view]").forEach((b) => {
    b.addEventListener("click", () => go(b.dataset.view as View));
  });
  document.getElementById("nav-quit")!.addEventListener("click", () => win.quit());
  document.getElementById("tb-min")!.addEventListener("click", () => win.min());
  document.getElementById("tb-max")!.addEventListener("click", () => win.toggleMax());
  document.getElementById("tb-close")!.addEventListener("click", () => win.quit());
  document.addEventListener("keydown", (e) => {
    if (!e.ctrlKey) return;
    const map: Record<string, View> = { "1": "console", "2": "import", "3": "files", "4": "settings" };
    const v = map[e.key];
    if (v) {
      e.preventDefault();
      go(v);
    }
  });
  // Real version (Go is authoritative; generated TS can lag between edits).
  api
    .getVersion()
    .then((v) => {
      const chip = document.getElementById("tb-ver");
      if (chip) chip.textContent = `v${v.version}`;
    })
    .catch(() => {});
  go("console");
}

document.readyState === "loading" ? document.addEventListener("DOMContentLoaded", boot) : boot();
