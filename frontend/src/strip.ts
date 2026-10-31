// Global transfer strip (50px, bottom): progress line, filename, speed,
// percent, Cancel, "+N more" when several transfers run at once.
import { api } from "./api";
import { esc, fmtRate, pct } from "./dom";
import { listen, store } from "./state";

let host: HTMLElement | null = null;

export function mountTransferStrip(el: HTMLElement): void {
  host = el;
  el.className = "transfer-strip hidden";
  el.addEventListener("click", (ev) => {
    const target = ev.target as HTMLElement;
    if (target.dataset.cancel) {
      void api.cancelTransfer(target.dataset.cancel);
    }
  });
  listen(paint);
  paint();
}

function paint(): void {
  if (!host) return;
  const active = Array.from(store.transfers.values()).filter((t) => t.state === "active");
  if (active.length === 0) {
    host.classList.add("hidden");
    return;
  }
  const first = active[active.length - 1];
  const more = active.length - 1;
  host.classList.remove("hidden");
  host.innerHTML = `
    <span class="label">Transfer</span>
    <span class="bar"><i style="width:${pct(first.bytes, first.total)}%"></i></span>
    <span class="name">${esc(first.name)}</span>
    <span class="num">${esc(fmtRate(first.bps))}</span>
    <span class="num">· ${pct(first.bytes, first.total)}%</span>
    ${more > 0 ? `<span class="more">+${more} more</span>` : ""}
    <span class="spacer"></span>
    <span class="num faint">${first.dir === "in" ? "receiving" : "sending"}</span>
    <button class="link" data-cancel="${esc(first.id)}" title="Abort this transfer">Cancel</button>`;
}
