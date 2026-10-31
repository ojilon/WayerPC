// Find · Requests pane (concept 06): the log of phone-initiated queries with
// an expandable "bundle sent to phone" detail.
//
// Wired in phase 3: api.getRequestLog() for the initial list, then the
// `request-update` event (via the shared store) keeps it live without polling.
import type { Ctx, RequestLogEntry } from "../../api";
import { esc, fmtBytes, fmtClock, fmtDate, qa } from "../../dom";
import { listen, refreshRequests, store } from "../../state";

function isToday(unixSeconds: number): boolean {
  const d = new Date(unixSeconds * 1000);
  const now = new Date();
  return (
    d.getFullYear() === now.getFullYear() &&
    d.getMonth() === now.getMonth() &&
    d.getDate() === now.getDate()
  );
}

function summary(r: RequestLogEntry): string {
  const count = r.matchCount || 0;
  const plural = `match${count === 1 ? "" : "es"}`;
  switch (r.outcome) {
    case "no-match":
      return "no matches";
    case "aborted":
      return `${count} ${plural} · aborted`;
    case "error":
      return `${count} ${plural} · failed`;
    case "pending":
      return `${count} ${plural} · waiting`;
    default:
      return `${count} ${plural}`;
  }
}

function body(r: RequestLogEntry): string {
  if (!r.bundle || r.bundle.length === 0) {
    return `<div class="req-body hidden"><div class="inner"><div class="sub-label">Bundle sent to phone</div>
      <p class="tiny muted">No bundle was sent for this request.</p></div></div>`;
  }
  return `<div class="req-body hidden"><div class="inner">
    <div class="sub-label">Bundle sent to phone</div>
    <div class="rows">
      ${r.bundle
        .map(
          (b) => `<div class="row" style="min-height:34px">
            <div class="grow"><div class="title" style="font-weight:400">${esc(b.name)}</div></div>
            <span class="tiny muted nowrap">${esc(fmtBytes(b.size))}</span>
            <span class="tiny muted nowrap" style="width:84px;text-align:right">${esc(fmtDate(b.mtime))}</span>
            <span style="width:14px"></span>
          </div>`,
        )
        .join("")}
    </div>
    <p class="tiny" style="margin:12px 0 0;color:var(--muted)">
      ${r.picked ? `Phone selected <b>${esc(r.pickedName ?? String(r.picked))}</b>` : "Phone did not choose a file"}
    </p>
  </div></div>`;
}

export function mountRequests(root: HTMLElement, _ctx: Ctx): () => void {
  const rowsHost = () => root.querySelector<HTMLElement>("#rq-rows");
  const countHost = () => root.querySelector<HTMLElement>("#rq-count");

  function render(): void {
    // Newest first, like the concept.
    const entries = [...store.requests].sort((a, b) => b.ts - a.ts);
    const today = entries.filter((r) => isToday(r.ts)).length;
    const host = rowsHost();
    const count = countHost();
    if (!host || !count) return;
    count.textContent = `TODAY · ${today} REQUEST${today === 1 ? "" : "S"}`;
    if (entries.length === 0) {
      host.innerHTML = `<div class="state">
        <div class="title">No requests yet</div>
        <div class="body">When the phone searches this PC, every query lands here with the numbered list it received.</div>
        <div class="actions"><a class="link" href="#/find">Ask the phone for a file</a></div>
      </div>`;
      return;
    }
    host.innerHTML = entries
      .map(
        (r) => `<div class="req" data-req="${esc(r.id)}">
          <div class="req-head">
            <span class="time">${esc(fmtClock(r.ts))}</span>
            <span class="dev"><span class="dot green"></span>${esc(r.device || "phone")}</span>
            <span>Query &ldquo;${esc(r.query)}&rdquo; · ${esc(summary(r))}</span>
            <button class="link faint" data-toggle="${esc(r.id)}">+ details</button>
          </div>
          ${body(r)}
        </div>`,
      )
      .join("");

    qa<HTMLButtonElement>(host, "[data-toggle]").forEach((btn) => {
      btn.addEventListener("click", () => {
        const req = host.querySelector<HTMLElement>(`[data-req="${btn.dataset.toggle}"]`);
        const detail = req?.querySelector<HTMLElement>(".req-body");
        if (!detail) return;
        const open = !detail.classList.contains("hidden");
        detail.classList.toggle("hidden", open);
        btn.textContent = open ? "+ details" : "− expanded";
      });
    });
  }

  root.innerHTML = `
    <div class="listhead">
      <span class="section-label">Request log</span>
      <span class="tiny muted" id="rq-count">—</span>
    </div>
    <div class="rows" id="rq-rows"></div>
    <p class="hint">Numbered result lists are sent to the phone. The returned number identifies the chosen file.</p>`;

  const unlisten = listen(render);
  render();
  void refreshRequests();

  return () => unlisten();
}
