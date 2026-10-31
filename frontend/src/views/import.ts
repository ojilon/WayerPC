// Import (concept 08): search user storage, tick results, or browse via OS
// dialogs. Paths are registered in the catalog, never copied.
// TODO(phase2): switch the search below to the same local file index that
// powers Find (SearchUser stays as the fallback until then).
import { api, type Ctx, type SearchHit } from "../api";
import { esc, q } from "../dom";
import { icon } from "../icons";
import { refreshStatus } from "../state";

export function mountImport(root: HTMLElement, ctx: Ctx): () => void {
  root.innerHTML = `
    <section class="page">
      <header class="page-head">
        <div>
          <h1 class="screen">Import</h1>
          <p class="screen-sub">Add files to your searchable catalog</p>
        </div>
      </header>

      <div class="findbar">
        <label class="field">
          ${icon("find", 15)}
          <input id="im-q" type="search" placeholder="Search files and folders" autocomplete="off" />
        </label>
        <div class="rowflex" style="gap:18px">
          <button id="im-files" class="link">Browse files</button>
          <button id="im-folder" class="link">Browse folder</button>
        </div>
      </div>

      <div class="listhead">
        <span class="section-label" id="im-section">Suggested files</span>
        <span class="right tiny muted" id="im-count"></span>
      </div>
      <div id="im-rows" class="rows"></div>
      <div id="im-state"></div>

      <p class="hint">Items are catalogued for search. They are not copied or moved.</p>
      <div class="rulefoot">
        <span class="selected-count" id="im-sel">0 selected</span>
        <button class="link" id="im-add" disabled>Add to catalog</button>
      </div>
    </section>`;

  const input = q<HTMLInputElement>(root, "#im-q");
  const rowsHost = q(root, "#im-rows");
  const countEl = q(root, "#im-count");
  const selEl = q(root, "#im-sel");
  const addBtn = q<HTMLButtonElement>(root, "#im-add");

  let hits: SearchHit[] = [];
  const selected = new Set<string>();
  let searching = false;

  const updateSel = (): void => {
    selEl.textContent = `${selected.size} selected`;
    addBtn.disabled = selected.size === 0;
  };

  const paint = (): void => {
    rowsHost.innerHTML = "";
    if (!searching && hits.length === 0) {
      countEl.textContent = "";
      return;
    }
    countEl.textContent = `${hits.length} item${hits.length === 1 ? "" : "s"}`;
    hits.forEach((hit) => {
      const row = document.createElement("div");
      row.className = "row";
      row.innerHTML = `
        <input class="box" type="checkbox" ${selected.has(hit.path) ? "checked" : ""} aria-label="Select ${esc(hit.name)}" />
        <div class="grow">
          <div class="title">${esc(hit.name)}</div>
          <div class="path">${esc(hit.path)}</div>
        </div>
        <div class="actions"></div>`;
      const box = row.querySelector("input.box") as HTMLInputElement;
      box.addEventListener("change", () => {
        if (box.checked) selected.add(hit.path);
        else selected.delete(hit.path);
        updateSel();
      });
      const reveal = document.createElement("button");
      reveal.className = "link";
      reveal.textContent = "Preview";
      reveal.addEventListener("click", () => void api.reveal(hit.path).then((r) => r !== "ok" && ctx.toast(r)));
      (row.querySelector(".actions") as HTMLElement).appendChild(reveal);
      rowsHost.appendChild(row);
    });
    updateSel();
  };

  const search = async (): Promise<void> => {
    const query = input.value.trim();
    if (!query) {
      hits = [];
      searching = false;
      countEl.textContent = "";
      rowsHost.innerHTML = "";
      return;
    }
    searching = true;
    try {
      hits = await api.searchUser(query, 200);
    } catch {
      hits = [];
      ctx.toast("Search failed — try again");
    }
    paint();
  };

  const addPaths = async (paths: string[]): Promise<void> => {
    if (paths.length === 0) return;
    try {
      const res = await api.catalogAddPaths(paths);
      ctx.toast(`Catalogued ${res.added} item(s)${res.skipped ? `, ${res.skipped} skipped` : ""}`);
      selected.clear();
      await refreshStatus();
      await search();
    } catch {
      ctx.toast("Could not add those items");
    }
  };

  let timer = 0;
  input.addEventListener("input", () => {
    window.clearTimeout(timer);
    timer = window.setTimeout(() => void search(), 220);
  });

  q(root, "#im-files").addEventListener("click", async () => {
    const paths = await api.pickFiles().catch(() => null);
    if (paths && paths.length) await addPaths(paths);
  });
  q(root, "#im-folder").addEventListener("click", async () => {
    const folder = await api.pickFolder("Choose a folder to import").catch(() => "");
    if (!folder) return;
    try {
      const res = await api.catalogAddFolder(folder);
      ctx.toast(`Catalogued ${res.added} file(s)${res.skipped ? `, ${res.skipped} skipped` : ""}`);
      await refreshStatus();
      await search();
    } catch {
      ctx.toast("Could not import that folder");
    }
  });
  addBtn.addEventListener("click", () => void addPaths(Array.from(selected)));

  updateSel();
  return () => {
    window.clearTimeout(timer);
  };
}
