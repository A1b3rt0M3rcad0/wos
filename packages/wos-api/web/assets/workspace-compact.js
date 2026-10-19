import { el, button, details } from "./ui.js";

// Disclosure preferences belong to a selection, never to another identity.
export function compactWorkspace(
  host,
  view,
  {
    onKanban,
    onVerify,
    onMaterial,
    onActivity,
    onReference,
    disclosures = new Map(),
  } = {},
) {
  const key = `${view.subject.kind}/${view.subject.ref.kind}/${view.subject.ref.id}/${view.subject.criterion_id || ""}`;
  const previous = disclosures.get(key) || {};
  const management = details("Details and management");
  management.className = "workspace-management";
  const history = details("Activity history");
  history.className = "workspace-history";
  history.addEventListener("toggle", () => {
    if (history.open && view.coverage?.activity?.status === "not_loaded")
      onActivity?.();
  });
  management.open = Boolean(previous.management);
  history.open = Boolean(previous.history);
  for (const [name, node] of [
    ["management", management],
    ["history", history],
  ])
    node.addEventListener("toggle", () => {
      const value = disclosures.get(key) || {};
      disclosures.set(key, { ...value, [name]: node.open });
    });
  const intro = el("div", undefined, "workspace-compact-intro");
  for (const node of [...host.children]) {
    const facet = node.dataset.workspaceFacet;
    if (facet === "activity") history.append(node);
    else if (
      ["work", "attention"].includes(facet) ||
      node.matches(".workspace-path,.workspace-state,.workspace-meaning")
    )
      intro.append(node);
    else management.append(node);
  }
  const work = intro.querySelector('[data-workspace-facet="work"]');
  if (work) {
    const summary = work.querySelector(".detail-description");
    if (summary) intro.insertBefore(summary, work);
    if (work.querySelector("h4,button")) management.prepend(work);
    else work.remove();
  }
  const attention = intro.querySelector('[data-workspace-facet="attention"]');
  const obligation = attention?.querySelector("li,p");
  if (obligation) {
    const next = el("section", undefined, "workspace-next-obligation");
    next.append(
      el("h3", "What needs attention"),
      el("p", obligation.textContent),
      button(
        "View all obligations",
        () => {
          management.open = true;
          attention.scrollIntoView({ block: "nearest" });
        },
        "quiet",
      ),
    );
    intro.append(next);
    management.prepend(attention);
  }
  const person =
    (view.people || []).find(
      (person) =>
        person.role === "executor" &&
        person.authority_state === "current_authority",
    ) ||
    (view.people || []).find((person) =>
      ["declared_owner", "declared_assignee"].includes(person.role),
    );
  if (person)
    intro.append(
      el(
        "p",
        `${person.role === "executor" ? "Recorded execution authority" : person.role === "declared_owner" ? "Declared owner" : "Declared assignee"}: ${person.actor_ref?.display_name || person.actor_ref?.id || person.principal_id}`,
        "field-help",
      ),
    );
  const applicable = (view.criteria || []).filter(
    (item) => item.definition?.required && item.definition.status !== "retired",
  );
  if (view.coverage?.criteria?.status !== "not_applicable") {
    const met = applicable.filter(
      (item) => item.assessment_applicable && item.assessment?.result === "met",
    ).length;
    const waived = applicable.filter(
      (item) =>
        item.assessment_applicable && item.assessment?.result === "waived",
    ).length;
    intro.append(
      el(
        "p",
        applicable.length === 0 &&
          view.coverage?.criteria?.status === "complete"
          ? "No required success condition defined"
          : `${met}/${applicable.length} required criteria met · ${waived} waived${view.coverage?.criteria?.status === "complete" ? "" : " · loaded criteria only; coverage incomplete"}`,
        "field-help",
      ),
    );
  }
  if (view.conclusion_impact) {
    const impact = view.conclusion_impact;
    intro.append(
      el(
        "p",
        `${impact.open_tasks} open descendant Tasks · ${impact.open_objectives} open descendant Objectives`,
        "field-help",
      ),
    );
  }
  if (
    view.coverage?.required_objectives &&
    view.coverage.required_objectives.status !== "not_applicable"
  ) {
    const required = view.required_objectives || [];
    const accepted = required.filter(
      (item) => item.lifecycle === "achieved",
    ).length;
    intro.append(
      el(
        "p",
        view.coverage.required_objectives.status === "complete"
          ? `${accepted}/${required.length} required Objectives achieved`
          : `${accepted} achieved among ${required.length} loaded required Objectives · more may exist`,
        "field-help",
      ),
    );
  }
  const ref = view.subject.ref;
  const terminal = [
    "achieved",
    "done",
    "cancelled",
    "failed",
    "abandoned",
  ].includes(view.header.lifecycle);
  if (view.subject.kind === "entity" && terminal) {
    const result = button(
      ["achieved", "done"].includes(view.header.lifecycle)
        ? "View accepted result"
        : "View decision",
      () => {
        management.open = true;
        const conclusion =
          management.querySelector('[data-workspace-facet="conclusion"]') ||
          management;
        conclusion.scrollIntoView({ block: "nearest" });
      },
      "workspace-primary-action",
    );
    intro.append(result);
  }
  if (
    onKanban &&
    view.subject.kind === "entity" &&
    ["outcome", "objective"].includes(ref.kind)
  ) {
    const primary = button(
      ref.kind === "outcome" ? "Open Objectives Kanban" : "Open Tasks Kanban",
      () => onKanban(view),
      terminal ? "quiet" : "workspace-primary-action workspace-kanban-open",
    );
    (terminal ? management : intro).append(primary);
  }
  const criteria = management.querySelector(
    '[data-workspace-facet="criteria"]',
  );
  if (criteria && onVerify)
    criteria.append(button("Verify criteria", () => onVerify(view), "quiet"));
  if (["outcome", "objective", "work_item"].includes(ref.kind)) {
    const materials = el(
      "section",
      undefined,
      "detail-section workspace-materials",
    );
    materials.append(el("h3", "Materials"));
    const linked = new Map();
    for (const edge of view.relationships?.items || []) {
      for (const endpoint of [edge.source_ref, edge.target_ref])
        if (endpoint && ["artifact", "evidence"].includes(endpoint.kind))
          linked.set(`${endpoint.kind}/${endpoint.id}`, endpoint);
    }
    for (const [kind, label] of [
      ["artifact", "Artifacts — produced work"],
      ["evidence", "Evidence — observations for verification"],
    ]) {
      const refs = [...linked.values()].filter((ref) => ref.kind === kind);
      materials.append(
        el("h4", `${label} · ${refs.length} linked records loaded`),
      );
      for (const target of refs) {
        const node = view.relationships?.nodes?.find(
          (node) => node.ref.kind === target.kind && node.ref.id === target.id,
        );
        const link = button(
          node?.title || label,
          () => onReference?.(target),
          "quiet",
        );
        materials.append(link);
      }
    }
    materials.append(
      el(
        "p",
        view.coverage?.relationships?.status === "complete"
          ? "These are the materials linked to this item. Linking proof does not approve a criterion."
          : "Material coverage is partial or not loaded. Absence here does not mean no material exists.",
        "field-help",
      ),
    );
    management.prepend(materials);
  }
  if (
    onMaterial &&
    (view.capabilities || []).some((cap) =>
      ["RegisterArtifactInContext", "RegisterEvidenceWithLinks"].includes(
        cap.action_id,
      ),
    )
  )
    management.prepend(button("Add material", () => onMaterial(view), "quiet"));
  host.replaceChildren(intro, management, history);
  host.classList.add("workspace-compact");
}
