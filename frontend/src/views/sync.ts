// Sync (concept 07). Two columns (this PC / phone), one direction arrow per
// row, "In both" collapsed.
//
// Phase 6 wiring: api.startSyncScan() fills the plan (the PC side is
// received/ + shared/ flattened by name, the phone side is a `scope:"sync"`
// manifest), `sync-result` refreshes it and `sync-progress` updates rows while
// api.applySync() copies the ticked files. Nothing is ever deleted, and the
// primary button is a single filled "Sync selected (N)" control.
import { api, onEvent, type Ctx, type SyncEntry, type SyncProgress, type SyncResult } from "../api";
import { esc, fmtBytes, fmtClock, q, qa } from "../dom";
import { icon } from "../icons";
import { listen, phoneAddr, phoneLinked, store } from "../state";

type Dir = "push" | "pull";

const NO_LINK = "Needs a connected phone (protocol v2 control link)";

export function mountSync(root: HTMLElement, ctx: Ctx): () => void {
  let rows: SyncEntry[] = [];
  let scannedAt = 0;
  let note = "";
  let error = "";
  let scanning = false;
  let applying = false;
  let runId = "";
  const selected = new Set<string>();
  /** Arrow overrides for files that exist on both sides. */
  const overrides = new Map<string, Dir>();
  const progress = new Map<string, SyncProgress>();

  const linked = () => phoneLinked();
  const dirOf = (e: SyncEntry): Dir => {
    const over = overrides.get(e.name);
    if (over) return over;
    return e.action === "pull" ? "pull" : "push";
  };

  const arrow = (dir: Dir, flippable: boolean) =>
    `<span class="arrow${flippable ? " dir" : ""}"${flippable ? ' title="Click to flip the direction"' : ' title="Only on one side — direction is fixed"'}>${icon(
      dir === "push" ? "arrowRight" : "arrowLeft",
      14,
    )}</span>`;
  const idleArrow = `<span class="arrow">${icon("arrowRight", 14)}</span>`;

  const stateOf = (name: string): SyncProgress | undefined => progress.get(name.toLowerCase());

  const item = (r: SyncEntry, dir: Dir | null, meta: string, flippable: boolean) => {
    const pr = stateOf(r.name);
    const badge = pr
      ? pr.state === "active"
        ? ` <span class="badge">copying…</span>`
        : pr.state === "done"
          ? ` <span class="badge">done</span>`
          : pr.state === "error"
            ? ` <span class="badge">failed</span>`
            : ""
      : "";
    return `<div class="sync-item">
      <input class="box" type="checkbox" data-name="${esc(r.name)}" ${selected.has(r.name) ? "checked" : ""} />
      ${dir ? arrow(dir, flippable) : idleArrow}
      <span class="nowrap">${esc(r.name)}${badge}</span>
      <span class="meta">${esc(meta)}</span>
    </div>`;
  };

  const group = (label: string, body: string) =>
    `<div class="sync-group"><div class="section-label">${esc(label)}</div>${body}</div>`;

  const render = (): void => {
    const onlyPc = rows.filter((r) => r.verdict === "only_pc");
    const onlyPhone = rows.filter((r) => r.verdict === "only_phone");
    const different = rows.filter((r) => r.verdict === "newer_pc" || r.verdict === "newer_phone");
    const sameCount = rows.filter((r) => r.verdict === "same").length;

    const apply = q<HTMLButtonElement>(root, "#sy-apply");
    const scan = q<HTMLButtonElement>(root, "#sy-scan");
    apply.textContent = `Sync selected (${selected.size})`;
    apply.disabled = selected.size === 0 || !linked() || applying;
    apply.title = !linked()
      ? NO_LINK
      : selected.size === 0
        ? "Tick the files to copy"
        : "Copy the selected files in their arrow direction";
    scan.disabled = scanning || !linked();
    scan.title = linked() ? "Compare received/ + shared/ with the phone again" : NO_LINK;
    q(root, "#sy-count").textContent = scanning ? "Scanning…" : `${selected.size} selected`;
    q(root, "#sy-last").textContent = scannedAt
      ? `Last scanned ${fmtClock(scannedAt)}${note ? ` · ${note}` : ""}`
      : scanning
        ? "Scanning…"
        : "Not scanned yet";
    q(root, "#sy-device").textContent = linked()
      ? store.status?.device || store.v2?.connectedName || "connected"
      : `no control link · ${phoneAddr(store.status)}`;

    const stateHost = q(root, "#sy-state");
    if (error) {
      stateHost.innerHTML = `<div class="state error">
        <div class="title">Sync could not compare</div>
        <div class="body">${esc(error)}</div>
        <div class="actions"><button class="link" id="sy-retry">Try again</button></div>
      </div>`;
      stateHost.querySelector("#sy-retry")?.addEventListener("click", () => void scanNow());
    } else if (!linked()) {
      stateHost.innerHTML = `<div class="state">
        <div class="title">No phone connected</div>
        <div class="body">${esc(NO_LINK)}. Open the phone app on the same hotspot — then Sync can compare both sides.</div>
        <div class="actions"><button class="link" id="sy-addr">Show the connection address</button></div>
      </div>`;
      stateHost.querySelector("#sy-addr")?.addEventListener("click", () => ctx.toast(`Phone connects to ${phoneAddr(store.status)}`));
    } else if (!rows.length) {
      stateHost.innerHTML = scanning
        ? `<div class="state"><div class="title">Comparing…</div><div class="body">Reading the phone's file list. Nothing is moved until you press Sync selected.</div></div>`
        : `<div class="state"><div class="title">Nothing to compare yet</div><div class="body">Run a scan to see which files live only on this PC, only on the phone, or differ.</div><div class="actions"><button class="link" id="sy-first">Scan now</button></div></div>`;
      stateHost.querySelector("#sy-first")?.addEventListener("click", () => void scanNow());
    } else {
      stateHost.innerHTML = "";
    }

    q(root, "#sy-pc").innerHTML = `
      ${group(`Only on PC · ${onlyPc.length}`, onlyPc.map((r) => item(r, "push", fmtBytes(r.pc?.size ?? 0), false)).join(""))}
      ${group(
        `Different version · ${different.length}`,
        different
          .map((r) => {
            const dir = dirOf(r);
            const meta = `${fmtBytes(r.pc?.size ?? 0)} vs ${fmtBytes(r.phone?.size ?? 0)}`;
            return item(r, dir, meta, true);
          })
          .join(""),
      )}
      <div class="inboth">${icon("chevronRight", 12)} In both · ${sameCount} file${sameCount === 1 ? "" : "s"} (identical)</div>`;

    q(root, "#sy-phone").innerHTML = `
      ${group(`Only on phone · ${onlyPhone.length}`, onlyPhone.map((r) => item(r, "pull", fmtBytes(r.phone?.size ?? 0), false)).join(""))}
      ${group(
        `Different version · ${different.length}`,
        different
          .map((r) => {
            const dir = dirOf(r) === "push" ? "pull" : "push";
            const meta = `${fmtBytes(r.phone?.size ?? 0)} vs ${fmtBytes(r.pc?.size ?? 0)}`;
            return item(r, dir, meta, true);
          })
          .join(""),
      )}
      <div class="inboth">${icon("chevronRight", 12)} In both · ${sameCount} file${sameCount === 1 ? "" : "s"} (identical)</div>`;

    bind();
  };

  const bind = (): void => {
    qa<HTMLInputElement>(root, ".sync-item input.box").forEach((el) => {
      el.addEventListener("change", () => {
        const name = el.dataset.name ?? "";
        if (el.checked) selected.add(name);
        else selected.delete(name);
        render();
      });
    });
    qa<HTMLElement>(root, ".sync-item .arrow.dir").forEach((el) => {
      el.addEventListener("click", () => {
        const name = el.closest(".sync-item")?.querySelector("span.nowrap")?.textContent ?? "";
        const row = rows.find((r) => r.name === name);
        if (!row || row.verdict === "only_pc" || row.verdict === "only_phone" || row.verdict === "same") {
          ctx.toast("Only files that exist on both sides can change direction");
          return;
        }
        overrides.set(row.name, dirOf(row) === "push" ? "pull" : "push");
        render();
      });
    });
  };

  const applyResult = (res: SyncResult): void => {
    if (res.runId && runId && res.runId !== runId && rows.length) {
      // A stale result from an earlier run: ignore it.
      return;
    }
    rows = res.rows ?? [];
    scannedAt = res.scannedAt || 0;
    note = res.note ?? "";
    error = res.error ?? "";
    if (error) {
      selected.clear();
      overrides.clear();
    } else {
      // Drop selections whose file no longer needs attention.
      const names = new Set(rows.filter((r) => r.action !== "none").map((r) => r.name));
      for (const n of [...selected]) if (!names.has(n)) selected.delete(n);
    }
    scanning = false;
    render();
  };

  async function scanNow(): Promise<void> {
    if (!linked()) {
      ctx.toast(NO_LINK);
      return;
    }
    error = "";
    scanning = true;
    render();
    try {
      runId = await api.startSyncScan();
    } catch (err) {
      error = String(err);
      scanning = false;
      render();
    }
  }

  root.innerHTML = `
    <section class="page">
      <header class="page-head">
        <div>
          <h1 class="screen">Sync</h1>
          <p class="screen-sub">Compare files before transferring</p>
        </div>
        <div class="head-links">
          <button class="link faint" id="sy-selectall">Select all</button>
          <button class="primary-action" id="sy-apply">Sync selected (0)</button>
        </div>
      </header>
      <div class="sync-head">
        <span class="tiny muted" id="sy-last">Not scanned yet</span>
        <span class="tiny muted"><button class="link faint" id="sy-scan">Scan again</button> · <span id="sy-count"></span></span>
      </div>
      <div class="sync-cols">
        <div class="sync-col">
          <div class="col-head"><b>This PC</b><span>received/ + shared/</span></div>
          <div id="sy-pc"></div>
        </div>
        <div class="sync-col">
          <div class="col-head"><b>Phone</b><span id="sy-device">no control link</span></div>
          <div id="sy-phone"></div>
        </div>
      </div>
      <div id="sy-state"></div>
      <p class="hint">Nothing is ever deleted in either direction. Same = equal size and mtime within 2 s, or equal sha256.</p>
    </section>`;

  q(root, "#sy-scan").addEventListener("click", () => void scanNow());
  q(root, "#sy-selectall").addEventListener("click", () => {
    const names = rows.filter((r) => r.action !== "none").map((r) => r.name);
    const allSelected = names.length > 0 && names.every((n) => selected.has(n));
    selected.clear();
    if (!allSelected) names.forEach((n) => selected.add(n));
    render();
  });
  q(root, "#sy-apply").addEventListener("click", async () => {
    if (!selected.size) return;
    if (!linked()) {
      ctx.toast(NO_LINK);
      return;
    }
    applying = true;
    progress.clear();
    render();
    try {
      runId = await api.applySync([...selected]);
      ctx.toast(`Copying ${selected.size} file${selected.size === 1 ? "" : "s"}…`);
    } catch (err) {
      ctx.toast(String(err));
      applying = false;
      render();
    }
  });

  const offResult = onEvent("sync-result", (res: SyncResult) => {
    if (!res) return;
    applying = false;
    progress.clear();
    applyResult(res);
  });
  const offProgress = onEvent("sync-progress", (p: SyncProgress) => {
    if (!p || !p.name) return;
    progress.set(p.name.toLowerCase(), p);
    if (p.state === "done" || p.state === "error") {
      if (p.state === "done") selected.delete(p.name);
    }
    render();
  });
  const offLink = listen(() => render());

  render();
  void scanNow();

  return () => {
    offResult();
    offProgress();
    offLink();
  };
}
