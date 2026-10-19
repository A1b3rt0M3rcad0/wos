import { copy, message } from "./en-US.js";
import { el, inputField } from "./ui.js";
import { display } from "./presentation.js";

const consequences = {
  child_lifecycles_unchanged: copy.text.formConclusionLifecycleUnchanged,
  child_proof_unchanged: copy.text.formConclusionProofUnchanged,
  outcome_operational_scope_closed: copy.text.formConclusionOutcomeClosed,
  objective_closed_to_new_children: copy.text.formConclusionObjectiveClosed,
  descendant_workflow_unchanged: copy.text.formConclusionDescendantsUnchanged,
};

// The human confirmation is tied to one server read revision. Refreshing it is
// deliberate and clears acknowledgement without replacing the person's reason.
export function conclusionImpactEditor(initialView) {
  const node = el("section", undefined, "nested-editor workspace-conclusion-impact");
  let view, acknowledgement, valid = false;
  function update(next) {
    view = next;
    acknowledgement = null;
    const impact = view?.conclusion_impact;
    const coverage = view?.coverage?.conclusion_impact;
    valid = !!impact && ["complete", "partial"].includes(coverage?.status)
      && typeof view.outcome_revision === "string" && /^[0-9]+$/.test(view.outcome_revision)
      && typeof view.entity_version === "string" && /^[1-9][0-9]*$/.test(view.entity_version)
      && Number.isSafeInteger(impact.open_objectives) && impact.open_objectives >= 0
      && Number.isSafeInteger(impact.open_tasks) && impact.open_tasks >= 0
      && Array.isArray(impact.open_items) && Array.isArray(impact.consequences)
      && impact.consequences.every((code) => consequences[code]);
    node.replaceChildren(el("h3", copy.text.formConclusionImpact));
    if (!valid) {
      node.append(el("p", copy.text.formConclusionImpactUnavailable, "field-help"));
      return;
    }
    const total = impact.open_objectives + impact.open_tasks;
    node.append(el("p", message("formConclusionOpenCount",
      impact.open_objectives, impact.open_tasks)));
    if (impact.open_items.length) {
      const list = el("ul", undefined, "workspace-impact-items");
      for (const item of impact.open_items) {
        const row = el("li");
        row.append(el("strong", item.title || display(item.ref.kind)),
          el("span", ` · ${display(item.ref.kind)} · ${display(item.lifecycle)}`));
        list.append(row);
      }
      node.append(list);
    } else node.append(el("p", copy.text.formConclusionNoOpenWork, "field-help"));
    if (coverage.status === "partial")
      node.append(el("p", message("formConclusionPartial", impact.open_items.length, total), "field-help"));
    const effect = el("ul");
    for (const code of impact.consequences) effect.append(el("li", consequences[code]));
    node.append(effect);
    if (total) {
      acknowledgement = inputField(copy.text.formConclusionConfirm, { type: "checkbox", required: true });
      node.append(acknowledgement.node);
    }
  }
  update(initialView);
  return {
    node, update,
    get() {
      if (!valid) throw Error(copy.text.formConclusionImpactUnavailable);
      if (acknowledgement && !acknowledgement.get()) throw Error(copy.text.formConclusionReviewRequired);
      return { expected_version: view.entity_version, observed_outcome_revision: view.outcome_revision };
    },
  };
}
