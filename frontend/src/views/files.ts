// Files (concept 02): Received / Imported catalog / Shared, hairline rows,
// per-row Reveal + Send (+ Remove for the catalog). Lists render in chunks of
// 100 ("Show more") because received/ already holds ~100 files and grows.
import { api, onEvent, type CatalogEntry, type Ctx, type FileItem } from "../api";
import { emptyState, errorState } from "../components/states";
import { esc, fmtBytes, fmtDateTime, PAGE_SIZE, q, qa } from "../dom";
import { navigate } from "../nav";
import { listen, phoneLinked, store } from "../state";

type Segment = "received" | "catalog" | "shared";
type SortKey = "newest" | "name" | "size";

interface Row {
  name: string;
  path: string;
  size: number;
  mtime: string | number;
  missing?: boolean;
}

export function mountFiles(root: HTMLElement, ctx: Ctx): () => void {
  root.innerHTML = `
    <section class="page">
      <header class="page-head">
        <div>
          <h1 class="screen">Files</h1>
          <p class="screen-sub">Browse files available to this PC</p>
        </div>
      </header>

      <div class="tabrow">
        <div class="tabs" role="tablist">
          <button class="tab" role="tab" data-seg="received">Received</button>
          <button class="tab" role="tab" data-seg="catalog">Imported catalog</button>
          <button class="tab" role="tab" data-seg="shared">Shared</button>
        </div>
        <div class="right" id="f-count">—</div>
      </div>

      <div class="listhead">
        <span class="section-label" id="f-section">Recently received</span>
        <label class="tiny muted">Sort
          <select id="f-sort" class="plain">
            <option value="newest">Newest first</option>
            <option value="name">Name</option>
            <option value="size">Size</option>
          </select>
        </label>
      </div>

      <div id="f-list" class="rows"></div>
      <div id="f-foot"></div>
      <div id="f-state"></div>
    </section>`;

  const $ = <T extends HTMLElement = HTMLElement>(sel: string) => q<T>(root, sel);
  const list = $("f-list");
  const foot = $("f-foot");
  const stateHost = $("f-state");
  const countEl = $("f-count");
  const sectionEl = $("f-section");

  let seg: Segment = "received";
  let sort: SortKey = "newest";
  let rows: Row[] = [];
  let shown = PAGE_SIZE;
  let loadFailed = false;

  const sectionLabel = (s: Segment) =>
    s === "received" ? "Recently received" : s === "catalog" ? "Imported files" : "Shared folder";

  const setTab = (next: Segment): void => {
    seg = next;
    shown = PAGE_SIZE;
    sectionEl.textContent = sectionLabel(next);
    qa<HTMLButtonElement>(root, ".tab[data-seg]").forEach((b) => {
      const on = b.dataset.seg === next;
      b.classList.toggle("on", on);
      b.setAttribute("aria-selected", on ? "true" : "false");
    });
    void load();
  };

  const sortRows = (input: Row[]): Row[] => {
    const copy = input.slice();
    switch (sort) {
      case "name":
        copy.sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: "base" }));
        break;
      case "size":
        copy.sort((a, b) => b.size - a.size);
        break;
      default:
        copy.sort((a, b) => new Date(b.mtime).getTime() - new Date(a.mtime).getTime());
        break;
    }
    return copy;
  };

  const rowEl = (row: Row): HTMLElement => {
    const el = document.createElement("div");
    el.className = "row";
    el.innerHTML = `
      <div class="grow">
        <div class="title">${esc(row.name)}${row.missing ? ' <span class="badge">missing on disk</span>' : ""}</div>
        <div class="path">${esc(row.path)}</div>
        <div class="meta">${esc(fmtBytes(row.size))} · ${esc(fmtDateTime(row.mtime))}</div>
      </div>
      <div class="actions"></div>`;

    const actions = el.querySelector(".actions") as HTMLElement;
    const add = (label: string, fn: () => void, disabledReason?: string) => {
      const b = document.createElement("button");
      b.className = "link";
      b.textContent = label;
      if (disabledReason) {
        b.setAttribute("aria-disabled", "true");
        b.title = disabledReason;
        b.disabled = true;
      } else {
        b.addEventListener("click", fn);
      }
      actions.appendChild(b);
      return b;
    };

    add("Reveal", () => {
      void api.reveal(row.path).then((res) => {
        if (res && res !== "ok") ctx.toast(res);
      });
    });
    // Send needs a protocol-v2 control link; without one it renders disabled
    // with a tooltip explaining exactly what is missing.
    add(
      "Send",
      () => {
        ctx.toast(`Sending ${row.name}…`);
        api
          .sendToPhone(row.path)
          .then(() => ctx.toast(`${row.name} sent to the phone`))
          .catch((err) => ctx.toast(String(err)));
      },
      phoneLinked() ? undefined : "No phone connected — the phone must be on this network with protocol v2 enabled",
    );
    if (seg === "catalog") {
      add("Remove", async () => {
        try {
          await api.catalogRemove(row.path);
          ctx.toast(`Removed ${row.name} from the catalog`);
          await load();
        } catch {
          ctx.toast("Could not remove that entry");
        }
      });
    }
    return el;
  };

  const paint = (): void => {
    list.innerHTML = "";
    stateHost.innerHTML = "";
    foot.innerHTML = "";

    if (loadFailed) {
      list.classList.add("hidden");
      stateHost.appendChild(
        errorState({
          title: "Couldn't load files",
          body: "The data folder may have moved or is unavailable.",
          actions: [
            { label: "Open settings", onClick: () => navigate("settings") },
            { label: "Retry", onClick: () => void load() },
          ],
        }),
      );
      countEl.textContent = "—";
      return;
    }
    list.classList.remove("hidden");

    if (rows.length === 0) {
      countEl.textContent = "0 files";
      const map: Record<Segment, { title: string; body: string; actions: { label: string; onClick: () => void }[] }> = {
        received: {
          title: "No received files yet",
          body: "Files sent from your phone will appear here.",
          actions: [{ label: "Check connection", onClick: () => navigate("settings") }],
        },
        catalog: {
          title: "Nothing imported yet",
          body: "Import files or folders to make them searchable and available to the phone.",
          actions: [{ label: "Go to Import", onClick: () => navigate("import") }],
        },
        shared: {
          title: "The shared folder is empty",
          body: "Drop files into shared/ to serve them to the phone.",
          actions: [
            {
              label: "Open folder",
              onClick: () => {
                const dir = store.status?.shared;
                if (dir) void api.reveal(dir);
              },
            },
          ],
        },
      };
      const s = map[seg];
      stateHost.appendChild(emptyState(s));
      return;
    }

    countEl.textContent = `${rows.length} file${rows.length === 1 ? "" : "s"}`;
    const sorted = sortRows(rows);
    const visible = sorted.slice(0, shown);
    const frag = document.createDocumentFragment();
    visible.forEach((r) => frag.appendChild(rowEl(r)));
    list.appendChild(frag);

    if (sorted.length > visible.length) {
      const remaining = sorted.length - visible.length;
      foot.innerHTML = `<div class="listfoot">
        <span>Showing ${visible.length} of ${sorted.length}</span>
        <button class="link" id="f-more">Show ${Math.min(PAGE_SIZE, remaining)} more</button>
      </div>`;
      q(root, "#f-more").addEventListener("click", () => {
        shown += PAGE_SIZE;
        paint();
      });
    } else if (sorted.length > PAGE_SIZE) {
      foot.innerHTML = `<div class="listfoot"><span>Showing all ${sorted.length}</span><span></span></div>`;
    }
  };

  const load = async (): Promise<void> => {
    loadFailed = false;
    try {
      if (seg === "catalog") {
        const entries: CatalogEntry[] = await api.catalogList();
        rows = entries.map((e) => ({
          name: e.name,
          path: e.path,
          size: e.sizeBytes,
          mtime: e.importedAt,
          missing: e.exists === false,
        }));
      } else {
        const files: FileItem[] = seg === "received" ? await api.listReceived() : await api.listShared();
        rows = files
          .filter((f) => !f.isDir)
          .map((f) => ({ name: f.name, path: f.path, size: f.size, mtime: f.modTime }));
      }
    } catch {
      rows = [];
      loadFailed = true;
    }
    paint();
  };

  qa<HTMLButtonElement>(root, ".tab[data-seg]").forEach((b) => {
    b.addEventListener("click", () => setTab(b.dataset.seg as Segment));
  });
  $<HTMLSelectElement>("#f-sort").addEventListener("change", (ev) => {
    sort = (ev.target as HTMLSelectElement).value as SortKey;
    shown = PAGE_SIZE;
    paint();
  });

  const off = onEvent("catalog-changed", () => void load());
  // The phone link comes and goes: repaint so Send enables/disables live.
  const offLink = listen(() => paint());
  setTab("received");

  return () => {
    off();
    offLink();
  };
}
