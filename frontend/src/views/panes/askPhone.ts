// Find · Phone pane (concept 05). Real three-step flow:
//   1. Local check   — what this PC already has (no network needed)
//   2. Phone response — api.askPhone → numbered match list
//   3. Choose file    — api.pickAskMatch (the phone pushes the bytes)
//
// Earlier steps collapse to one line, the current step is expanded. The pane
// also listens to `ask-update`, so a Send started elsewhere (Files → Send)
// shows up here, and it disables the phone actions with a tooltip when no
// protocol-v2 control link exists.
import { api, type Ctx, type PhoneItem } from "../../api";
import { esc, fmtBytes, fmtDate, q } from "../../dom";
import { listen, phoneLinked, refreshRequests, refreshV2 } from "../../state";

type Stage = "local" | "querying" | "choose" | "done" | "none" | "error";

const LINK_TITLE = "Needs a connected phone (protocol v2 control link)";

interface LocalHit {
  name: string;
  path: string;
}

export function mountAskPhone(root: HTMLElement, ctx: Ctx): () => void {
  let stage: Stage = "local";
  let query = "";
  let requestId = "";
  let matches: PhoneItem[] = [];
  let local: LocalHit[] = [];
  let chosen: { n: number; name: string; path: string } | null = null;
  let errMsg = "";
  let busy = false;

  root.innerHTML = `
    <div id="ap-head"></div>
    <div id="ap-query" class="findbar" style="margin-top:12px">
      <input id="ap-input" class="box" type="text" placeholder="What should we look for on the phone?" autocomplete="off" spellcheck="false" />
      <button class="primary-action" id="ap-go">Search phone</button>
    </div>
    <div id="ap-body"></div>
    <div class="linkrow" style="justify-content:flex-start">
      <a class="link" href="#/find/requests">View request log</a>
    </div>`;

  const input = q<HTMLInputElement>(root, "#ap-input");
  const go = q<HTMLButtonElement>(root, "#ap-go");

  const linked = () => phoneLinked();
  const gate = () => {
    const off = !linked() || busy;
    go.setAttribute("aria-disabled", off ? "true" : "false");
    go.title = !linked() ? LINK_TITLE : "";
    go.disabled = off;
    input.disabled = busy;
  };

  const stepClass = (i: 1 | 2 | 3): string => {
    const order: Record<Stage, 1 | 2 | 3> = { local: 1, querying: 2, choose: 3, none: 3, done: 3, error: 1 };
    const current = order[stage];
    if (i === current) return "on";
    return i < current ? "done" : "";
  };

  const head = (): string => {
    const chip = requestId ? `<span class="tiny muted mono">REQUEST ${esc(requestId)}</span>` : "";
    return `<div class="listhead" style="padding-top:18px">
      <div class="stepper">
        <span class="step ${stepClass(1)}"><span class="n">1</span> Local check</span>
        <span class="step-dash"></span>
        <span class="step ${stepClass(2)}"><span class="n">2</span> Phone response</span>
        <span class="step-dash"></span>
        <span class="step ${stepClass(3)}"><span class="n">3</span> Choose file</span>
      </div>
      ${chip}
    </div>
    ${query ? `<p class="screen-sub" style="margin:4px 0 0">Search for &ldquo;${esc(query)}&rdquo; on your phone</p>` : `<p class="screen-sub" style="margin:4px 0 0">Type what you need — your phone searches its own storage.</p>`}`;
  };

  const localBlock = (expanded: boolean): string => {
    const rows = local
      .map(
        (m) => `<div class="row">
          <div class="grow"><div class="title">${esc(m.name)}</div><div class="path">${esc(m.path)}</div></div>
          <div class="actions"><button class="link" data-reveal="${esc(m.path)}">Reveal</button></div>
        </div>`,
      )
      .join("");
    if (!expanded) {
      return `<div class="step-block collapsed">a) Local check · ${local.length} match${local.length === 1 ? "" : "es"} already on this PC</div>`;
    }
    return `<div class="step-block">
      <h3>a) Local check</h3>
      <div class="sub">${local.length} match${local.length === 1 ? "" : "es"} already on this PC${
        local.length ? " — open one of these and skip the phone" : ""
      }</div>
      ${rows ? `<div class="rows">${rows}</div>` : ""}
    </div>`;
  };

  const phoneBlock = (): string => {
    if (stage === "querying") {
      return `<div class="step-block">
        <h3>b) Waiting for phone response</h3>
        <div class="rowflex gap-sm" style="margin-top:8px"><span class="dot amber"></span>
        <span class="tiny muted">Request sent · waiting for the phone to answer</span></div>
      </div>`;
    }
    if (stage === "local") return "";
    const label =
      stage === "none"
        ? "b) Phone response · nothing found"
        : stage === "error"
          ? "b) Phone response · failed"
          : `b) Phone response · ${matches.length} file${matches.length === 1 ? "" : "s"} found`;
    if (stage === "choose" || stage === "done") {
      return `<div class="step-block collapsed">${label}</div>`;
    }
    return `<div class="step-block"><h3>${esc(label.replace("b) ", ""))}</h3>
      ${errMsg ? `<div class="sub" style="color:var(--red)">${esc(errMsg)}</div>` : `<div class="sub">The phone has no file matching this search.</div>`}
    </div>`;
  };

  const chooseBlock = (): string => {
    if (stage !== "choose") return "";
    return `<div class="step-block">
      <h3>c) Pick a result from your phone · ${matches.length} file${matches.length === 1 ? "" : "s"} found</h3>
      <div class="rows">
        ${matches
          .map(
            (m) => `<div class="row">
              <span class="mono faint" style="width:18px">${m.n}</span>
              <div class="grow"><div class="title">${esc(m.name)}</div></div>
              <span class="tiny muted nowrap">${esc(fmtBytes(m.size))}</span>
              <span class="tiny muted nowrap" style="width:84px;text-align:right">${esc(fmtDate(m.mtime))}</span>
              <div class="actions"><button class="link" data-pick="${m.n}">Send</button></div>
            </div>`,
          )
          .join("")}
      </div>
      <div class="rulefoot" style="padding-top:14px">
        <label class="tiny muted">Type a number or Abort
          <input id="ap-n" class="mono" style="width:56px;background:transparent;border:0;border-bottom:1px solid var(--border);outline:none;text-align:center" value="1" />
        </label>
        <button class="link" id="ap-send">Send selection</button>
        <button class="link faint" id="ap-abort">Abort</button>
      </div>
    </div>`;
  };

  const doneBlock = (): string => {
    if (stage !== "done" || !chosen) return "";
    return `<div class="step-block">
      <h3>c) Received · ${esc(chosen.name)}</h3>
      <div class="sub">Saved to the received folder. Request ${esc(requestId)} closed.</div>
      <div class="rows"><div class="row">
        <div class="grow"><div class="title">${esc(chosen.name)}</div><div class="path">${esc(chosen.path)}</div></div>
        <div class="actions">
          <button class="link" data-reveal="${esc(chosen.path)}">Reveal</button>
          <button class="link" id="ap-again">Search again</button>
        </div>
      </div></div>
    </div>`;
  };

  const render = (): void => {
    q(root, "#ap-head").innerHTML = head();
    q(root, "#ap-body").innerHTML = `
      ${localBlock(stage === "local")}
      ${phoneBlock()}
      ${chooseBlock()}
      ${doneBlock()}
      ${linked() ? "" : `<p class="hint">${esc(LINK_TITLE)}. Open the phone app on the same hotspot, then search again.</p>`}
      <div id="ap-state"></div>`;
    gate();
    wire();
  };

  const runLocalCheck = async (text: string): Promise<void> => {
    local = [];
    try {
      const [byName, bySearch] = await Promise.all([api.searchApp(text), api.searchUser(text, 20)]);
      const seen = new Set<string>();
      for (const hit of [...byName, ...bySearch]) {
        if (seen.has(hit.path)) continue;
        seen.add(hit.path);
        local.push({ name: hit.name, path: hit.path });
        if (local.length >= 10) break;
      }
    } catch {
      local = [];
    }
  };

  const startAsk = async (): Promise<void> => {
    const text = input.value.trim();
    if (!text) {
      ctx.toast("Type something to look for first.");
      return;
    }
    query = text;
    requestId = "";
    matches = [];
    chosen = null;
    errMsg = "";
    busy = true;
    stage = "querying";
    render();

    await runLocalCheck(query);

    try {
      const res = await api.askPhone(query);
      requestId = res.id || "";
      if (res.state === "matches") {
        matches = res.matches ?? [];
        stage = matches.length ? "choose" : "none";
      } else if (res.state === "none") {
        stage = "none";
      } else {
        stage = "error";
        errMsg = res.error || "The phone did not answer.";
      }
    } catch (err) {
      stage = "error";
      errMsg = String(err);
    }
    busy = false;
    render();
    await refreshRequests();
  };

  const pick = async (n: number): Promise<void> => {
    const item = matches.find((m) => m.n === n);
    if (!item) {
      ctx.toast(`There is no match number ${n}.`);
      return;
    }
    if (!requestId) {
      ctx.toast("That request has expired — search again.");
      return;
    }
    busy = true;
    render();
    try {
      const path = await api.pickAskMatch(requestId, n);
      chosen = { n, name: item.name, path };
      stage = "done";
      ctx.toast(`Saved ${item.name}`);
    } catch (err) {
      ctx.toast(`Could not fetch ${item.name}: ${String(err)}`);
      stage = "choose";
    }
    busy = false;
    render();
    await refreshRequests();
  };

  const abort = async (): Promise<void> => {
    if (!requestId) {
      stage = "local";
      render();
      return;
    }
    try {
      await api.abortAsk(requestId);
      ctx.toast("Request aborted.");
    } catch {
      ctx.toast("Aborting failed — the phone may already be gone.");
    }
    stage = "local";
    requestId = "";
    matches = [];
    render();
    await refreshRequests();
  };

  // Static controls are created once, so their listeners are attached once.
  go.addEventListener("click", () => {
    void startAsk();
  });
  input.addEventListener("keydown", (e) => {
    if (e.key === "Enter") void startAsk();
  });

  function wire(): void {
    root.querySelectorAll<HTMLButtonElement>("[data-reveal]").forEach((b) => {
      b.addEventListener("click", () => {
        void api.reveal(b.dataset.reveal ?? "");
      });
    });
    const send = root.querySelector<HTMLButtonElement>("#ap-send");
    if (send) {
      send.disabled = busy;
      send.addEventListener("click", () => {
        const n = Number(q<HTMLInputElement>(root, "#ap-n").value || "0");
        void pick(n);
      });
    }
    root.querySelectorAll<HTMLButtonElement>("[data-pick]").forEach((b) => {
      b.addEventListener("click", () => {
        void pick(Number(b.dataset.pick));
      });
    });
    const ab = root.querySelector<HTMLButtonElement>("#ap-abort");
    if (ab) {
      ab.addEventListener("click", () => {
        void abort();
      });
    }
    const again = root.querySelector<HTMLButtonElement>("#ap-again");
    if (again) {
      again.addEventListener("click", () => {
        stage = "local";
        requestId = "";
        matches = [];
        chosen = null;
        render();
        input.focus();
      });
    }
    const num = root.querySelector<HTMLInputElement>("#ap-n");
    num?.addEventListener("input", () => {
      num.value = num.value.replace(/[^0-9]/g, "").slice(0, 2);
    });
  }

  const unlisten = listen(() => {
    // Another screen started a send (Files → Send) or the link dropped.
    gate();
  });
  void refreshV2();
  render();
  input.focus();

  return () => {
    unlisten();
  };
}
