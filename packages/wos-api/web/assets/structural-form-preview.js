import { copy } from "./en-US.js";
import { el, button, inputField } from "./ui.js";
import { display, kindName } from "./presentation.js";
import { placementField } from "./workspace-form-fields.js";

const sameRef = (a, b) => Boolean(a && b && a.namespace_id === b.namespace_id
  && a.outcome_id === b.outcome_id && a.kind === b.kind && a.id === b.id);
const exact = (value, positive = false) => {
  const text = typeof value === "string" ? value
    : Number.isSafeInteger(value) ? String(value) : "";
  if (!/^(0|[1-9][0-9]*)$/.test(text) || BigInt(text) > 18446744073709551615n
    || (positive && text === "0")) throw Error(copy.text.formReviewTargets);
  return text;
};

// A destination selection is only a proposal. The coherent server preview and
// an explicit acknowledgement are required before this editor emits a command.
export function structuralMoveEditor({ scope, entity, kind = entity._kind,
  workspaceView, request, outcomeTitle }) {
  if (!["work_item", "objective"].includes(kind)) throw Error(copy.text.formReviewTargets);
  const ref = { ...scope, kind, id: entity.id };
  const key = kind === "work_item" ? "objective_id" : "parent_objective_id";
  const node = el("section", undefined, "workspace-structural-editor");
  const preview = el("section", undefined, "workspace-move-preview");
  const status = el("p", copy.text.formMovePreviewRequired, "field-help");
  status.setAttribute("role", "status");
  const confirm = inputField(copy.text.formMoveConfirm, { type: "checkbox", value: false });
  confirm.input.disabled = true;
  let result, controller, run = 0, disposed = false;
  function invalidate() {
    ++run;
    controller?.abort();
    result = undefined;
    confirm.input.checked = false;
    confirm.input.disabled = true;
    preview.replaceChildren();
    status.textContent = copy.text.formMovePreviewChanged;
    inspect.disabled = false;
  }
  const placement = placementField({ scope, request, parent: entity[key] ?? null,
    title: workspaceView?.placement?.parent?.title,
    outcomeTitle, label: copy.text.formPreviousPlacement, onChange: invalidate,
    parentRequired: kind === "work_item" });
  const inspect = button(copy.text.formPreviewMove, review, "quiet");
  node.append(placement.node, inspect, status, preview, confirm.node,
    el("p", copy.text.formMovePreserves, "field-help"));

  const ownRef = (value) => value && value.namespace_id === scope.namespace_id
    && value.outcome_id === scope.outcome_id;
  function named(value) {
    return ownRef(value?.ref) ? value.title || kindName(value.ref.kind) : copy.text.workspaceUnknown;
  }
  function listBlockers(parent, state, title) {
    const part = el("div", undefined, "workspace-move-blockers");
    part.append(el("h5", title));
    if (!state || !Array.isArray(state.blockers)) {
      part.append(el("p", copy.text.workspaceUnknown));
    } else {
      const knownTotal = state.coverage?.total_count;
      const total = Number.isSafeInteger(knownTotal) && knownTotal >= 0 ? knownTotal
        : state.coverage?.status === "partial" ? copy.text.workspaceUnknown : state.blockers.length;
      part.append(el("p", `${copy.text.formMoveBlockers}: ${total}`));
      for (const blocker of state.blockers) {
        if (!ownRef(blocker.ref) || !ownRef(blocker.blocked_ref)) continue;
        const item = el("p", blocker.description || copy.text.blocker);
        if (blocker.inherited) item.append(el("small", copy.text.formMoveInherited));
        part.append(item);
      }
      if (state.coverage?.status && state.coverage.status !== "complete") {
        part.append(el("p", copy.text.formMoveBlockersPartial, "field-help"),
          el("small", `${state.blockers.length} / ${total}`));
      }
    }
    parent.append(part);
  }
  async function review() {
    let destination;
    try { destination = placement.get(); } catch (error) {
      status.textContent = error.message; return;
    }
    invalidate();
    const generation = ++run;
    controller = new AbortController();
    inspect.disabled = true;
    status.textContent = copy.text.formMovePreviewLoading;
    try {
      const view = await request(`/namespaces/${encodeURIComponent(scope.namespace_id)}/outcomes/${encodeURIComponent(scope.outcome_id)}/workspace-view`, {
        method: "POST", signal: controller.signal,
        body: JSON.stringify({ subject: { kind: "entity", ref }, placement_preview: { parent_id: destination } }),
      });
      if (disposed || generation !== run || placement.get() !== destination) return;
      const impact = view.placement_preview;
      const parentRef = { ...scope, kind: destination ? "objective" : "outcome", id: destination ?? scope.outcome_id };
      if (view.subject?.kind !== "entity" || !sameRef(view.subject.ref, ref)
        || !impact || !sameRef(impact.new_parent?.ref, parentRef) || !ownRef(impact.old_parent?.ref))
        throw Error(copy.text.formReviewTargets);
      const version = exact(view.entity_version, true), revision = exact(view.outcome_revision);
      preview.append(el("h3", copy.text.formMoveImpact));
      for (const [title, value] of [[copy.text.formPreviousPlacement, impact.old_parent],
        [copy.text.formNewPlacement, impact.new_parent]]) {
        const line = el("p", undefined, "field-context");
        line.append(el("small", title), el("strong", named(value)));
        preview.append(line);
      }
      const count = (value) => Number.isSafeInteger(value) && value >= 0 ? String(value) : copy.text.workspaceUnknown;
      preview.append(el("h4", copy.text.formMoveAffected),
        el("p", `${copy.text.objectives}: ${count(impact.affected_objectives)} · ${copy.text.tasks}: ${count(impact.affected_tasks)}`));
      if (impact.readiness_before || impact.readiness_after) {
        const readiness = el("section", undefined, "workspace-move-readiness");
        readiness.append(el("h4", copy.text.formMoveReadiness));
        for (const [label, value] of [[copy.text.formMoveBefore, impact.readiness_before],
          [copy.text.formMoveAfter, impact.readiness_after]]) {
          const line = el("p", undefined, "workspace-record-fact");
          line.append(el("small", label), el("strong", value?.display_state
            ? display(value.display_state) : copy.text.workspaceUnknown));
          for (const reason of value?.readiness_reasons || []) line.append(el("span", display(reason)));
          readiness.append(line);
        }
        preview.append(readiness);
      }
      for (const change of impact.blocking_changes || []) {
        if (!ownRef(change.ref)) continue;
        const item = el("article", undefined, "workspace-move-change");
        // The server supplies only authorized, bounded work; IDs are inert data.
        item.append(el("h4", change.title || kindName(change.ref.kind)),
          el("small", kindName(change.ref.kind)));
        listBlockers(item, change.before, copy.text.formMoveBefore);
        listBlockers(item, change.after, copy.text.formMoveAfter);
        preview.append(item);
      }
      if (impact.coverage?.status !== "complete") {
        preview.append(el("p", copy.text.formMovePartial, "field-help"));
        if (impact.coverage?.returned_count !== undefined)
          preview.append(el("p", `${impact.coverage.returned_count} / ${count(impact.coverage.total_count)}`, "field-help"));
      }
      if (impact.availability !== "available") {
        status.textContent = impact.error?.message || copy.text.formMoveUnavailable;
        for (const reason of impact.reason_codes || []) preview.append(el("p", display(reason), "field-help"));
        return;
      }
      result = { [key]: destination, expected_version: version, expected_outcome_revision: revision };
      confirm.input.disabled = false;
      status.textContent = copy.text.formMovePreviewRequired;
    } catch (error) {
      if (!disposed && generation === run && error.name !== "AbortError") status.textContent = error.message;
    } finally {
      if (!disposed && generation === run) inspect.disabled = false;
    }
  }
  return {
    node,
    get() {
      if (!result || !confirm.get() || placement.get() !== result[key])
        throw Error(copy.text.formMovePreviewRequired);
      return { ...result };
    },
    dispose() { disposed = true; ++run; controller?.abort(); placement.dispose(); },
  };
}
