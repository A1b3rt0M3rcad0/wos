import { el, button } from "./ui.js";
import { display } from "./presentation.js";
import { humanActions } from './actions.js';
import { kanbanOperation, operationGroup } from './kanban-operations.js';
import {
  renderWorkspaceView,
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
  requestTimeline,
  query,
  onAction,
  onReturn,
  scopeKey,
  onError,
  operationHost,
  canLeaveOperation = () => true,
  onOperationEnd,
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
  let operation = false, dock;
  function restoreHost() {
    if (dock) { dock.parent.insertBefore(operationHost, dock.next?.parentNode === dock.parent ? dock.next : null); dock = null; }
    operation = false;
    delete dialog.dataset.operation;
  }
  const current = () => frames.at(-1);
  function selectView(view) {
    const frame = current();
    if (frame.selected) (frame.selectionStack ||= []).push(frame.selected);
    frame.selected = view;
  }
  function backToWork() {
    const frame = current(), ref = frame.selected?.subject.ref;
    ++epoch;
    frame.loading = false;
    frame.selected = frame.selectionStack?.pop() || null;
    frame.moveMessage = '';
    render();
    if (frame.selected) dialog.querySelector('.workspace-kanban-inspection h3')?.focus();
    else if (ref) dialog.querySelector(`[data-kanban-ref="${key(ref)}"]`)?.focus();
  }
  function valid(token) {
    return epoch === token && sessionScope === scopeKey();
  }
  function assertFrameRevision(view, frame = current()) {
    if (view.outcome_revision !== (frame.revision || frame.view.outcome_revision))
      throw new Error("The work changed. Refresh this Kanban before inspecting another revision.");
  }
  function fail(error, token) {
    if (!valid(token)) return;
    current().error =
      error.code === "precondition_failed"
        ? "The work changed. Refresh this Kanban before loading another page."
        : error.message || "Unable to load this view.";
    render();
  }
  function close(restore = true) {
    if (restore && operation && !canLeaveOperation()) return;
    restoreHost();
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
    if (operation) { if (canLeaveOperation()) finishOperation(); return; }
    if (current()?.selected) {
      backToWork();
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
    saveScroll();
    const frame = current(),
      token = ++epoch;
    frame.loading = true;
    frame.error = "";
    render();
    try {
      const [page, parent, selected, ancestors] = await Promise.all([
        requestCards({
          parent_ref: frame.view.subject.ref,
          limit: 50,
          ...(append ? { cursor: frame.coverage.next_cursor } : {}),
        }),
        append ? Promise.resolve(null) : query(frame.view.subject),
        !append && frame.selected
          ? query(frame.selected.subject)
          : Promise.resolve(null),
        !append ? Promise.all((frame.selectionStack || []).map(view => query(view.subject))) : Promise.resolve([]),
      ]);
      if ((parent && page.outcome_revision !== parent.outcome_revision) ||
          (selected && page.outcome_revision !== selected.outcome_revision) ||
          ancestors.some(view => view.outcome_revision !== page.outcome_revision))
        throw new Error(
          "The work changed during loading. Refresh to read one consistent revision.",
        );
      if (!valid(token)) return;
      if (append && page.outcome_revision !== frame.revision)
        throw new Error(
          "The work changed. Refresh this Kanban before loading another page.",
        );
      if (parent) frame.view = parent;
      if (selected) frame.selected = selected;
      if (!append) frame.selectionStack = ancestors;
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
      detailScroll: new Map(),
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
    if (key(current().view.subject.ref) === key(view.subject.ref)) {
      current().selected = null;
      current().mode = undefined;
      current().content =
        view.subject.ref.kind === "outcome" ? "objectives" : "tasks";
    } else frames.push(makeFrame(view));
    render();
    await load();
  }
  function saveScroll() {
    if (current() && !current().selected && !operation)
      current().horizontalScroll = dialog.querySelector('.workspace-kanban-columns')?.scrollLeft || 0;
    if (current()?.selected || operation) return;
    for (const node of dialog.querySelectorAll("[data-column]"))
      current()?.scroll.set(
        `${current().content}/${node.dataset.column}`,
        node.scrollTop,
      );
  }
  async function inspect(ref) {
    if (operation) return;
    saveScroll();
    const token = ++epoch;
    try {
      const view = await query({ kind: "entity", ref });
      if (!valid(token)) return;
      assertFrameRevision(view);
      selectView(view);
      current().moveMessage = '';
      render();
      dialog.querySelector(".workspace-kanban-inspection h3")?.focus();
    } catch (error) {
      fail(error, token);
    }
  }
  async function move(ref, stage) {
    if (operation) return;
    saveScroll();
    const token = ++epoch;
    try {
      const view = await query({ kind: 'entity', ref });
      if (!valid(token)) return;
      assertFrameRevision(view);
      current().selected = view;
      const intent = kanbanOperation(view, stage);
      current().moveMessage = intent.message || '';
      const cap = intent.capability;
      if (cap && cap.availability !== 'available' && cap.transport !== 'signed_host')
        current().moveMessage = `No state changed. ${(cap.reason_codes || []).map(display).join('. ')}. Review the impediments and requirements below.`;
      render();
      if (intent.review) {
        current().moveMessage += view.entity?.contracts_enabled
          ? ' Signed delivery and independent review must be completed in the authorized host.'
          : ' This Task has no separate submission stage; assess its criteria and record explicit completion when satisfied.';
        render();
      } else if (cap && (cap.availability === 'available' || cap.transport === 'signed_host')) {
        await act(cap.action_id, cap, view);
      }
      dialog.querySelector('.workspace-kanban-inspection h3')?.focus();
    } catch (error) { fail(error, token); }
  }
  async function act(action, capability, view) {
    if (operation) return;
    saveScroll();
    if (operationHost) {
      operation = true;
      ++epoch;
      dock ||= { parent: operationHost.parentNode, next: operationHost.nextSibling };
      const body = dialog.querySelector('.workspace-kanban-body');
      const surface = el('section', undefined, 'workspace-kanban-operation');
      surface.append(button('Back to item', () => { if (canLeaveOperation()) finishOperation(); }, 'quiet'));
      surface.append(operationHost);
      body.replaceChildren(surface);
      dialog.querySelector('.workspace-kanban-controls').hidden = true;
      dialog.querySelector('.workspace-kanban-footer').hidden = true;
      dialog.dataset.operation = 'true';
      try { await onAction(action, capability, view); }
      catch (error) { onError?.(error); await finishOperation(); }
      return;
    }
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
    if (operation) return;
    const focused = document.activeElement;
    const hadFocus = dialog.contains(focused);
    const previousContext = dialog.dataset.context;
    const selection =
      focused?.tagName === "INPUT"
        ? [focused.selectionStart, focused.selectionEnd]
        : null;
    if (!current() || sessionScope !== scopeKey()) return reset();
    const frame = current();
    const previousDetail = dialog.querySelector('.workspace-kanban-inspection');
    if (previousDetail?.dataset.subject)
      frame.detailScroll.set(previousDetail.dataset.subject, previousDetail.scrollTop);
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
    title.tabIndex = -1;
    header.append(title, button("Close Kanban", close, "quiet"));
    const breadcrumb = el("nav", undefined, "workspace-kanban-breadcrumb");
    breadcrumb.setAttribute("aria-label", "Kanban context");
    for (const [i, item] of frames.entries())
      breadcrumb.append(
        button(
          item.mode === "criteria"
            ? "Criteria review"
            : item.mode === "materials"
              ? "Materials"
              : item.view.header.title,
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
    const filterCount =
      Number(frame.cancelled) +
      Number(frame.required && frame.content === "objectives") +
      Number(Boolean(frame.priority)) +
      Number(Boolean(frame.assignee));
    filters.append(
      el(
        "summary",
        filterCount ? `Filters · ${filterCount} active` : "Filters",
      ),
    );
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
    cancelled.dataset.control = "cancelled";
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
        section.addEventListener('dragover', event => {
          if (event.dataTransfer?.types.includes('application/x-wos-kanban')) { event.preventDefault(); event.dataTransfer.dropEffect = 'move'; }
        });
        section.addEventListener('drop', event => {
          event.preventDefault();
          const item = frame.items.find(item => key(item.ref) === event.dataTransfer?.getData('application/x-wos-kanban'));
          if (item && item.stage !== stage) move(item.ref, stage);
        });
        const matching = items.filter((item) => item.stage === stage);
        section.append(el("h3", `${label} · ${matching.length} loaded`));
        for (const item of matching) {
          const card = button(
            "",
            () => inspect(item.ref),
            "workspace-kanban-card",
          );
          card.dataset.kanbanRef = key(item.ref);
          card.draggable = true;
          card.addEventListener('dragstart', event => {
            event.dataTransfer.setData('application/x-wos-kanban', key(item.ref));
            event.dataTransfer.effectAllowed = 'move';
          });
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
          if (item.criteria?.active)
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
          const label = el('label', 'Move to…', 'workspace-kanban-move');
          const target = el('select');
          target.setAttribute('aria-label', `Move ${item.title} to`);
          target.append(Object.assign(el('option', 'Choose an operation'), {value: ''}));
          for (const [value, title] of stages) {
            if (value !== item.stage) target.append(Object.assign(el('option', title), {value}));
          }
          target.addEventListener('change', () => { const value = target.value; target.value = ''; if (value) move(item.ref, value); });
          label.append(target); section.append(label);
        }
        if (!matching.length)
          section.append(el("p", "No matching loaded items.", "field-help"));
        columns.append(section);
        section.scrollTop = frame.scroll.get(`${frame.content}/${stage}`) || 0;
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
                  assertFrameRevision(view, frame);
                  selectView(view);
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
      if (frame.view.coverage?.criteria?.next_cursor) checklist.append(button('Load more criteria', async () => {
        const token = ++epoch;
        try {
          const next = await query(frame.view.subject, {criteria_cursor:frame.view.coverage.criteria.next_cursor});
          if (!valid(token)) return;
          assertFrameRevision(next, frame);
          frame.view.criteria.push(...next.criteria); frame.view.coverage.criteria = next.coverage.criteria; render();
        } catch (error) { fail(error, token); }
      }, 'quiet'));
      columns.replaceChildren(checklist);
      controls.hidden = true;
      status.hidden = true;
    } else renderColumns();
    if (frame.selected) {
      columns.hidden = true;
      const selected = frame.selected,
        aside = el("section", undefined, "workspace-kanban-inspection");
      aside.dataset.subject = JSON.stringify(selected.subject);
      const heading = el("h3", selected.header.title);
      heading.tabIndex = -1;
      aside.append(
        heading,
        button(
          frame.selectionStack?.length ? "Back to previous item" : "Back to Kanban",
          () => {
            backToWork();
          },
          "quiet",
        ),
      );
      const available = workspaceActions(selected);
      const primaryNames = selected.subject.kind === 'criterion' ? ['record_criterion_assessment']
        : selected.subject.ref.kind === 'work_item'
          ? selected.header.lifecycle === 'backlog' ? ['activate_work_item']
            : selected.header.lifecycle === 'todo' ? ['claim_work_item','acquire_signed_work_contract','acquire_work_contract']
              : ['reclaim_work_item','acquire_signed_review_contract','complete_work_item']
          : selected.subject.ref.kind === 'objective' ? ['start_objective','achieve_objective']
            : selected.subject.ref.kind === 'issue' ? ['investigate_issue','resolve_issue']
              : selected.subject.ref.kind === 'blocker' ? ['resolve_blocker'] : ['activate_outcome','achieve_outcome'];
      let next = available.find(cap => primaryNames.includes(workspaceCommandName(cap.action_id)));
      if (next?.reason_codes?.some(code => ['missing_required_criterion','required_active_criterion_missing'].includes(code)))
        next = available.find(cap => workspaceCommandName(cap.action_id) === 'add_criterion') || next;
      if (next) {
        const guide = el('section', undefined, 'workspace-next-operation');
        guide.append(el('h4', 'Next operation'));
        const signed = next.transport === 'signed_host';
        guide.append(el('p', signed
          ? 'Continue acquisition, delivery or independent review in your authorized agent host. WOS records coordination; the browser does not sign or execute work.'
          : next.availability === 'available'
            ? 'Review and confirm the next operation. Assignment, execution, proof and acceptance are separate records.'
            : `Requirements are not satisfied: ${(next.reason_codes || []).map(display).join(', ')}. Inspect the impediments and acceptance criteria before continuing.`));
        const control = button(signed ? 'Continue in authorized host' : humanActions[workspaceCommandName(next.action_id)]?.label || 'Check requirements', () => act(next.action_id, next, selected), 'workspace-primary-action');
        control.dataset.actionId = workspaceCommandName(next.action_id);
        control.inert = frame.loading;
        guide.append(control); aside.append(guide);
      }
      const content = el("div");
      content.inert = frame.loading;
      aside.append(content);
      if (frame.moveMessage) {
        const feedback = el('p', frame.moveMessage, 'workspace-move-feedback');
        feedback.setAttribute('role', 'status');
        aside.append(feedback);
      }
      renderWorkspaceView(content, {
        view: selected,
        operational: true,
        disclosures,
        onKanban: (view) => navigate(view),
        onReference: inspect,
        onSubject: async (subject) => {
          const token = ++epoch;
          try { const view = await query(subject); if (valid(token)) { assertFrameRevision(view, frame); selectView(view); render(); } }
          catch (error) { fail(error, token); }
        },
        onTimelineMore: requestTimeline ? async (page) => {
          const token = ++epoch;
          try {
            const next = await requestTimeline(selected.subject, page.next_cursor);
            if (!valid(token)) return;
            if (next.history_through_revision !== page.history_through_revision) throw new Error('History changed. Refresh this item before continuing.');
            selected.activity = {...next, items:[...(selected.activity?.items || []), ...next.items]};
            selected.coverage.activity = {status:next.next_cursor?'partial':'complete',next_cursor:next.next_cursor}; render();
          } catch (error) { fail(error, token); }
        } : undefined,
        onFacetMore: async (facet, cursor) => {
          const token = ++epoch;
          try {
            const name = {criteria:'criteria',roadmap:'roadmap',relationships:'relation',required_objectives:'required_objectives'}[facet] || 'relation';
            const next = await query(selected.subject, { [`${name}_cursor`]: cursor });
            if (!valid(token)) return;
            assertFrameRevision(next, frame);
            if (['criteria','required_objectives'].includes(facet)) selected[facet].push(...next[facet]);
            else if (facet === 'roadmap') selected.roadmap.revisions.push(...next.roadmap.revisions);
            else { selected.relationships.items.push(...next.relationships.items); selected.relationships.nodes.push(...next.relationships.nodes); }
            selected.coverage[facet] = next.coverage[facet]; render();
          } catch (error) { fail(error, token); }
        },
        onActivity: async () => {
          if (frame.activityLoading) return;
          frame.activityLoading = true;
          const token = ++epoch;
          try {
            const view = await query(selected.subject, {
              include_activity: true,
            });
            if (valid(token)) {
              assertFrameRevision(view, frame);
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
        onCriterion: async (definition, owner) => {
          const token = ++epoch;
          try {
            const view = await query({
              kind: "criterion",
              ref: owner,
              criterion_id: definition.id,
            });
            if (!valid(token)) return;
            assertFrameRevision(view, frame);
            selectView(view);
            render();
          } catch (error) {
            fail(error, token);
          }
        },
        onFullRecord: () => act(null, null, selected),
        onNote: (entry, options) =>
          act("read_entity_note", { entry, options }, selected),
        onAction: (action, capability) => act(action, capability, selected),
      });
      const actions = el("div", undefined, 'workspace-operation-groups');
      actions.inert = frame.loading;
      aside.append(actions);
      for (const label of ['Execute and conclude', 'Responsibility', 'Prove and verify', 'Resolve impediments', 'Plan and document']) {
        const group = el('section'); group.append(el('h4', label));
        for (const cap of workspaceActions(selected).filter(c => operationGroup(c) === label)) {
          const name = workspaceCommandName(cap.action_id);
          const control = button(humanActions[name]?.label || 'Continue in authorized agent host', () => act(cap.action_id, cap, selected), 'quiet');
          control.dataset.actionId = name;
          if (cap.availability !== 'available') control.append(el('small', cap.transport === 'signed_host' ? 'External authority required' : `Check requirements: ${(cap.reason_codes || []).map(display).join(', ')}`));
          group.append(control);
        }
        if (group.children.length > 1) actions.append(group);
      }
      actions.append(button('All item actions', () => act('more_actions', null, selected), 'quiet'));
      body.append(aside);
    }
    const footer = el("footer", undefined, "workspace-kanban-footer");
    if (frame.error) {
      const error = el("p", frame.error, "error");
      error.setAttribute("role", "alert");
      footer.append(error);
    }
    if (frame.coverage.next_cursor) {
      const more = button(
        "Load more children",
        () => {
          if (!frame.loading) load(true);
        },
        "quiet",
      );
      more.setAttribute("aria-disabled", String(frame.loading));
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
    const detail = dialog.querySelector('.workspace-kanban-inspection');
    if (detail) detail.scrollTop = frame.detailScroll.get(detail.dataset.subject) || 0;
    dialog.querySelector('.workspace-kanban-columns').scrollLeft = frame.horizontalScroll || 0;
    const context = `${key(frame.view.subject.ref)}/${frame.content}/${frame.mode || ""}`;
    dialog.dataset.context = context;
    // Restore after mounting: detached columns have no scroll geometry.
    for (const column of dialog.querySelectorAll("[data-column]"))
      column.scrollTop =
        frame.scroll.get(`${frame.content}/${column.dataset.column}`) || 0;
    if (hadFocus) {
      const controls = [
        ...dialog.querySelectorAll("button,input,select,summary"),
      ];
      let target;
      if (previousContext === context && focused !== dialog)
        target = controls.find((node) => {
          if (focused.dataset.control)
            return node.dataset.control === focused.dataset.control;
          if (focused.dataset.kanbanRef)
            return node.dataset.kanbanRef === focused.dataset.kanbanRef;
          const label = focused.getAttribute("aria-label");
          if (label)
            return (
              node.tagName === focused.tagName &&
              node.getAttribute("aria-label") === label
            );
          return (
            node.tagName === focused.tagName &&
            node.textContent === focused.textContent
          );
        });
      (target || title).focus({ preventScroll: true });
      if (
        target?.tagName === "INPUT" &&
        selection?.every((value) => value !== null)
      )
        target.setSelectionRange(...selection);
    }
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
  async function finishOperation() {
    if (!operation) return;
    restoreHost();
    onOperationEnd?.();
    delete dialog.dataset.operation;
    render();
    if (current().mode) {
      const token = ++epoch;
      try { const view = await query(current().view.subject); if (valid(token)) { current().view = view; current().selected = null; render(); } }
      catch (error) { fail(error, token); }
    } else await load();
    dialog.querySelector('.workspace-kanban-inspection h3, #workspace-kanban-title')?.focus({ preventScroll: true });
  }
  return {
    open,
    close,
    reset,
    resume,
    finishOperation,
    get operating() { return operation; },
    get suspended() {
      return suspended;
    },
    get isOpen() {
      return dialog.open;
    },
  };
}
