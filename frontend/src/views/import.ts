// Import view: search user storage, tick hits, or browse via OS dialogs.
// Selected paths are REGISTERED in SQLite — never copied.
import { api, type Ctx, type SearchHit } from "../api";

export function mountImport(root: HTMLElement, ctx: Ctx): () => void {
  root.innerHTML = `
    <section class="page">
      <header class="page-head">
        <div>
          <h2>Import library</h2>
          <p class="muted">Search documents and drives (system files and AppData are skipped).
          Selected items are recorded in the catalog — they are not copied.</p>
        </div>
      </header>
      <div class="card">
        <div class="searchbar">
          <input id="i-q" class="input grow" placeholder="File or folder name…" />
          <button id="i-go" class="btn primary">Search</button>
          <button id="i-files" class="btn ghost">Browse files</button>
          <button id="i-folder" class="btn ghost">Browse folder</button>
        </div>
        <div id="i-status" class="muted small pad-top">Type a name and hit Search.</div>
      </div>
      <div class="card grow-card">
        <div class="row between">
          <b>Results</b>
          <div class="row">
            <button id="i-all" class="btn ghost sm">Select all</button>
            <button id="i-none" class="btn ghost sm">Clear</button>
            <button id="i-import" class="btn primary sm">Import selected</button>
          </div>
        </div>
        <div id="i-list" class="results"></div>
      </div>
    </section>`;

  const $ = (id: string) => root.querySelector<HTMLElement>(`#${id}`)!;
  const list = $("i-list") as HTMLElement;
  const status = $("i-status") as HTMLElement;
  let hits: SearchHit[] = [];
  let busy = false;

  const paint = () => {
    list.innerHTML = "";
    if (hits.length === 0) {
      list.innerHTML = `<div class="empty">No results yet.</div>`;
      return;
    }
    for (const h of hits) {
      const row = document.createElement("label");
      row.className = "hit";
      row.innerHTML = `
        <input type="checkbox" data-path="${escapeAttr(h.path)}" />
        <div class="hit-main">
          <div class="hit-name">${escapeHtml(h.name)}</div>
          <div class="hit-path muted small">${escapeHtml(h.path)}</div>
        </div>`;
      list.appendChild(row);
    }
  };

  const selected = (): string[] =>
    [...list.querySelectorAll<HTMLInputElement>("input[type=checkbox]:checked")].map(
      (c) => c.dataset.path ?? "",
    ).filter(Boolean);

  const doImport = async (paths: string[]) => {
    if (paths.length === 0) {
      status.textContent = "Select at least one result.";
      return;
    }
    busy = true;
    status.textContent = `Importing ${paths.length} path(s)…`;
    try {
      const res = await api.catalogAddPaths(paths);
      status.textContent = `Imported ${res.added} path(s)${res.skipped ? ` (${res.skipped} skipped)` : ""}.`;
      ctx.toast(`Imported ${res.added} path(s) into the catalog`);
    } catch (e) {
      status.textContent = `Import failed: ${e}`;
    } finally {
      busy = false;
    }
  };

  const search = async () => {
    const q = ($("i-q") as HTMLInputElement).value.trim();
    if (!q || busy) return;
    busy = true;
    status.textContent = `Searching for “${q}”…`;
    try {
      hits = await api.searchUser(q, 200);
      status.textContent =
        hits.length === 0 ? "No matches in user storage." : `${hits.length} result(s) — tick the ones to import.`;
      paint();
    } catch (e) {
      status.textContent = `Search failed: ${e}`;
    } finally {
      busy = false;
    }
  };

  ($("i-go") as HTMLButtonElement).addEventListener("click", search);
  ($("i-q") as HTMLInputElement).addEventListener("keydown", (e) => {
    if (e.key === "Enter") search();
  });
  ($("i-files") as HTMLButtonElement).addEventListener("click", async () => {
    const paths = await api.pickFiles().catch(() => null);
    if (paths?.length) doImport(paths);
  });
  ($("i-folder") as HTMLButtonElement).addEventListener("click", async () => {
    const folder = await api.pickFolder("Choose a folder to import").catch(() => "");
    if (folder) {
      try {
        const res = await api.catalogAddFolder(folder);
        status.textContent = `Imported ${res.added} file(s) from ${folder}.`;
        ctx.toast(`Imported ${res.added} file(s)`);
      } catch (e) {
        status.textContent = `Import failed: ${e}`;
      }
    }
  });
  ($("i-all") as HTMLButtonElement).addEventListener("click", () => {
    list.querySelectorAll<HTMLInputElement>("input[type=checkbox]").forEach((c) => (c.checked = true));
  });
  ($("i-none") as HTMLButtonElement).addEventListener("click", () => {
    list.querySelectorAll<HTMLInputElement>("input[type=checkbox]").forEach((c) => (c.checked = false));
  });
  ($("i-import") as HTMLButtonElement).addEventListener("click", () => doImport(selected()));

  paint();
  return () => {};
}

function escapeHtml(s: string): string {
  return s.replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]!));
}

function escapeAttr(s: string): string {
  return escapeHtml(s);
}
