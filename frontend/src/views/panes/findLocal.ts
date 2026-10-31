// Find · Search pane (concept 03 + the empty/error states of concept 10).
// Drives StartFind and paints `find-hits` batches as they arrive; the newest
// request wins, older ones are cancelled.
import { api, onEvent, type Ctx, type FindDoneEvent, type FindHit, type FindHitsEvent } from "../../api";
import { emptyState, errorState } from "../../components/states";
import { esc, fmtBytes, fmtDateTime, q } from "../../dom";
import { icon } from "../../icons";
import { navigate } from "../../nav";
import { store } from "../../state";

export function mountFindLocal(root: HTMLElement, ctx: Ctx): () => void {
  root.innerHTML = `
    <div class="findbar" style="margin-top:22px">
      <label class="field">
        ${icon("find", 15)}
        <input id="fd-q" type="search" placeholder="Search this PC" autocomplete="off" spellcheck="false" />
      </label>
      <button id="fd-go" class="link">Search</button>
    </div>
    <div class="listhead">
      <span class="section-label">Results</span>
      <span class="right tiny muted" id="fd-count"></span>
    </div>
    <div id="fd-results" class="rows results"></div>
    <div id="fd-state"></div>
    <div class="linkrow" style="justify-content:flex-start">
      <span class="tiny faint">Not found here?</span>
      <a class="link" href="#/find/phone">Ask phone</a>
    </div>
    <p class="hint" id="fd-hint">System files and AppData are skipped. <a class="link" href="#/find/rules">Adjust match rules</a></p>`;

  const input = q<HTMLInputElement>(root, "#fd-q");
  const results = q(root, "#fd-results");
  const stateHost = q(root, "#fd-state");
  const countEl = q(root, "#fd-count");
  const hint = q(root, "#fd-hint");

  let searchId = "";
  let pending = "";
  let total = 0;
  let lastQuery = "";
  let timer = 0;
  const hits: FindHit[] = [];

  const meter = (score: number) => `
    <div class="score">
      <span class="meter"><i style="width:${Math.round(score * 100)}%"></i></span>
      <span class="num">score ${score.toFixed(2)}</span>
    </div>`;

  const rowHtml = (hit: FindHit) => `
    <div class="row result">
      <div class="grow">
        <div class="title">${esc(hit.name)}</div>
        <div class="path">${esc(hit.path)}</div>
        <div class="meta">${esc(fmtBytes(hit.size))} · Modified ${esc(fmtDateTime(hit.mtime))} · <span class="reason">${esc(hit.reason)}</span></div>
      </div>
      ${meter(hit.score)}
      <div class="actions"></div>
    </div>`;

  const addRows = (batch: FindHit[]): void => {
    const start = hits.length;
    hits.push(...batch);
    const wrap = document.createElement("div");
    wrap.innerHTML = batch.map(rowHtml).join("");
    const children = Array.from(wrap.children) as HTMLElement[];
    children.forEach((rowEl, i) => {
      const hit = hits[start + i];
      const actions = rowEl.querySelector(".actions") as HTMLElement;
      const reveal = document.createElement("button");
      reveal.className = "link";
      reveal.textContent = "Reveal";
      reveal.addEventListener("click", () => void api.reveal(hit.path).then((r) => r !== "ok" && ctx.toast(r)));
      actions.appendChild(reveal);
      results.appendChild(rowEl);
    });
  };

  const showNoResults = (): void => {
    results.innerHTML = "";
    stateHost.innerHTML = "";
    if (!lastQuery) {
      countEl.textContent = "";
      return;
    }
    countEl.textContent = "0 matches";
    stateHost.appendChild(
      emptyState({
        title: "No matches found",
        body: "Try another name or adjust your match rules.",
        actions: [
          { label: "Match rules", onClick: () => navigate("find/rules") },
          { label: "Ask phone", onClick: () => navigate("find/phone") },
        ],
      }),
    );
  };

  const showError = (note: string): void => {
    results.innerHTML = "";
    stateHost.innerHTML = "";
    countEl.textContent = "";
    stateHost.appendChild(
      errorState({
        title: "Search could not complete",
        body: note || "The local file index is unavailable.",
        actions: [
          {
            label: "Rebuild index",
            onClick: async () => {
              await api.rebuildIndex().catch(() => {});
              ctx.toast("Rebuilding the file index in the background");
              runSearch();
            },
          },
          { label: "Retry", onClick: () => runSearch() },
        ],
      }),
    );
  };

  const runSearch = (): void => {
    const query = input.value.trim();
    lastQuery = query;
    hits.length = 0;
    results.innerHTML = "";
    stateHost.innerHTML = "";
    if (pending) {
      void api.cancelFind(pending).catch(() => {});
      pending = "";
    }
    if (!query) {
      countEl.textContent = "";
      return;
    }
    countEl.textContent = "searching…";
    api.startFind(query, { limit: 50 })
      .then((id) => {
        // A newer keystroke may have started another search meanwhile.
        if (lastQuery !== query) {
          void api.cancelFind(id).catch(() => {});
          return;
        }
        pending = id;
      })
      .catch(() => showError("The local file index is unavailable."));
  };

  const offHits = onEvent("find-hits", (payload: FindHitsEvent) => {
    if (!payload || payload.searchId !== pending) return;
    stateHost.innerHTML = "";
    addRows(payload.hits ?? []);
    total = hits.length;
    countEl.textContent = `${total} match${total === 1 ? "" : "es"}`;
  });

  const offDone = onEvent("find-done", (payload: FindDoneEvent) => {
    if (!payload || payload.searchId !== pending) return;
    pending = "";
    total = payload.total ?? hits.length;
    if (total === 0) showNoResults();
    else countEl.textContent = `${total} match${total === 1 ? "" : "es"} · ${payload.tookMs} ms`;
    if (payload.skippedNote) {
      hint.innerHTML = `${esc(payload.skippedNote)} · <a class="link" href="#/find/rules">Adjust match rules</a>`;
    } else {
      hint.innerHTML = `System files and AppData are skipped. <a class="link" href="#/find/rules">Adjust match rules</a>`;
    }
  });

  // Keep the route's counter honest if the index is still building.
  const indexNote = () => {
    const st = store.status;
    if (!st) return;
    api
      .getIndexState()
      .then((state) => {
        if (state.building) countEl.textContent = "index building — partial results";
      })
      .catch(() => {});
  };

  q(root, "#fd-go").addEventListener("click", runSearch);
  input.addEventListener("input", () => {
    window.clearTimeout(timer);
    timer = window.setTimeout(runSearch, 220);
  });
  input.addEventListener("keydown", (ev) => {
    if (ev.key === "Enter") runSearch();
  });

  input.focus();
  indexNote();

  return () => {
    window.clearTimeout(timer);
    offHits();
    offDone();
    if (pending) void api.cancelFind(pending).catch(() => {});
    if (searchId) void api.cancelFind(searchId).catch(() => {});
  };
}
