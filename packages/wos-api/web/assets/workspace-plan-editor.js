import { copy, message } from "./en-US.js";
import { el, button, inputField, details } from "./ui.js";
import { criteriaEditor } from "./workspace-form-fields.js";
import { kindName } from "./presentation.js";

// This catches draft mistakes before dispatch. The server performs the same
// bounded-plan validation again inside its transaction and authorization guard.
export function validatePlanItems(items) {
  if (!Array.isArray(items) || !items.length) throw Error(copy.text.formBatchEmpty);
  if (items.length > 50) throw Error(copy.text.formBatchItemLimit);
  const keys = new Map();
  let criteriaCount = 0;
  for (const item of items) {
    if (!item.local_key || keys.has(item.local_key))
      throw Error(copy.text.formBatchDuplicateKey);
    if (!["objective", "work_item"].includes(item.kind))
      throw Error(copy.text.formBatchInvalidKind);
    if (!item.title?.trim()) throw Error(copy.text.formBatchMissingTitle);
    if (item.parent_local_key && item.parent_objective_id)
      throw Error(copy.text.formBatchMultipleParents);
    criteriaCount += item.initial_criteria?.length || 0;
    keys.set(item.local_key, item);
  }
  if (criteriaCount > 100) throw Error(copy.text.formConditionsLimit);
  for (const item of items) {
    if (!item.parent_local_key) continue;
    const parent = keys.get(item.parent_local_key);
    if (!parent || parent.kind !== "objective")
      throw Error(copy.text.formBatchInvalidParent);
    const visited = new Set([item.local_key]);
    let current = parent;
    while (current) {
      if (visited.has(current.local_key)) throw Error(copy.text.formBatchCycle);
      visited.add(current.local_key);
      current = current.parent_local_key ? keys.get(current.parent_local_key) : null;
    }
  }
  return items.map((item) => structuredClone(item));
}

export function planSummary(items) {
  return {
    required: items.filter((item) => item.kind === "objective" && item.required_for_outcome).length,
    optional: items.filter((item) => item.kind === "objective" && !item.required_for_outcome).length,
    tasks: items.filter((item) => item.kind === "work_item").length,
  };
}

export function planEditor({ kind = "work_item", parent, parentTitle, outcomeTitle } = {}) {
  const node = el("section", undefined, "workspace-plan-editor");
  const source = inputField(copy.text.formBatchTitles, { type: "textarea", required: true });
  source.input.rows = 5;
  const preview = el("div", undefined, "workspace-plan-preview");
  const feedback = el("p", undefined, "field-help");
  feedback.setAttribute("role", "status");
  let entries = [], previewed = "";
  node.append(source.node, el("p", copy.text.formBatchPreviewHelp, "field-help"));
  const parentLabel = parentTitle || (parent ? copy.text.objective : copy.text.formDirectOutcome);
  const summary = el("p", undefined, "field-context");
  summary.append(el("small", copy.text.formCurrentPlacement), el("strong", parentLabel));
  if (!parent && outcomeTitle) summary.append(el("span", outcomeTitle));
  node.append(summary);

  function read() {
    return entries.map((entry) => {
      const choice = entry.parent.get();
      const isObjective = entry.kind.get() === "objective";
      const item = {
        local_key: entry.key, kind: entry.kind.get(), title: entry.title.get() || "",
        required_for_outcome: isObjective ? entry.required.get() : false,
      };
      if (choice === "existing" && parent) item.parent_objective_id = parent;
      else if (choice?.startsWith("local:")) item.parent_local_key = choice.slice(6);
      const criteria = entry.criteria.get();
      if (criteria.length) item.initial_criteria = criteria;
      return item;
    });
  }
  function updateSummary() {
    const counts = planSummary(entries.map((entry) => ({
      kind: entry.kind.get(), required_for_outcome: entry.required.get(),
    })));
    feedback.textContent = message("formBatchSummary", counts.required, counts.optional, counts.tasks);
  }
  function refreshParents() {
    for (const entry of entries) {
      const current = entry.parent.get();
      entry.parent.input.replaceChildren(
        ...[
          ["outcome", copy.text.formDirectOutcome],
          ...(parent ? [["existing", parentLabel]] : []),
          ...entries.filter((other) => other !== entry && other.kind.get() === "objective")
            .map((other) => [`local:${other.key}`, other.title.get() || copy.text.objective]),
        ].map(([value, label]) => Object.assign(el("option", label), { value })),
      );
      // Keep invalidated local selections visible so changing a parent type
      // cannot silently reparent an existing draft to the Outcome.
      if (current && ![...entry.parent.input.options].some((option) => option.value === current))
        entry.parent.input.append(Object.assign(el("option", copy.text.formBatchInvalidParent), { value: current }));
      entry.parent.input.value = current || (parent ? "existing" : "outcome");
      entry.required.node.hidden = entry.kind.get() !== "objective";
    }
    updateSummary();
  }
  function buildPreview() {
    try {
      const titles = (source.get() || "").split(/\r?\n/).map((title) => title.trim()).filter(Boolean);
      validatePlanItems(titles.map((title, index) => ({ local_key: `item-${index + 1}`, kind, title })));
      entries = [];
      preview.replaceChildren();
      for (const [index, text] of titles.entries()) {
        const row = el("fieldset", undefined, "nested-editor workspace-plan-row");
        const title = inputField(copy.text.title, { required: true, value: text });
        const type = inputField(copy.text.formBatchType, {
          options: [["objective", copy.text.objective], ["work_item", kindName("work_item")]], value: kind,
        });
        const parentField = inputField(copy.text.formBatchParent, { options: [["outcome", copy.text.formDirectOutcome]] });
        parentField.input.value = "outcome";
        if (parent) {
          parentField.input.append(Object.assign(el("option", parentLabel), { value: "existing" }));
          parentField.input.value = "existing";
        }
        const required = inputField(copy.text.requiredForOutcome, { type: "checkbox", value: true });
        const criteria = criteriaEditor();
        const extra = details(copy.text.formDetails);
        extra.append(criteria.node);
        const entry = { key: `item-${index + 1}`, title, kind: type, parent: parentField, required, criteria };
        entries.push(entry);
        row.append(el("legend", String(index + 1)), title.node, type.node, parentField.node, required.node, extra);
        for (const field of [title, type]) field.input.addEventListener("input", refreshParents);
        required.input.addEventListener("change", updateSummary);
        preview.append(row);
      }
      previewed = source.get();
      refreshParents();
      preview.scrollIntoView?.({ block: "nearest" });
    } catch (error) {
      feedback.textContent = error.message;
    }
  }
  node.append(button(copy.text.formBatchPreview, buildPreview, "quiet"), feedback, preview);
  return {
    node,
    get() {
      if (!entries.length) throw Error(copy.text.formBatchEmpty);
      if (source.get() !== previewed) throw Error(copy.text.formBatchPreviewChanged);
      return validatePlanItems(read());
    },
  };
}
