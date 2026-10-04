// Files view: Received ⇄ Imported catalog ⇄ Shared segments with sorting,
// per-row Reveal, and catalog Remove. Refreshes on "catalog-changed".
import { api, fmtDate, fmtMB, onEvent, type CatalogEntry, type Ctx, type FileItem } from "../api";

type Segment = "received" | "catalog" | "shared";
type SortKey = "name" | "size" | "date";

export function mountLibrary(root: HTMLElement, ctx: Ctx): () => void {
  root.innerHTML = `
    <section class="page">
      <header class="page-head">
        <div>
          <h2>Files</h2>
          <p class="muted">Uploads from the phone plus everything the catalog knows about.</p>
        </div>
        <button id="f-refresh" class="btn ghost">Refresh</button>
      </header>
      <div class="card">
        <div class="row between wrap">
          <div class="seg" role="tablist">
            <button data-seg="received" class="on">Received</button>
            <button data-seg="catalog">Imported catalog</button>
            <button data-seg="shared">Shared</button>
          </div>
          <label class="muted small">Sort
            <select id="f-sort" class="input sm">
              <option value="date">Newest</option>
              <option value="name">Name</option>
              <option value="size">Size</option>
            </select>
          </label>
        </div>
        <div id="f-hint" class="muted small pad-top"></div>
      </div>
      <div class="card grow-card"><div id="f-list" class="results"></div></div>
    </section>`;

  const $ = (id: string) => root.querySelector<HTMLElement>(`#${id}`)!;
  const list = $("f-list") as HTMLElement;
  const hint = $("f-hint") as HTMLElement;
  let seg: Segment = "received";
  let sort: SortKey = "date";

  const row = (title: string, sub: string, extra: string, missing: boolean) => {
    const el = document.createElement("div");
    el.className = "file";
    el.innerHTML = `
      <div class="file-main">
        <div class="file-name">${escapeHtml(title)}${missing ? ' <span class="badge">missing on disk</span>' : ""}</div>
        <div class="file-path muted small">${escapeHtml(sub)}</div>
        ${extra ? `<div class="muted small">${escapeHtml(extra)}</div>` : ""}
      </div>
      <div class="row file-actions"></div>`;
    return el;
  };

  const action = (parent: HTMLElement, label: string, fn: () => void, danger = false) => {
    const b = document.createElement("button");
    b.className = `btn ghost sm${danger ? " danger" : ""}`;
    b.textContent = label;
    b.addEventListener("click", (e) => {
      e.stopPropagation();
      fn();
    });
    parent.appendChild(b);
  };

  const paintCatalog = async () => {
    const rows = await api.catalogList().catch(() => [] as CatalogEntry[]);
    const sorted = sortRows(
      rows.map((r) => ({ title: r.name, sub: r.path, size: r.sizeBytes, date: r.importedAt, ref: r })),
      sort,
    );
    hint.textContent = `${rows.length} imported path(s) — used by /ask on the phone`;
    list.innerHTML = rows.length === 0 ? `<div class="empty">Catalog is empty. Use Import to add files.</div>` : "";
    for (const r of sorted) {
      const el = row(r.title, r.sub, `${fmtMB(r.size)} · ${fmtDate(r.date)}`, r.ref.exists === false);
      const acts = el.querySelector(".file-actions") as HTMLElement;
      action(acts, "Reveal", () => api.reveal(r.ref.path).then(doneToast).catch(showErr));
      action(
        acts,
        "Remove",
        async () => {
          if (!(await ctx.confirm(`Remove ${r.ref.name} from the catalog? (file stays on disk)`))) return;
          await api.catalogRemove(r.ref.path).catch(showErr);
          paint();
        },
        true,
      );
      list.appendChild(el);
    }
  };

  const paintFiles = async (which: "received" | "shared") => {
    const files = await (which === "received" ? api.listReceived() : api.listShared()).catch(
      () => [] as FileItem[],
    );
    const sorted = sortRows(
      files
        .filter((f) => !f.isDir)
        .map((f) => ({ title: f.name, sub: f.path, size: f.size, date: f.modTime, ref: f })),
      sort,
    );
    hint.textContent = `${sorted.length} file(s) in ${which}/`;
    list.innerHTML = sorted.length === 0 ? `<div class="empty">Nothing here yet.</div>` : "";
    for (const r of sorted) {
      const el = row(r.title, r.sub, `${fmtMB(r.size)} · ${fmtDate(r.date)}`, false);
      const acts = el.querySelector(".file-actions") as HTMLElement;
      action(acts, "Reveal", () => api.reveal(r.ref.path).then(doneToast).catch(showErr));
      list.appendChild(el);
    }
  };

  const doneToast = (msg: string) => {
    if (msg !== "ok" && !msg.endsWith("(mock)")) ctx.toast(msg);
  };
  const showErr = (e: unknown) => ctx.toast(String(e));

  const paint = () => {
    list.innerHTML = `<div class="empty">Loading…</div>`;
    if (seg === "catalog") void paintCatalog();
    else void paintFiles(seg);
  };

  root.querySelectorAll<HTMLButtonElement>(".seg button").forEach((b) => {
    b.addEventListener("click", () => {
      root.querySelectorAll(".seg button").forEach((x) => x.classList.remove("on"));
      b.classList.add("on");
      seg = b.dataset.seg as Segment;
      paint();
    });
  });
  ($("f-sort") as HTMLSelectElement).addEventListener("change", (e) => {
    sort = (e.target as HTMLSelectElement).value as SortKey;
    paint();
  });
  ($("f-refresh") as HTMLButtonElement).addEventListener("click", paint);

  const off = onEvent("catalog-changed", paint);
  paint();
  return () => off();
}

function sortRows<T extends { title: string; size: number; date: string }>(rows: T[], sort: SortKey): T[] {
  const by = {
    name: (a: T, b: T) => a.title.localeCompare(b.title),
    size: (a: T, b: T) => b.size - a.size,
    date: (a: T, b: T) => b.date.localeCompare(a.date),
  }[sort];
  return [...rows].sort(by);
}

function escapeHtml(s: string): string {
  return String(s).replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]!));
}
