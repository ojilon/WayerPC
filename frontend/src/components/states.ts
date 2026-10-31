// Empty + error states (concept 10). Calm, actionable feedback: a short
// title, one line of explanation, and text links — never a red box or a card.
export interface StateAction {
  label: string;
  onClick: () => void;
}

export interface StateOptions {
  title: string;
  body?: string;
  actions?: StateAction[];
}

function build(opts: StateOptions, error: boolean): HTMLElement {
  const el = document.createElement("div");
  el.className = `state${error ? " error" : ""}`;
  el.innerHTML = `
    ${error ? "" : '<span class="rule"></span>'}
    <div class="title"></div>
    <div class="body"></div>
    <div class="actions"></div>`;
  el.querySelector(".title")!.textContent = opts.title;
  const body = el.querySelector(".body")!;
  if (opts.body) body.textContent = opts.body;
  else body.remove();
  const actions = el.querySelector(".actions")!;
  (opts.actions ?? []).forEach((a) => {
    const b = document.createElement("button");
    b.className = "link";
    b.textContent = a.label;
    b.addEventListener("click", a.onClick);
    actions.appendChild(b);
  });
  if ((opts.actions ?? []).length === 0) actions.remove();
  return el;
}

/** "Nothing here yet" (files empty, no search results, empty sync set). */
export function emptyState(opts: StateOptions): HTMLElement {
  return build(opts, false);
}

/** "Something went wrong" (unreadable folder, index unavailable). */
export function errorState(opts: StateOptions): HTMLElement {
  return build(opts, true);
}
