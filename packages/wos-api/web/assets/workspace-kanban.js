import { el, button } from "./ui.js";
import { display } from "./presentation.js";
import {
  renderWorkspaceView,
  renderWorkspaceActions,
  workspaceActions,
  workspaceCommandName,
} from "./workspace-view.js";

const stages = [
  ["todo", "To do"],
  ["in_progress", "In progress"],
  ["verification", "Verification"],
  ["accepted", "Accepted"],
];
const key = (ref) => `${ref.kind}/${ref.id}`;

// One modal, an explicit navigation stack, and read-only presentation stages.
// Every mutation leaves through the existing authorized command surface.
export function createContextualKanban({
  requestCards,
  query,
  onAction,
  onReturn,
  scopeKey,
  onError,
} = {}) {
  const dialog = el("dialog", undefined, "workspace-kanban");
  dialog.id = "workspace-kanban";
  dialog.setAttribute("aria-labelledby", "workspace-kanban-title");
  document.body.append(dialog);
  let frames = [],
    epoch = 0,
    sessionScope = "",
    returnFocus,
    suspended = false;
  const disclosures = new Map();
  const current = () => frames.at(-1);
  function valid(token) {
    return epoch === token && sessionScope === scopeKey();
  }
  function fail(error, token) {
    if (!valid(token)) return;
    current().error = error.message || "Unable to load this view.";
    render();
  }
  function close(restore = true) {
    ++epoch;
    suspended = false;
    const origin = frames[0]?.view,
      scope = sessionScope;
    if (dialog.open) dialog.close();
    if (restore && origin && onReturn)
      Promise.resolve(onReturn(origin))
        .then((target) => {
          if (sessionScope !== scope || scopeKey() !== scope) return;
          const focus = returnFocus?.isConnected ? returnFocus : target;
          focus?.focus({ preventScroll: true });
        })
        .catch(onError);
  }
  dialog.addEventListener("cancel", (event) => {
    event.preventDefault();
    if (current()?.selected) {
      const ref = current().selected.subject.ref;
      current().selected = null;
      render();
      dialog.querySelector(`[data-kanban-ref="${key(ref)}"]`)?.focus();
    } else close();
  });
  dialog.addEventListener("close", () => {
    returnFocus?.isConnected && returnFocus.focus({ preventScroll: true });
  });
  function reset() {
    close(false);
    frames = [];
    disclosures.clear();
    dialog.replaceChildren();
    sessionScope = "";
  }
  async function load(append = false) {
    const frame = current(),
      token = ++epoch;
    frame.loading = true;
    frame.error = "";
    render();
    try {
      const [page, parent] = await Promise.all([
        requestCards({
          parent_ref: frame.view.subject.ref,
          limit: 50,
          ...(append ? { cursor: frame.coverage.next_cursor } : {}),
        }),
        append ? Promise.resolve(null) : query(frame.view.subject),
      ]);
      if (parent && page.outcome_revision !== parent.outcome_revision)
        throw new Error(
          "The work changed during loading. Refresh to read one consistent revision.",
        );
      if (!valid(token)) return;
      if (append && page.outcome_revision !== frame.revision)
        throw new Error(
          "The work changed. Refresh this Kanban before loading another page.",
        );
      if (parent) frame.view = parent;
      frame.items = append ? [...frame.items, ...page.items] : page.items;
      frame.revision = page.outcome_revision;
      frame.coverage = page.coverage;
    } catch (error) {
      fail(error, token);
    } finally {
      if (valid(token)) {
        frame.loading = false;
        render();
      }
    }
  }
  function makeFrame(view) {
    return {
      view,
      content: view.subject.ref.kind === "outcome" ? "objectives" : "tasks",
      items: [],
      coverage: { status: "not_loaded" },
      search: "",
      cancelled: false,
      required: false,
      priority: "",
      assignee: "",
      filtersOpen: false,
      loading: false,
      scroll: new Map(),
    };
  }
  async function open(view, { mode } = {}) {
    if (
      !view?.subject?.ref ||
      (!mode && !["outcome", "objective"].includes(view.subject.ref.kind))
    )
      return;
    reset();
    sessionScope = scopeKey();
    returnFocus = document.activeElement;
    frames = [makeFrame(view)];
    current().mode = mode;
    dialog.showModal();
    render();
    if (!mode) await load();
  }
  async function navigate(view) {
    saveScroll();
    frames.push(makeFrame(view));
    render();
    await load();
  }
  function saveScroll() {
    for (const node of dialog.querySelectorAll("[data-column]"))
      current()?.scroll.set(node.dataset.column, node.scrollTop);
  }
  async function inspect(ref) {
    const token = ++epoch;
    try {
      const view = await query({ kind: "entity", ref });
      if (!valid(token)) return;
      current().selected = view;
      render();
      dialog.querySelector(".workspace-kanban-inspection h3")?.focus();
    } catch (error) {
      fail(error, token);
    }
  }
  async function act(action, capability, view) {
    saveScroll();
    suspended = true;
    ++epoch;
    dialog.close();
    try {
      await onAction(action, capability, view);
    } catch (error) {
      onError?.(error);
    }
  }
  function render() {
    if (!current() || sessionScope !== scopeKey()) return reset();
    const frame = current();
    const header = el("header", undefined, "workspace-kanban-header");
    const title = el(
      "h2",
      frame.mode === "criteria"
        ? "Verify criteria"
        : frame.mode === "materials"
          ? "Add material"
          : frame.content === "objectives"
            ? "Objectives Kanban"
            : "Tasks Kanban",
    );
    title.id = "workspace-kanban-title";
    header.append(title, button("Close Kanban", close, "quiet"));
    const breadcrumb = el("nav", undefined, "workspace-kanban-breadcrumb");
    breadcrumb.setAttribute("aria-label", "Kanban context");
    for (const [i, item] of frames.entries())
      breadcrumb.append(
        button(
          item.view.header.title,
          () => {
            saveScroll();
            ++epoch;
            frames = frames.slice(0, i + 1);
            render();
            if (current().coverage.status === "not_loaded") load();
          },
          "quiet",
        ),
      );
    const controls = el("div", undefined, "workspace-kanban-controls");
    for (const [content, label] of [
      [
        "objectives",
        frame.view.subject.ref.kind === "objective"
          ? "Subobjectives"
          : "Objectives",
      ],
      [
        "tasks",
        frame.view.subject.ref.kind === "outcome" ? "Direct Tasks" : "Tasks",
      ],
    ]) {
      const control = button(
        label,
        () => {
          saveScroll();
          frame.content = content;
          frame.selected = null;
          render();
        },
        "quiet",
      );
      control.setAttribute("aria-pressed", String(frame.content === content));
      controls.append(control);
    }
    const search = el("input");
    search.type = "search";
    search.placeholder = "Find in loaded items";
    search.value = frame.search;
    search.setAttribute("aria-label", "Search loaded Kanban items");
    search.addEventListener("input", () => {
      frame.search = search.value;
      renderColumns();
    });
    controls.append(search);
    const filters = el("details", undefined, "workspace-kanban-filters");
    filters.append(el("summary", "Filters"));
    filters.open = frame.filtersOpen;
    filters.addEventListener("toggle", () => {
      frame.filtersOpen = filters.open;
    });
    const filterBody = el("div");
    filters.append(filterBody);
    controls.append(filters);
    const required = button(
      "Required Objectives only",
      () => {
        frame.required = !frame.required;
        render();
      },
      "quiet",
    );
    required.setAttribute("aria-pressed", String(frame.required));
    required.hidden = frame.content !== "objectives";
    filterBody.append(required);
    for (const [field, label, options] of [
      [
        "priority",
        "Filter loaded items by priority",
        [
          ["", "All priorities"],
          ["critical", "Critical"],
          ["high", "High"],
          ["normal", "Normal"],
          ["low", "Low"],
        ],
      ],
      [
        "assignee",
        "Filter loaded items by declared responsibility",
        [
          ["", "All declared participants"],
          ...new Map(
            frame.items.flatMap((item) =>
              (item.participants || [])
                .filter((person) => person.actor_ref?.id)
                .map((person) => [
                  person.actor_ref.id,
                  [
                    person.actor_ref.id,
                    person.actor_ref.display_name || person.actor_ref.id,
                  ],
                ]),
            ),
          ).values(),
        ],
      ],
    ]) {
      const select = el("select");
      select.setAttribute("aria-label", label);
      for (const [value, title] of options) {
        const option = el("option", title);
        option.value = value;
        select.append(option);
      }
      select.value = frame[field];
      select.addEventListener("change", () => {
        frame[field] = select.value;
        renderColumns();
      });
      filterBody.append(select);
    }
    const cancelled = button(
      frame.cancelled ? "Hide cancelled" : "Show cancelled",
      () => {
        frame.cancelled = !frame.cancelled;
        render();
      },
      "quiet",
    );
    cancelled.setAttribute("aria-pressed", String(frame.cancelled));
    filterBody.append(cancelled);
    controls.append(button("Refresh", () => load(), "quiet"));
    const status = el(
      "p",
      frame.loading
        ? "Loading work…"
        : frame.coverage.status === "complete"
          ? `${frame.items.length} direct children loaded · revision ${frame.revision}`
          : "Partial view of direct children. More work may exist. Counts below describe loaded cards only.",
      "workspace-kanban-status",
    );
    status.setAttribute("role", "status");
    const body = el("div", undefined, "workspace-kanban-body");
    const columns = el("div", undefined, "workspace-kanban-columns");
    body.append(columns);
    function renderColumns() {
      columns.replaceChildren();
      const wanted = frame.content === "tasks" ? "work_item" : "objective";
      const items = frame.items.filter(
        (item) =>
          item.ref.kind === wanted &&
          item.title.toLowerCase().includes(frame.search.toLowerCase()) &&
          (!frame.required ||
            wanted !== "objective" ||
            item.required_for_outcome) &&
          (!frame.priority || item.priority === frame.priority) &&
          (!frame.assignee ||
            (item.participants || []).some(
              (person) => person.actor_ref?.id === frame.assignee,
            )),
      );
      for (const [stage, label] of [
        ...stages,
        ...(frame.cancelled ? [["cancelled", "Cancelled / out of scope"]] : []),
      ]) {
        const section = el("section", undefined, "workspace-kanban-column");
        section.dataset.column = stage;
        const matching = items.filter((item) => item.stage === stage);
        section.append(el("h3", `${label} · ${matching.length} loaded`));
        for (const item of matching) {
          const card = button(
            "",
            () => inspect(item.ref),
            "workspace-kanban-card",
          );
          card.dataset.kanbanRef = key(item.ref);
          card.append(
            el("strong", item.title),
            el("small", display(item.lifecycle)),
          );
          if (item.display_state !== item.lifecycle)
            card.append(el("span", display(item.display_state)));
          if (item.ref.kind === "objective")
            card.append(
              el(
                "small",
                item.required_for_outcome
                  ? "Required for Outcome"
                  : "Optional for Outcome",
              ),
            );
          if (item.criteria)
            card.append(
              el(
                "small",
                `${item.criteria.met} met · ${item.criteria.waived} waived · ${item.criteria.pending} pending · ${item.criteria.active} active criteria`,
              ),
            );
          for (const person of item.participants || [])
            card.append(
              el(
                "small",
                `Declared responsibility: ${person.actor_ref?.display_name || person.actor_ref?.id || "Not recorded"}`,
              ),
            );
          if (item.attention_reason)
            card.append(el("small", display(item.attention_reason)));
          if (item.ref.kind === "work_item")
            card.append(
              el(
                "small",
                "Open to inspect current dependencies, blockers and execution authority.",
              ),
            );
          section.append(card);
        }
        if (!matching.length)
          section.append(el("p", "No matching loaded items.", "field-help"));
        columns.append(section);
        section.scrollTop = frame.scroll.get(stage) || 0;
      }
    }
    if (frame.mode === "materials") {
      const wizard = el("section", undefined, "workspace-material-wizard");
      wizard.append(
        el(
          "p",
          "Choose what you want to record. An Artifact is a deliverable; Evidence is an observation used for verification. Recording either does not approve a criterion.",
        ),
      );
      for (const [action, label, help] of [
        [
          "RegisterArtifactInContext",
          "What was produced — Artifact",
          "Register a commit, pull request, document or other deliverable reference.",
        ],
        [
          "RegisterEvidenceWithLinks",
          "Proof or observation — Evidence",
          "Use an external source, an Artifact, an authenticated observation or existing Evidence, with an explicit stance.",
        ],
      ]) {
        const cap = frame.view.capabilities.find(
          (cap) => cap.action_id === action,
        );
        if (cap) {
          wizard.append(
            button(label, () => act(cap.action_id, cap, frame.view), "quiet"),
            el("p", help, "field-help"),
          );
        }
      }
      wizard.append(
        el(
          "p",
          "External references are supplied by participants. WOS does not fetch or validate their contents. File upload is not available.",
          "field-help",
        ),
      );
      columns.replaceChildren(wizard);
      controls.hidden = true;
      status.hidden = true;
    } else if (frame.mode === "criteria") {
      const checklist = el("section", undefined, "workspace-review-checklist");
      checklist.append(
        el(
          "p",
          `Review ${frame.view.header.title} · observed revision ${frame.view.outcome_revision}`,
        ),
        el(
          "p",
          "Record each assessment through its authorized action. Verification and acceptance are separate decisions. Waived criteria are exceptions, not verified success.",
          "field-help",
        ),
      );
      for (const criterion of frame.view.criteria || []) {
        const definition = criterion.definition;
        if (!definition || definition.status === "retired") continue;
        const row = el("article", undefined, "workspace-criterion");
        row.append(
          el("h3", definition.title),
          el("p", definition.description || ""),
          el(
            "strong",
            criterion.assessment_applicable && criterion.assessment
              ? display(criterion.assessment.result)
              : "Not verified",
          ),
          el(
            "small",
            ` · definition ${definition.criterion_revision} · ${display(definition.verification_mode)}`,
          ),
        );
        row.append(
          button(
            "Review this criterion",
            async () => {
              const token = ++epoch;
              try {
                const view = await query({
                  kind: "criterion",
                  ref: definition.owner_ref,
                  criterion_id: definition.id,
                });
                if (valid(token)) {
                  frame.selected = view;
                  render();
                }
              } catch (error) {
                fail(error, token);
              }
            },
            "quiet",
          ),
        );
        checklist.append(row);
      }
      if (!frame.view.criteria?.length)
        checklist.append(
          el(
            "p",
            "No criteria loaded. Add an explicit success condition before verifying.",
          ),
        );
      if (frame.view.coverage?.criteria?.status !== "complete")
        checklist.append(
          el(
            "p",
            "Criteria coverage is incomplete. Inspect the owner to load remaining criteria.",
            "field-help",
          ),
        );
      columns.replaceChildren(checklist);
      controls.hidden = true;
      status.hidden = true;
    } else renderColumns();
    if (frame.selected) {
      const selected = frame.selected,
        aside = el("section", undefined, "workspace-kanban-inspection");
      const heading = el("h3", selected.header.title);
      heading.tabIndex = -1;
      aside.append(
        heading,
        button(
          "Close item",
          () => {
            frame.selected = null;
            render();
            dialog
              .querySelector(`[data-kanban-ref="${key(selected.subject.ref)}"]`)
              ?.focus();
          },
          "quiet",
        ),
      );
      const content = el("div");
      aside.append(content);
      renderWorkspaceView(content, {
        view: selected,
        disclosures,
        onKanban: (view) => navigate(view),
        onReference: inspect,
        onActivity: async () => {
          if (frame.activityLoading) return;
          frame.activityLoading = true;
          const token = ++epoch;
          try {
            const view = await query(selected.subject, {
              include_activity: true,
            });
            if (valid(token)) {
              frame.selected = view;
              render();
            }
          } catch (error) {
            fail(error, token);
          } finally {
            frame.activityLoading = false;
          }
        },
        onVerify: (view) => {
          saveScroll();
          frames.push({ ...makeFrame(view), mode: "criteria" });
          render();
        },
        onMaterial: (view) => {
          saveScroll();
          frames.push({ ...makeFrame(view), mode: "materials" });
          render();
        },
        onCriterion: (definition, owner) =>
          query({ kind: "criterion", ref: owner, criterion_id: definition.id })
            .then((view) => {
              frame.selected = view;
              render();
            })
            .catch(onError),
        onFullRecord: () => act(null, null, selected),
        onNote: (entry, options) =>
          act("read_entity_note", { entry, options }, selected),
        onAction: (action, capability) => act(action, capability, selected),
      });
      const actions = el("div");
      aside.append(actions);
      renderWorkspaceActions(actions, {
        view: selected,
        primaryAlreadyPresent:
          selected.subject.kind === "entity" &&
          (["outcome", "objective"].includes(selected.subject.ref.kind) ||
            ["done", "cancelled", "achieved", "failed", "abandoned"].includes(
              selected.header.lifecycle,
            )),
        onAction: (action, capability) => act(action, capability, selected),
      });
      body.append(aside);
    }
    const footer = el("footer", undefined, "workspace-kanban-footer");
    if (frame.error) {
      const error = el("p", frame.error, "error");
      error.setAttribute("role", "alert");
      footer.append(error);
    }
    if (frame.coverage.next_cursor) {
      const more = button("Load more children", () => load(true), "quiet");
      more.disabled = frame.loading;
      footer.append(more);
    }
    if (!frame.mode) {
      const actions = el("div", undefined, "workspace-primary-actions");
      for (const capability of workspaceActions(frame.view)) {
        const name = workspaceCommandName(capability.action_id);
        const labels = {
          create_objective:
            frame.view.subject.ref.kind === "objective"
              ? "Add subobjective"
              : "Add Objective",
          create_work_item:
            frame.view.subject.ref.kind === "outcome"
              ? "Add direct Task"
              : "Add Task",
        };
        if (labels[name])
          actions.append(
            button(
              labels[name],
              () => act(capability.action_id, capability, frame.view),
              "quiet",
            ),
          );
      }
      actions.append(
        button(
          "Inspect parent and acceptance",
          () => {
            frame.selected = frame.view;
            render();
          },
          "quiet",
        ),
      );
      footer.append(actions);
    }
    dialog.replaceChildren(header, breadcrumb, controls, status, body, footer);
  }
  async function resume() {
    if (!suspended || sessionScope !== scopeKey()) return;
    suspended = false;
    current().selected = null;
    dialog.showModal();
    if (current().mode) {
      const token = ++epoch;
      try {
        const view = await query(current().view.subject);
        if (valid(token)) {
          current().view = view;
          render();
        }
      } catch (error) {
        fail(error, token);
      }
    } else await load();
  }
  return {
    open,
    close,
    reset,
    resume,
    get suspended() {
      return suspended;
    },
    get isOpen() {
      return dialog.open;
    },
  };
}
