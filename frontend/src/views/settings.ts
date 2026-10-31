// Settings view: version, storage locations (choose/relocate data root),
// server address, and the first-run onboarding banner.
import { api, fmtMB, type Ctx, type StorageInfo, type VersionInfo } from "../api";

export function mountSettings(root: HTMLElement, ctx: Ctx): () => void {
  root.innerHTML = `
    <section class="page">
      <header class="page-head">
        <div>
          <h2>Settings</h2>
          <p class="muted">Where WayerPC lives, what it serves, which build this is.</p>
        </div>
      </header>
      <div id="s-onboard" class="banner warn hidden"></div>
      <div class="cards">
        <div class="card">
          <b>Data folder</b>
          <p class="muted small">Holds <code>shared/</code>, <code>received/</code> and the SQLite catalog.
          Pick any drive or subfolder — e.g. <code>D:\\projects\\</code>.</p>
          <div id="s-root" class="pathelt">…</div>
          <div class="row pad-top">
            <button id="s-choose" class="btn primary sm">Choose folder…</button>
            <button id="s-open" class="btn ghost sm">Open data folder</button>
          </div>
          <div id="s-usage" class="muted small pad-top"></div>
        </div>
        <div class="card">
          <b>Server</b>
          <div id="s-addr" class="pathelt">…</div>
          <p class="muted small">The phone app connects to this address over the hotspot.
          Host/port come from the saved config.</p>
        </div>
        <div class="card">
          <b>About</b>
          <div id="s-ver" class="verline">…</div>
          <p class="muted small" id="s-desc"></p>
        </div>
      </div>
    </section>`;

  const $ = (id: string) => root.querySelector<HTMLElement>(`#${id}`)!;

  const paint = async () => {
    const [info, ver] = await Promise.all([
      api.getStorageInfo().catch(() => null as StorageInfo | null),
      api.getVersion().catch(() => null as VersionInfo | null),
    ]);
    if (info) {
      ($("s-root") as HTMLElement).textContent = info.dataRoot || "(not chosen yet)";
      ($("s-usage") as HTMLElement).textContent = info.onboarded
        ? `${fmtMB(info.diskUsage)} on disk · catalog at ${info.dbPath}`
        : "No data folder yet.";
      const ob = $("s-onboard") as HTMLElement;
      if (!info.onboarded || !info.dataRoot) {
        ob.classList.remove("hidden");
        ob.textContent =
          "Welcome — choose a folder for WayerPC data (drive root or any subfolder works).";
      } else {
        ob.classList.add("hidden");
      }
    }
    if (ver) {
      ($("s-ver") as HTMLElement).textContent = `${ver.appName} v${ver.version} (protocol ${ver.protocolVersion})`;
      ($("s-desc") as HTMLElement).textContent = ver.description;
    }
    const st = await api.getStatus().catch(() => null);
    if (st) ($("s-addr") as HTMLElement).textContent = `${st.server?.addr ?? "—"} (${st.server?.running ? "online" : "offline"})`;
  };

  ($("s-choose") as HTMLButtonElement).addEventListener("click", async () => {
    const folder = await api.pickFolder("Choose where WayerPC keeps its data").catch(() => "");
    if (!folder) return;
    if (!(await ctx.confirm(`Use ${folder} for WayerPC data? Existing data moves with it.`))) return;
    try {
      await api.setDataRoot(folder);
      ctx.toast("Data folder updated");
      paint();
    } catch (e) {
      ctx.toast(`Could not move data: ${e}`);
    }
  });
  ($("s-open") as HTMLButtonElement).addEventListener("click", async () => {
    const info = await api.getStorageInfo().catch(() => null);
    if (info?.dataRoot) api.reveal(info.dataRoot).catch((e) => ctx.toast(String(e)));
  });

  paint();
  return () => {};
}
