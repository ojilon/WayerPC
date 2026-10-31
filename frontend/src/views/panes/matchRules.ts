// Find · Match rules pane (concept 04). Order is the tie-breaker, weight is
// 0..100, each rule can be disabled. Persisted as JSON in the setting
// `match_rules` and consumed by internal/rank.
import { api, type Ctx, type MatchRule } from "../../api";
import { esc, q, qa } from "../../dom";
import { icon } from "../../icons";

interface RuleMeta {
  label: string;
  hint: string;
}

// Labels mirror internal/rank's Rule.Label(); the hint explains the score.
const META: Record<string, RuleMeta> = {
  exact_name: { label: "Exact name", hint: "query equals the file name (extension ignored)" },
  name_contains: { label: "Name contains", hint: "share of query words found in the name" },
  extension: { label: "Extension", hint: "query names the file type (pdf, docx…)" },
  recently_modified: { label: "Recently modified", hint: "30-day half-life decay" },
  folder_hint: { label: "Folder hint", hint: "query words found in the folder path" },
  fuzzy_name: { label: "Fuzzy name", hint: "typo tolerance (bigram similarity)" },
};

export function mountMatchRules(root: HTMLElement, ctx: Ctx): () => void {
  let rules: MatchRule[] = [];
  let dragIndex = -1;
  let loaded = false;

  const paint = (): void => {
    const host = q(root, "#mr-rows");
    host.innerHTML = rules
      .map((r, i) => {
        const meta = META[r.id] ?? { label: r.id, hint: "" };
        return `<div class="rule" draggable="true" data-i="${i}" title="${esc(meta.hint)}">
          <span class="handle" title="Drag to change the order">${icon("drag", 14)}</span>
          <span class="name">${esc(meta.label)}</span>
          <span class="weight">
            <input class="slider" type="range" min="0" max="100" value="${r.weight}" data-i="${i}" aria-label="${esc(meta.label)} weight" />
            <span class="num">${r.weight}</span>
          </span>
          <span class="value" style="text-align:right"></span>
          <span class="enabled"><input class="box" type="checkbox" ${r.enabled ? "checked" : ""} data-i="${i}" aria-label="Enable ${esc(meta.label)}" /></span>
        </div>`;
      })
      .join("");
    bind();
  };

  const bind = (): void => {
    qa<HTMLInputElement>(root, 'input[type="range"]').forEach((el) => {
      el.addEventListener("input", () => {
        const i = Number(el.dataset.i);
        rules[i].weight = Number(el.value);
        const num = el.parentElement?.querySelector(".num");
        if (num) num.textContent = el.value;
      });
    });
    qa<HTMLInputElement>(root, 'input[type="checkbox"]').forEach((el) => {
      el.addEventListener("change", () => {
        rules[Number(el.dataset.i)].enabled = el.checked;
      });
    });
    qa<HTMLElement>(root, ".rule").forEach((row) => {
      row.addEventListener("dragstart", (ev) => {
        dragIndex = Number(row.dataset.i);
        row.classList.add("dragging");
        (ev as DragEvent).dataTransfer?.setData("text/plain", String(dragIndex));
      });
      row.addEventListener("dragend", () => {
        row.classList.remove("dragging");
        dragIndex = -1;
      });
      row.addEventListener("dragover", (ev) => ev.preventDefault());
      row.addEventListener("drop", (ev) => {
        ev.preventDefault();
        const target = Number(row.dataset.i);
        if (dragIndex < 0 || dragIndex === target) return;
        const [moved] = rules.splice(dragIndex, 1);
        rules.splice(target, 0, moved);
        dragIndex = -1;
        paint();
      });
    });
  };

  const load = async (): Promise<void> => {
    try {
      rules = await api.getMatchRules();
    } catch {
      rules = [];
    }
    loaded = rules.length > 0;
    paint();
    if (!loaded) {
      q(root, "#mr-status").textContent = "Rules unavailable — the data folder is not open yet.";
    }
  };

  root.innerHTML = `
    <div class="rulehead" style="margin-top:20px">
      <span></span><span>Rule</span><span>Weight</span><span style="text-align:right">Value</span><span style="text-align:right">Enabled</span>
    </div>
    <div class="rules" id="mr-rows"></div>
    <div class="rulefoot">
      <button class="link" id="mr-reset">Reset to default</button>
      <button class="link" id="mr-save">Save</button>
      <span class="tiny faint" id="mr-status"></span>
    </div>
    <p class="hint">Higher weights appear earlier in search results. Drag rules to change their order —
      the order breaks ties between equally scored files.</p>`;

  q(root, "#mr-reset").addEventListener("click", async () => {
    try {
      await api.resetMatchRules();
      await load();
      ctx.toast("Match rules reset to defaults");
    } catch {
      ctx.toast("Could not reset the rules");
    }
  });

  q(root, "#mr-save").addEventListener("click", async () => {
    try {
      await api.setMatchRules(rules);
      await load();
      const top = rules.filter((r) => r.enabled && r.weight > 0).sort((a, b) => b.weight - a.weight)[0];
      ctx.toast(top ? `Saved — strongest rule: ${META[top.id]?.label ?? top.id} (${top.weight})` : "Saved — every rule is disabled");
    } catch {
      ctx.toast("Could not save the rules");
    }
  });

  void load();
  return () => {};
}
