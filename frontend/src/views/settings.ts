// Settings (concept 09), with the data-folder control the concept dropped
// (it is the only way to provision/relocate the app) and the REAL database
// path (Data/wayerpc.db, not the concept's catalog.db).
import { api, type Ctx, type DeviceRow, type StorageInfo, type V2State } from "../api";
import { q } from "../dom";
import { navigate } from "../nav";
import { phoneAddr, refreshStatus, store, listen } from "../state";

export function mountSettings(root: HTMLElement, ctx: Ctx): () => void {
  root.innerHTML = `
    <section class="page">
      <header class="page-head">
        <div>
          <h1 class="screen">Settings</h1>
          <p class="screen-sub">WayerPC preferences and connection details</p>
        </div>
      </header>

      <div class="settings-section">
        <span class="section-label">Data folder</span>
        <div class="deflist">
          <div class="defrow">
            <span class="label">Received files</span>
            <span class="value mono" id="st-received">—</span>
            <button class="link" id="st-open-received">Open folder</button>
          </div>
          <div class="defrow">
            <span class="label">Catalog location</span>
            <span class="value mono" id="st-db">—</span>
            <span></span>
          </div>
          <div class="defrow">
            <span class="label">Data folder</span>
            <span class="value mono" id="st-root">—</span>
            <button class="link" id="st-choose">Choose folder…</button>
          </div>
          <div class="defrow">
            <span class="label">Disk usage</span>
            <span class="value" id="st-usage">—</span>
            <span></span>
          </div>
        </div>
      </div>

      <div class="settings-section">
        <span class="section-label">Server</span>
        <div class="deflist">
          <div class="defrow">
            <span class="label">Status</span>
            <span class="value" id="st-server"><span class="dot amber"></span> <span id="st-server-text">—</span></span>
            <button class="link" id="st-toggle">Stop server</button>
          </div>
          <div class="defrow">
            <span class="label">Phone connection</span>
            <span class="value">Hotspot mode · <span class="mono" id="st-addr">—</span></span>
            <button class="link" id="st-help">Connection help</button>
          </div>
        </div>
      </div>

      <div class="settings-section">
        <span class="section-label">Match rules</span>
        <div class="deflist">
          <div class="defrow">
            <span class="label">Search ranking</span>
            <span class="value" id="st-ranking">Default weights</span>
            <button class="link" id="st-rules">Edit match rules</button>
          </div>
          <div class="defrow">
            <span class="label">Local file index</span>
            <span class="value" id="st-index">Not built yet</span>
            <button class="link" id="st-index-action">Rebuild index</button>
          </div>
        </div>
      </div>

      <div class="settings-section">
        <span class="section-label">Transfer</span>
        <div class="deflist">
          <div class="defrow">
            <span class="label">Incoming files</span>
            <span class="value">Save to the received folder</span>
            <button class="link faint" id="st-incoming" title="Incoming files always land in received/; move the data folder to change the disk.">Change</button>
          </div>
          <div class="defrow">
            <span class="label">Resume partial transfers</span>
            <span class="value">Protocol v2 · received/.partial (kept 24 h)</span>
            <span></span>
          </div>
        </div>
      </div>

      <div class="settings-section">
        <span class="section-label">Phone protocol</span>
        <div class="deflist">
          <div class="defrow">
            <span class="label">Protocol v2 (two-way link)</span>
            <span class="value" id="st-v2">—</span>
            <button class="link" id="st-v2-toggle">Disable</button>
          </div>
          <div class="defrow">
            <span class="label">Known phones</span>
            <span class="value" id="st-devices">—</span>
            <span></span>
          </div>
        </div>
        <p class="hint">Classic protocol v1 (<span class="mono">/ask</span>, <span class="mono">/upload</span>) always stays on.
          Turning v2 off only stops the new <span class="mono">@</span>-framed link; partially received files stay in
          <span class="mono">received/.partial</span> for 24 hours so transfers can resume.</p>
      </div>

      <div class="settings-section">
        <span class="section-label">About</span>
        <div class="deflist">
          <div class="defrow">
            <span class="label" id="st-app">WayerPC</span>
            <span class="value" id="st-version">Version —</span>
            <span class="tiny faint" id="st-proto"></span>
          </div>
        </div>
      </div>
    </section>`;

  const $ = <T extends HTMLElement = HTMLElement>(sel: string) => q<T>(root, sel);
  let info: StorageInfo | null = null;

  const findFmt = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    const units = ["KB", "MB", "GB", "TB"];
    let v = bytes / 1024;
    let u = 0;
    while (v >= 1024 && u < units.length - 1) {
      v /= 1024;
      u += 1;
    }
    return `${v.toFixed(1)} ${units[u]}`;
  };

  const paintInfo = (): void => {
    if (!info) return;
    $("st-received").textContent = info.received || "—";
    $("st-db").textContent = info.dbPath || "—";
    $("st-root").textContent = info.dataRoot || "not set yet";
    $("st-usage").textContent = info.onboarded ? findFmt(info.diskUsage) : "—";
    $("st-choose").textContent = info.onboarded ? "Choose folder…" : "Choose the data folder…";
  };

  const paintIndex = async (): Promise<void> => {
    try {
      const state = await api.getIndexState();
      const built = state.builtAt ? new Date(state.builtAt * 1000).toLocaleString() : "never";
      const label = state.building
        ? `Building… ${Math.round((state.progress || 0) * 100)}%`
        : state.count > 0
          ? `${state.count.toLocaleString()} files · built ${built}`
          : "Not built yet";
      $("st-index").textContent = state.note ? `${label} · ${state.note}` : label;
    } catch {
      $("st-index").textContent = "Unavailable";
    }
    try {
      const rules = await api.getMatchRules();
      const top = rules.filter((r) => r.enabled && r.weight > 0).sort((a, b) => b.weight - a.weight)[0];
      $("st-ranking").textContent = top ? `Weighted · strongest: ${top.id.replace(/_/g, " ")} (${top.weight})` : "All rules disabled";
    } catch {
      $("st-ranking").textContent = "Default weights";
    }
  };

  const paintV2 = async (): Promise<void> => {
    let v2: V2State | null = null;
    try {
      v2 = await api.getV2State();
    } catch {
      v2 = null;
    }
    const value = $("st-v2");
    const toggle = $("st-v2-toggle") as HTMLButtonElement;
    if (!v2) {
      value.textContent = "Unavailable";
      toggle.disabled = true;
      return;
    }
    const links = v2.controlLinks > 0 ? (v2.connectedName || "phone") + " connected" : "no phone connected";
    value.innerHTML = v2.enabled
      ? `<span class="dot green"></span> Enabled · ${v2.protocolVersion} · ${links}`
      : `<span class="dot faint"></span> Disabled · classic protocol only`;
    toggle.textContent = v2.enabled ? "Disable" : "Enable";
    toggle.disabled = false;
    try {
      const devices = await api.getDevices();
      value.title = "";
      $("st-devices").innerHTML = devices.length
        ? devices
            .map(
              (d: DeviceRow) =>
                `<span class="mono">${d.name || d.did}</span>${d.connected ? ' <span class="dot green"></span>' : ""}`,
            )
            .join(" · ")
        : "None yet";
    } catch {
      $("st-devices").textContent = "Unavailable";
    }
  };

  const paintStatus = (): void => {
    const st = store.status;
    const srv = st?.server;
    const dot = $("st-server").querySelector(".dot")!;
    if (srv?.running) {
      dot.className = "dot green";
      $("st-server-text").textContent = "Listening";
      $("st-toggle").textContent = "Stop server";
    } else {
      dot.className = "dot red";
      $("st-server-text").textContent = "Stopped";
      $("st-toggle").textContent = "Start server";
    }
    $("st-addr").textContent = phoneAddr(st ?? null);
  };

  const load = async (): Promise<void> => {
    try {
      info = await api.getStorageInfo();
    } catch {
      info = null;
    }
    paintInfo();
    paintStatus();
    try {
      const v = await api.getVersion();
      $("st-version").textContent = `Version ${v.version}`;
      $("st-proto").textContent = `protocol v${v.protocolVersion}`;
    } catch {
      /* mock */
    }
    await paintIndex();
    await paintV2();
  };

  $("st-open-received").addEventListener("click", () => {
    if (info?.received) void api.reveal(info.received);
  });

  $("st-choose").addEventListener("click", async () => {
    const folder = await api.pickFolder("Choose the WayerPC data folder").catch(() => "");
    if (!folder) return;
    const hadData = Boolean(info?.onboarded);
    const ok = await ctx.confirm(
      hadData
        ? `Move the WayerPC data to:\n${folder}\n\nExisting files and the catalog move with it.`
        : `Use this folder for WayerPC data?\n${folder}`,
    );
    if (!ok) return;
    try {
      info = await api.setDataRoot(folder);
      paintInfo();
      await refreshStatus();
      ctx.toast("Data folder updated");
    } catch (err) {
      ctx.toast(String(err));
    }
  });

  $("st-toggle").addEventListener("click", async () => {
    if (store.status?.server?.running) await api.stopServer().catch(() => {});
    else await api.startServer().catch(() => {});
    await refreshStatus();
    paintStatus();
  });

  $("st-help").addEventListener("click", () => {
    ctx.toast(`Turn on the phone's hotspot, join it from this PC, then connect to ${phoneAddr(store.status)}`);
  });

  $("st-v2-toggle").addEventListener("click", async () => {
    const wasEnabled = store.status?.protocolV2 ?? true;
    try {
      const next = await api.setProtocolV2(!wasEnabled);
      ctx.toast(next.enabled ? "Protocol v2 enabled" : "Protocol v2 disabled — classic protocol only");
      await refreshStatus();
      await paintV2();
    } catch (err) {
      ctx.toast(String(err));
    }
  });

  $("st-rules").addEventListener("click", () => navigate("find/rules"));
  $("st-index-action").addEventListener("click", async () => {
    try {
      const res = await api.rebuildIndex();
      ctx.toast(res === "rebuilding" ? "Rebuilding the file index in the background" : res);
      window.setTimeout(() => void paintIndex(), 800);
    } catch {
      ctx.toast("Could not start the rebuild");
    }
  });
  $("st-incoming").addEventListener("click", () => {
    ctx.toast("Incoming files always land in received/ — move the data folder to change the disk");
  });

  const off = listen(paintStatus);
  void load();

  return () => off();
}
