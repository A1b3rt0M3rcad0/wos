import { copy } from "./en-US.js";
import { el, button, inputField, listField, details } from "./ui.js";
import { referencePicker, referenceKinds } from "./reference-picker.js";

// One definition is used by initial criteria and later criterion editing.
// These are human-flow suggestions, not changes to public command defaults.
export function criterionModes() {
  return [
    ["evidence_review", copy.text.evidenceReviewAssessRegisteredProof],
    ["attestation", copy.text.attestationAuthorizedDeclaration],
    ["external_evaluation", copy.text.externalEvaluationIndependentEvaluator],
  ];
}

export function criteriaEditor(initial = []) {
  const node = el("fieldset", undefined, "nested-editor workspace-criteria-editor");
  const rows = el("div", undefined, "workspace-criteria-rows");
  const empty = el("p", copy.text.formNoConditions, "field-help");
  const entries = [];
  node.append(el("legend", copy.text.formSuccessConditions),
    el("p", copy.text.formConditionsHelp, "field-help"), rows, empty);
  function add(value = {}) {
    const row = el("div", undefined, "nested-editor-row workspace-criterion-row");
    const title = inputField(copy.text.formSuccessQuestion, {
      required: true, value: value.title,
    });
    const required = inputField(copy.text.formConditionRequired, {
      type: "checkbox", value: value.required ?? true,
    });
    const mode = inputField(copy.text.howWillItBeVerified, {
      options: criterionModes(), value: value.verification_mode || "evidence_review",
    });
    const more = details(copy.text.description);
    const description = inputField(copy.text.description, {
      type: "textarea", value: value.description,
    });
    more.append(description.node);
    const entry = { row, title, required, mode, description };
    entries.push(entry);
    row.append(title.node, required.node, mode.node, more,
      button(copy.text.formRemoveCondition, () => {
        entries.splice(entries.indexOf(entry), 1);
        row.remove();
        empty.hidden = entries.length > 0;
        addButton.focus();
      }, "quiet editor-remove"));
    rows.append(row);
    empty.hidden = true;
    return title.input;
  }
  const addButton = button(copy.text.formAddCondition, () => {
    if (entries.length >= 100) return;
    add().focus();
  }, "quiet editor-add");
  node.append(addButton);
  initial.forEach(add);
  return {
    node,
    get() {
      if (entries.length > 100) throw Error(copy.text.formConditionsLimit);
      return entries.map(({ title, required, mode, description }) => {
        if (!title.get()) throw Error(copy.text.formConditionTitleMissing);
        return {
          title: title.get(), description: description.get() || "",
          required: required.get(), verification_mode: mode.get(),
        };
      });
    },
  };
}

export function executionSpecEditor({ initial = {}, scope, request }) {
  const node = el("section", undefined, "nested-editor workspace-spec-editor");
  const lists = {};
  for (const [key, label] of Object.entries({
    instructions: copy.text.instructions,
    constraints: copy.text.constraints,
    deliverables: copy.text.deliverables,
    scope_hints: copy.text.scopeHints,
  })) {
    const field = listField(label, initial?.[key] || []);
    node.append(field.node);
    lists[key] = field.get;
  }
  const context = referencePicker({
    label: copy.text.contextReferences, kinds: referenceKinds,
    scope, request, multiple: true, initial: initial?.context_refs || [],
  });
  node.append(context.node);
  return {
    node,
    get: () => ({
      ...Object.fromEntries(Object.entries(lists).map(([key, get]) => [key, get()])),
      context_refs: context.get(),
    }),
    dispose: context.dispose,
  };
}

export function localDateTime(value) {
  if (!value) return "";
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) return "";
  return new Date(date.getTime() - date.getTimezoneOffset() * 60000)
    .toISOString().slice(0, 16);
}

// The contextual parent stays fixed until the user explicitly asks to change it.
// Its label is presentation; authorization is always revalidated by the command.
export function placementField({
  scope, request, parent, title, label = copy.text.formCurrentPlacement,
  outcomeTitle, allowChange = true, onChange,
}) {
  const node = el("section", undefined, "field-context workspace-placement");
  const context = el("p", undefined, "workspace-placement-summary");
  context.append(el("small", label), el("strong", parent
    ? title || copy.text.objective : copy.text.formDirectOutcome));
  if (!parent && outcomeTitle) context.append(el("span", outcomeTitle));
  node.append(context);
  let picker;
  const selectedParent = () => picker ? picker.get()?.id ?? null : parent ?? null;
  let observed = parent ?? null;
  const observer = new MutationObserver(() => {
    let current;
    try { current = selectedParent(); } catch { current = undefined; }
    if (current !== observed) {
      observed = current;
      onChange?.(current);
    }
  });
  observer.observe(node, { childList: true, subtree: true });
  if (allowChange) node.append(button(copy.text.formChangePlacement, () => {
    if (picker) return picker.node.querySelector("input")?.focus();
    picker = referencePicker({
      label: copy.text.formNewPlacement, kinds: ["objective"], scope, request,
      initial: parent ? [{ ref: { ...scope, kind: "objective", id: parent }, title }] : [],
    });
    node.append(picker.node,
      el("p", copy.text.formParentChangeHelp, "field-help"));
    picker.node.querySelector("input")?.focus();
  }, "quiet"));
  return {
    node,
    get: selectedParent,
    dispose() { observer.disconnect(); picker?.dispose(); },
  };
}
