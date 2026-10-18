import { copy } from "./en-US.js";
import { el, button } from "./ui.js";
import { kindName, shortId } from "./presentation.js";

const categories = {
  Plan: ["create_outcome", "update_outcome", "create_objective", "update_objective",
    "create_work_item", "update_work_item", "create_roadmap", "open_roadmap_draft",
    "replace_roadmap_draft", "discard_roadmap_draft", "publish_roadmap_draft",
    "activate_roadmap_revision", "deactivate_roadmap_revision", "archive_roadmap", "reopen_roadmap"],
  Verify: ["add_criterion", "record_criterion_assessment", "revise_criterion",
    "retire_criterion", "achieve_outcome", "achieve_objective"],
  Connect: ["add_dependency", "remove_dependency"],
  Problems: ["create_issue", "report_issue_with_blocker", "create_blocker",
    "update_issue", "investigate_issue", "resolve_issue", "resolve_issue_and_blockers",
    "reopen_issue", "update_blocker_description", "resolve_blocker", "cancel_blocker"],
  Records: ["register_evidence", "retract_evidence", "create_evidence_link",
    "register_artifact", "withdraw_artifact", "propose_decision", "accept_decision", "reject_decision"],
};
const descriptions = {
  create_objective: "Objective", create_work_item: "Task", add_dependency: "Dependency",
  create_issue: "Issue", report_issue_with_blocker: "IssueBlocker", create_blocker: "Blocker",
  resolve_issue: "ResolveIssue", resolve_issue_and_blockers: "ResolveIssueBlockers",
  resolve_blocker: "ReleaseBlocker", cancel_blocker: "CancelBlocker",
  register_evidence: "Evidence", retract_evidence: "RetractEvidence", create_evidence_link: "EvidenceLink",
  register_artifact: "Artifact", propose_decision: "Decision", create_roadmap: "Roadmap",
  add_criterion: "Criterion", record_criterion_assessment: "Assessment",
  achieve_outcome: "Certify", achieve_objective: "Certify", complete_work_item: "CompleteTask",
};

// Presentation only: callers supply the already authorized, protocol-compatible
// action list. Grouping cannot add a command or decide whether it is available.
export function groupActions(actions = []) {
  const groups = new Map();
  for (const action of actions) {
    const category = Object.keys(categories).find((key) => categories[key].includes(action.name)) || "Progress";
    const key = `${action.advanced ? "advanced" : "normal"}:${category}`;
    if (!groups.has(key)) groups.set(key, { category, advanced: Boolean(action.advanced), actions: [] });
    groups.get(key).actions.push(action);
  }
  return [...groups.values()];
}

let descriptionSequence = 0;
export function renderActionMenu(host, { actions = [], kind = "outcome", entity = {}, onAction } = {}) {
  host.replaceChildren();
  const title = entity.title || entity.name || (entity.id ? shortId(entity.id) : "");
  const context = el("div", undefined, "action-context");
  context.append(el("strong", `${kindName(kind)}${title ? ` · ${title}` : ""}`),
    el("p", copy.text.actionScopeNote));
  host.append(context);
  const advanced = el("details");
  advanced.append(el("summary", copy.text.advanced));
  for (const group of groupActions(actions)) {
    const section = el("section", undefined, "action-group");
    section.append(el("h3", copy.text[`actionGroup${group.category}`]));
    const options = el("div", undefined, "action-group-options");
    for (const action of group.actions) {
      const control = button("", () => onAction?.(action), "quiet");
      control.append(el("span", action.label, "action-label"));
      // Keep existing control names stable, with the explanation announced as
      // a description rather than becoming part of the command's name.
      control.setAttribute("aria-label", action.label);
      const suffix = descriptions[action.name];
      if (suffix) {
        const description = el("small", copy.text[`actionDescription${suffix}`], "action-description");
        description.id = `action-description-${++descriptionSequence}`;
        control.setAttribute("aria-describedby", description.id);
        control.append(description);
      }
      options.append(control);
    }
    section.append(options);
    (group.advanced ? advanced : host).append(section);
  }
  if (advanced.children.length > 1) host.append(advanced);
  if (!actions.length)
    host.append(el("p", copy.text.noChangesAreAvailableForYourCurrentPermissionsAnd));
}
