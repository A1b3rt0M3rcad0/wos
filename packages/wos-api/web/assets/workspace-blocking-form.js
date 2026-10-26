import { copy, message } from "./en-US.js";
import { el, button, inputField } from "./ui.js";
import { display } from "./presentation.js";

// A scope preview is explicit and read-only. It is not permission to create a
// Blocker; the compound command revalidates the target and observed revision.
export function blockingImpactEditor({ scope, request, getTarget, getPropagation }) {
  const node = el("section", undefined, "workspace-conclusion-impact workspace-blocking-preview");
  const body = el("div", undefined, "workspace-blocking-preview-body");
  body.setAttribute("aria-live", "polite");
  let epoch = 0, observed, acknowledgement, disposed = false;
  const key = () => JSON.stringify({ ref: getTarget(), propagation: getPropagation() });
  const preview = button(copy.text.formPreviewBlocking, async () => {
    const ref = getTarget(), propagation = getPropagation();
    if (!ref || !propagation) return;
    const currentKey = key(), generation = ++epoch;
    observed = null;
    acknowledgement = null;
    body.replaceChildren(el("p", copy.text.formBlockingPreviewLoading, "field-help"));
    preview.disabled = true;
    try {
      const view = await request(`/namespaces/${encodeURIComponent(scope.namespace_id)}/outcomes/${encodeURIComponent(scope.outcome_id)}/workspace-view`, {
        method: "POST", body: JSON.stringify({ subject: { kind: "entity", ref }, blocker_preview: { propagation } }),
      });
      if (disposed || generation !== epoch || key() !== currentKey) return;
      const impact = view.blocker_preview, coverage = view.coverage?.blocker_preview;
      if (!impact || !["complete", "partial"].includes(coverage?.status)
        || impact.propagation !== propagation || !Array.isArray(impact.affected_items)
        || !Number.isSafeInteger(impact.affected_objectives) || !Number.isSafeInteger(impact.affected_tasks)
        || typeof view.outcome_revision !== "string"
        || !/^[0-9]+$/.test(view.outcome_revision)) throw Error(copy.text.formBlockingPreviewRequired);
      observed = { key: currentKey, revision: view.outcome_revision };
      body.replaceChildren(el("h4", copy.text.formBlockingPreviewTitle),
        el("p", message("formBlockingPreviewCount", impact.affected_outcomes || 0, impact.affected_objectives, impact.affected_tasks)));
      const items = el("ul");
      for (const item of impact.affected_items) {
        const row = el("li");
        row.append(el("strong", item.title || display(item.ref.kind)),
          el("span", ` · ${display(item.ref.kind)} · ${display(item.lifecycle)}`));
        items.append(row);
      }
      body.append(items);
      if (coverage.status === "partial") body.append(el("p", message("formBlockingPreviewPartial",
        impact.affected_items.length, coverage.total_count), "field-help"));
      body.append(el("p", copy.text.formBlockingPreviewHelp, "field-help"));
      acknowledgement = inputField(copy.text.formBlockingPreviewConfirm, { type: "checkbox" });
      body.append(acknowledgement.node);
    } catch (error) {
      if (!disposed && generation === epoch && key() === currentKey)
        body.replaceChildren(el("p", error.message, "error"));
    } finally {
      if (!disposed && generation === epoch) preview.disabled = false;
    }
  }, "quiet");
  node.append(preview, body);
  return {
    node,
    sync() {
      epoch++;
      observed = null;
      acknowledgement = null;
      preview.disabled = false;
      const subtree = getPropagation() === "subtree";
      preview.hidden = !subtree;
      body.replaceChildren(el("p", subtree ? copy.text.formBlockingPreviewRequired : copy.text.formBlockingPreviewDirect, "field-help"));
    },
    get() {
      if (getPropagation() !== "subtree") return {};
      if (!observed || observed.key !== key() || !acknowledgement?.get()) throw Error(copy.text.formBlockingPreviewRequired);
      return { observed_outcome_revision: observed.revision };
    },
    dispose() { disposed = true; epoch++; },
  };
}
