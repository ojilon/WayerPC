// Find hub (concepts 03/05/06/04). The header + tab row stay put; the pane
// below is chosen by the route (#/find, #/find/phone, #/find/requests,
// #/find/rules). Panes own their own data so wiring one never touches the
// others.
import type { Ctx } from "../api";
import { esc } from "../dom";
import type { Route } from "../nav";
import { mountAskPhone } from "./panes/askPhone";
import { mountFindLocal } from "./panes/findLocal";
import { mountMatchRules } from "./panes/matchRules";
import { mountRequests } from "./panes/requests";

interface HeadConfig {
  title: string;
  sub: string;
  right?: string;
}

export function mountFind(root: HTMLElement, ctx: Ctx, route: Route): () => void {
  const rules = route === "find/rules";
  const head: HeadConfig = rules
    ? { title: "Match rules", sub: "Rank local search results by relevance" }
    : route === "find/phone"
      ? { title: "Ask phone", sub: "Search your phone from this PC" }
      : route === "find/requests"
        ? { title: "Incoming request", sub: "Phone queries and responses" }
        : { title: "Find", sub: "Search files on this PC" };

  root.innerHTML = `
    <section class="page">
      <header class="page-head">
        <div>
          <h1 class="screen">${esc(head.title)}</h1>
          <p class="screen-sub" id="fd-sub">${esc(head.sub)}</p>
        </div>
        <div class="head-links">
          ${
            rules
              ? '<a class="link" href="#/find">&larr; Back to Find</a>'
              : '<a class="link" href="#/find/rules">Match rules</a>'
          }
        </div>
      </header>
      ${
        rules
          ? ""
          : `<div class="tabs" role="tablist">
              <a class="tab" role="tab" href="#/find" data-tab="find">Search</a>
              <a class="tab" role="tab" href="#/find/phone" data-tab="find/phone">Phone</a>
              <a class="tab" role="tab" href="#/find/requests" data-tab="find/requests">Requests</a>
            </div>`
      }
      <div id="fd-pane"></div>
    </section>`;

  const pane = root.querySelector("#fd-pane") as HTMLElement;
  const activeTab = route === "find/rules" ? "find" : route;
  root.querySelectorAll<HTMLElement>(".tabs .tab").forEach((el) => {
    const on = el.dataset.tab === activeTab;
    el.classList.toggle("on", on);
    el.setAttribute("aria-selected", on ? "true" : "false");
  });

  switch (route) {
    case "find/rules":
      return mountMatchRules(pane, ctx);
    case "find/phone":
      return mountAskPhone(pane, ctx);
    case "find/requests":
      return mountRequests(pane, ctx);
    case "find":
    default:
      return mountFindLocal(pane, ctx);
  }
}
