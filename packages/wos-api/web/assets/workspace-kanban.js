import { el, button } from "./ui.js";
import { display } from "./presentation.js";
import { humanActions } from './actions.js';
import { kanbanOperation, kanbanLane, kanbanLanes } from './kanban-operations.js';
import {
  renderWorkspaceView,
  workspaceActions,
  workspaceCommandName,
} from "./workspace-view.js";

const key = (ref) => `${ref.kind}/${ref.id}`;

// A contextual modal over the spatial Board. Operations replace, never stack over, it.
// Every mutation leaves through the existing authorized command surface.
export function createContextualKanban({
  requestCards,
  requestTimeline,
  readWork,
  query,
  onAction,
  onReturn,
  scopeKey,
  onError,
  operationHost,
  workspaceHost,
  canLeaveOperation = () => true,
  operationState = () => 'editing',
  onOperationEnd,
} = {}) {
  const dialog = el("dialog", undefined, "workspace-kanban");
  dialog.hidden = true;
  const operationDialog = el("dialog", undefined, "workspace-kanban-operation");
  operationDialog.id = "workspace-kanban-operation";
  operationDialog.setAttribute("aria-labelledby", "workspace-transition-title");
  document.body.append(operationDialog);
  dialog.id = "workspace-kanban";
  dialog.setAttribute("aria-labelledby", "workspace-kanban-title");
  document.body.append(dialog);
  let frames = [],
    epoch = 0,
    sessionScope = "",
    originView,
    returnFocus,
    suspended = false;
  const disclosures = new Map();
  let operation = false, preparation = null, dock, proofReturn = null;
  if (operationHost) new MutationObserver(() => {
    if (operation) {
      const phase = operationState(); operationDialog.dataset.phase = phase;
      const exit = operationDialog.querySelector('[data-operation-exit]');
      if (exit) exit.disabled = ['submitting','uncertain'].includes(phase);
    }
  }).observe(operationHost, {subtree:true,childList:true,attributes:true,characterData:true});
  const active = () => !dialog.hidden;
  function setActive(value) {
    dialog.hidden = !value;
    if (value && !operationDialog.open && !dialog.open) dialog.showModal();
    if (!value && dialog.open) dialog.close();
    document.documentElement.dataset.kanbanActive = String(value);
    if (workspaceHost) {
      workspaceHost.dataset.kanbanActive = String(value);
      for (const child of workspaceHost.children) if (child !== dialog) child.inert = value;
      if (workspaceHost.contains(operationHost)) operationHost.inert = value;
    }
  }
  function restoreHost() {
    if (dock) { dock.parent.insertBefore(operationHost, dock.next?.parentNode === dock.parent ? dock.next : null); dock = null; }
    if (operationHost) operationHost.inert = active();
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
    const origin = originView || frames[0]?.view,
      scope = sessionScope,
      originTrigger = returnFocus;
    if (operationDialog.open) operationDialog.close();
    preparation = null; proofReturn = null;
    setActive(false);
    if (restore && origin && onReturn)
      Promise.resolve(onReturn(origin, originTrigger))
        .then((target) => {
          if (sessionScope !== scope || scopeKey() !== scope) return;
          const focus = originTrigger?.isConnected ? originTrigger : target;
          focus?.focus({ preventScroll: true });
        })
        .catch(onError);
  }
  function dismissOperation() {
    if (operation && !canLeaveOperation()) return;
    ++epoch;
    preparation = null;
    if (current()) current().intent = null;
    if (operation) finishOperation();
    else { operationDialog.close(); if (active() && !dialog.open) dialog.showModal(); focusSelection(); }
  }
  operationDialog.addEventListener('cancel', event => { event.preventDefault(); dismissOperation(); });
  operationDialog.addEventListener('keydown', event => {
    if (event.key !== 'Tab' || event.defaultPrevented) return;
    const controls = [...operationDialog.querySelectorAll('button,input,select,textarea,a[href],summary,[tabindex]')]
      .filter(node => node.tabIndex >= 0 && !node.disabled && !node.closest('[hidden],[inert]') && node.checkVisibility());
    const first = controls[0], last = controls.at(-1);
    if (!first) { event.preventDefault(); operationDialog.querySelector('h2')?.focus(); }
    else if (event.shiftKey && (document.activeElement === first || !operationDialog.contains(document.activeElement))) { event.preventDefault(); last.focus(); }
    else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
  });
  dialog.addEventListener('cancel', event => { event.preventDefault(); if (current()?.selected) backToWork(); else close(); });
  dialog.addEventListener('keydown', event => {
    if (event.key !== 'Escape' || event.defaultPrevented || event.isComposing || event.target.closest('select, [role="combobox"][aria-expanded="true"]')) return;
    const menu = event.target.closest('details[open].workspace-card-menu, details[open].workspace-kanban-filters');
    event.preventDefault(); event.stopPropagation();
    if (menu) { menu.open = false; menu.querySelector('summary').focus(); }
    else if (current()?.selected) backToWork();
    else close();
  });
  function focusSelection() {
    const frame = current();
    const ref = frame?.selected?.subject.ref || frame?.lastRef;
    const target = ref && dialog.querySelector(`[data-kanban-ref="${key(ref)}"]`);
    (target || dialog.querySelector('#workspace-kanban-title'))?.focus({preventScroll:true});
  }
  function reset() {
    close(false);
    frames = [];
    originView = null;
    disclosures.clear();
    dialog.replaceChildren();
    operationDialog.replaceChildren();
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
    originView = view;
    let parent = view;
    if (!['outcome','objective'].includes(view.subject.ref.kind)) {
      const token = ++epoch;
      const parentRef = view.placement?.parent?.ref || {...view.subject.ref,kind:'outcome',id:view.subject.ref.outcome_id};
      parent = await query({kind:'entity',ref:parentRef});
      if (!valid(token)) return;
    }
    frames = [makeFrame(parent)];
    if (mode) current().selected = view;
    setActive(true);
    render();
    await load();
    if (mode === 'criteria') {
      const group = dialog.querySelector('[data-operational-group=criteria]');
      if (group) group.open = true;
    }
    if (mode === 'materials') showPreparation(current().selected, {kind:'inspection',remedyGroup:'material',message:'Choose what you want to record. An Artifact is a deliverable; Evidence is an observation. Recording either does not approve a criterion. File upload is not available.'});
    if (operationDialog.open) return;
    dialog.querySelector("#workspace-kanban-title")?.focus({preventScroll:true});
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
    if (current() && !operation)
      current().horizontalScroll = dialog.querySelector('.workspace-kanban-columns')?.scrollLeft || 0;
    if (operation) return;
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
  function operationHeader(title, view, message) {
    const header = el('header', undefined, 'workspace-transition-header');
    const heading = el('h2', title); heading.id = 'workspace-transition-title'; heading.tabIndex = -1;
    const exit = button('Back to item', dismissOperation, 'quiet'); exit.dataset.operationExit = '';
    header.append(heading, exit);
    if (view) header.append(el('p', view.header.title));
    if (current()?.intent && view) {
      const lanes = kanbanLanes(view.subject.ref.kind);
      const origin = kanbanLane({ref:view.subject.ref,lifecycle:view.header.lifecycle,display_state:view.header.display_state});
      const label = id => lanes.find(([key]) => key === id)?.[1] || display(id);
      header.append(el('p', `Requested: ${label(origin)} → ${label(current().intent.stage)}. Confirmation and current requirements still apply.`, 'field-help'));
    }
    if (message) header.append(el('p', message, 'field-help'));
    return header;
  }
  async function move(ref, stage) {
    if (operation) return;
    saveScroll();
    const token = ++epoch;
    current().transitionState = 'preparing';
    render();
    try {
      const view = await query({kind:'entity', ref});
      if (!valid(token)) return;
      assertFrameRevision(view);
      current().selected = view;
      current().lastRef = ref;
      current().intent = {ref, stage};
      const intent = kanbanOperation(view, stage);
      current().transitionState = intent.kind;
      current().moveMessage = intent.message || '';
      render();
      if (intent.kind === 'command' || intent.kind === 'external_handoff') await act(intent.capability.action_id, intent.capability, view);
      else showPreparation(view, intent);
    } catch (error) { fail(error, token); }
  }
  function showPreparation(view, intent) {
    preparation = {view, intent};
    const title = intent.kind === 'inspection' ? 'Review proof' : intent.sequential ? 'Make Task available, then acquire' : 'Prepare operation';
    const body = el('div', undefined, 'workspace-transition-body');
    const feedback = el('p', intent.message, 'workspace-move-feedback'); feedback.setAttribute('role','status');
    body.append(feedback);
    for (const reason of intent.reasons || []) body.append(el('p', display(reason), 'workspace-requirement'));
    if (intent.review) {
      body.replaceChildren();
      renderProofChecklist(body, view);
      operationDialog.replaceChildren(operationHeader('Review proof', view), body);
      operationDialog.dataset.phase = 'inspection';
      if (dialog.open) dialog.close();
      if (!operationDialog.open) operationDialog.showModal();
      operationDialog.querySelector('h2').focus({preventScroll:true});
      return;
    }
    const actions = el('div', undefined, 'workspace-transition-remedies');
    const cap = intent.capability;
    if (cap && (cap.availability === 'available' || cap.transport === 'signed_host'))
      actions.append(button(cap.transport === 'signed_host' ? 'Continue in authorized host' : humanActions[workspaceCommandName(cap.action_id)]?.label || 'Continue', () => act(cap.action_id, cap, view), 'workspace-primary-action'));
    // Existing authorized reads and command forms are the only remedies.
    const review = el('div');
    const read = async subject => {
      const token = ++epoch;
      try { const next = await query(subject); if (valid(token)) { assertFrameRevision(next); showPreparation(next, {...intent, capability:null}); } }
      catch(error) { fail(error, token); }
    };
    renderWorkspaceView(review, {view, operational:'compact', disclosures,
      onCriterion:(definition, owner) => read({kind:'criterion',ref:owner,criterion_id:definition.id}),
      onSubject:read, onReference:ref => read({kind:'entity',ref}),
      onAction:(action, capability) => act(action, capability, view)});
    for (const group of review.querySelectorAll('[data-operational-group]'))
      if ((group.dataset.operationalGroup === 'criteria' && (intent.review || intent.reasons?.some(reason => /criterion|assessment|proof/.test(reason))))
        || (group.dataset.operationalGroup === 'impediments' && intent.reasons?.some(reason => /block|depend/.test(reason)))) group.open = true;
    const choices = el('details', undefined, 'workspace-remedy-actions'); choices.open = true;
    choices.append(el('summary', 'Resolve requirements'));
    for (const remedy of workspaceActions(view)) {
      if (remedy === cap || remedy.availability !== 'available' || remedy.transport === 'signed_host') continue;
      const name = workspaceCommandName(remedy.action_id);
      if ((intent.remedyGroup === 'material' ? /evidence|artifact/ : /criterion|assessment|evidence|artifact|assignee|blocker|issue|dependency/).test(name)) {
        const control = button(humanActions[name]?.label || display(name), () => act(remedy.action_id,remedy,view),'quiet');
        control.dataset.actionId = name; choices.append(control);
      }
    }
    actions.append(choices); body.append(actions, review);
    if (current().intent) body.append(button('Recheck intended operation', () => { const target = current().intent; if(target) move(target.ref,target.stage); },'quiet'));
    operationDialog.replaceChildren(operationHeader(title, view), body);
    operationDialog.dataset.phase = intent.kind;
    if (dialog.open) dialog.close();
    if (!operationDialog.open) operationDialog.showModal();
    operationDialog.querySelector('h2').focus({preventScroll:true});
  }
  function renderProofChecklist(body, view) {
    const work = el('section', undefined, 'workspace-proof-work');
    work.append(el('h3', 'What to do'));
    if (view.header.description) work.append(el('p', view.header.description));
    for (const [field, title] of [['instructions','Instructions'],['deliverables','Expected result'],['constraints','Constraints'],['scope_hints','Scope hints']]) {
      const values = view.work?.[field] || view.entity?.execution_spec?.[field];
      if (values?.length) { work.append(el('h4',title)); const list=el('ul'); for(const text of values) list.append(el('li',text)); work.append(list); }
    }
    if (work.children.length === 1) work.append(el('p', 'No additional instructions recorded.'));
    if (readWork && (view.coverage?.work?.status === 'partial' || view.entity?.description && view.entity.description !== view.header.description)) {
      work.append(el('p','Instructions are previewed. Read the complete definition before reviewing proof.','field-help'));
      const complete=button('Read complete instructions',async()=>{
        const token=++epoch; complete.disabled=true;
        try {
          const result=await readWork(view.subject.ref); if(!valid(token))return;
          if(String(result.outcome_revision)!==String(view.outcome_revision) || String(result.value.version)!==String(view.entity_version)) throw Error('The work changed. Return to the Kanban and refresh before reviewing its full definition.');
          showPreparation({...view,header:{...view.header,description:result.value.description},entity:result.value,work:result.value.execution_spec,coverage:{...view.coverage,work:{status:'complete'}}},{kind:'inspection',review:true});
        } catch(error) {if(valid(token)){const message=el('p',error.message,'error');message.setAttribute('role','alert');work.append(message);}}
        finally {complete.disabled=false;}
      },'quiet'); work.append(complete);
    }
    body.append(work);
    const list = el('section', undefined, 'workspace-proof-checklist');
    list.setAttribute('aria-label', 'Task criteria checklist');
    list.append(el('h3', 'Criteria & proof'), el('p', 'Link or add evidence for each criterion, then record its assessment. Evidence alone does not mark a criterion as met.', 'field-help'));
    const actionFor = async (definition, name) => {
      const token = ++epoch;
      try {
        const criterion = await query({kind:'criterion',ref:definition.owner_ref,criterion_id:definition.id});
        if (!valid(token)) return;
        assertFrameRevision(criterion);
        const capability = workspaceActions(criterion).find(cap => workspaceCommandName(cap.action_id) === name);
        if (!capability) { const message=el('p','This operation is not available for your permissions.','error'); message.setAttribute('role','alert'); list.append(message); return; }
        await act(capability.action_id,capability,criterion);
      } catch(error) { if(valid(token)) {const message=el('p',error.message,'error');message.setAttribute('role','alert');list.append(message);} }
    };
    for (const item of view.criteria || []) {
      const definition = item.definition;
      if (!definition || definition.status === 'retired') continue;
      const row = el('article',undefined,'workspace-proof-criterion'); row.dataset.criterionId=definition.id;
      const met = Boolean(item.assessment_applicable && item.assessment?.result === 'met');
      const assess = button(definition.title, () => actionFor(definition,'record_criterion_assessment'), 'quiet workspace-proof-check');
      assess.setAttribute('role','checkbox'); assess.setAttribute('aria-checked',String(met));
      assess.setAttribute('aria-label', `${definition.title}: ${met ? 'met' : 'not verified'}. Record assessment`);
      row.append(assess,el('small',definition.required?'Required':'Optional'));
      if (definition.description) row.append(el('p',definition.description));
      row.append(el('p', item.assessment_applicable && item.assessment ? display(item.assessment.result) : 'Not assessed', 'workspace-proof-result'));
      row.append(el('p', item.evidence_linked ? 'Evidence linked — review it in the assessment.' : 'No linked evidence in this snapshot.', 'field-help'));
      row.append(button('Add or link evidence',()=>actionFor(definition,'register_evidence_with_links'),'quiet'));
      const artifact=workspaceActions(view).find(cap=>workspaceCommandName(cap.action_id)==='register_artifact_in_context');
      if(artifact) row.append(button('Add artifact',()=>act(artifact.action_id,artifact,view),'quiet'));
      list.append(row);
    }
    if (!(view.criteria || []).some(item=>item.definition?.status!=='retired')) list.append(el('p','No active criteria recorded. The Task still requires an explicit result to complete.'));
    if (view.coverage?.criteria?.next_cursor) list.append(button('Load more criteria',async()=>{
      const token=++epoch;
      try { const next=await query(view.subject,{criteria_cursor:view.coverage.criteria.next_cursor});
        if(!valid(token))return; assertFrameRevision(next); next.criteria=[...(view.criteria||[]),...next.criteria]; showPreparation(next,{kind:'inspection',review:true});
      } catch(error){fail(error,token);}
    },'quiet'));
    else if(view.coverage?.criteria?.status!=='complete') list.append(el('p','Criteria coverage is incomplete; this is not a complete acceptance checklist.','field-help'));
    body.append(list);
  }
  async function act(action, capability, view) {
    if (operation) return;
    if (capability && capability.availability !== 'available' && capability.transport !== 'signed_host') {
      showPreparation(view, {kind:'needs_preparation',capability,reasons:capability.reason_codes || [],
        message:'No state changed. Resolve the reported requirements before confirming this operation.'});
      return;
    }
    saveScroll(); proofReturn = preparation?.intent.review ? preparation.view.subject : null; preparation = null;
    if (operationHost) {
      operation = true; ++epoch;
      dock ||= { parent: operationHost.parentNode, next: operationHost.nextSibling };
      operationHost.inert = false;
      const explanation = {
        activate_work_item:'Make this Task available. This neither assigns it nor acquires execution authority.',
        claim_work_item:'Acquire a time-limited reservation. The server checks readiness, dependencies, blockers and your authority.',
        complete_work_item:'Conclude this Task explicitly. Its Objective and Outcome keep their own acceptance decisions.',
        achieve_objective:'Accept this Objective against its current conditions. This does not complete child work or accept the Outcome.',
      }[workspaceCommandName(action)];
      operationDialog.replaceChildren(operationHeader(humanActions[workspaceCommandName(action)]?.label || 'Item operation', view, explanation), operationHost);
      dialog.dataset.operation = 'true';
      operationDialog.dataset.phase = 'editing';
      if (dialog.open) dialog.close();
    if (!operationDialog.open) operationDialog.showModal();
      try { await onAction(action, capability, view); }
      catch(error) { onError?.(error); await finishOperation(); }
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
    dialog.dataset.phase = frame.transitionState || 'idle';
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
    header.append(title, button("Back to Canvas", close, "quiet"));
    header.append(el("p", frame.view.header.title, "workspace-kanban-context"));
    if (frames.length > 1) header.append(button("Back to parent", () => {
      saveScroll(); ++epoch; frames.pop(); render();
    }, "quiet"));
    const controls = el("div", undefined, "workspace-kanban-controls");
    const contextLabel = el('label', 'Context');
    const contextSelect = el('select'); contextSelect.setAttribute('aria-label', 'Kanban context');
    for (const [value, label] of [['objectives', frame.view.subject.ref.kind === 'objective' ? 'Subobjectives' : 'Objectives'],
      ['tasks', frame.view.subject.ref.kind === 'outcome' ? 'Direct Tasks' : 'Tasks']]) {
      const option = el('option', label); option.value = value; contextSelect.append(option);
    }
    contextSelect.value = frame.content;
    contextSelect.addEventListener('change', () => {
      saveScroll(); frame.content = contextSelect.value; frame.selected = null; render();
    });
    contextLabel.append(contextSelect); controls.append(contextLabel);
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
    const stateKey = () => JSON.stringify([key(frame.view.subject.ref), frame.content, frame.revision,
      frame.search, frame.cancelled, frame.required, frame.priority, frame.assignee]);
    const previousColumns = dialog.querySelector('.workspace-kanban-columns');
    const keepColumns = previousColumns?.workspaceItems === frame.items && previousColumns?.dataset.stateKey === stateKey();
    const columns = keepColumns ? previousColumns : el("div", undefined, "workspace-kanban-columns");
    body.append(columns);
    const laneKinds = kanbanLanes(frame.content === "objectives" ? "objective" : "work_item");
    function renderColumns() {
      columns.workspaceItems = frame.items;
      columns.dataset.stateKey = stateKey();
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
        ...laneKinds,
        ...(frame.cancelled ? [["cancelled", "Cancelled / out of scope"]] : []),
      ]) {
        const section = el("section", undefined, "workspace-kanban-column");
        section.dataset.column = stage;
        const help = el('p', '', 'workspace-drop-meaning'); help.hidden = true;
        section.append(help);
        section.addEventListener('dragover', event => {
          if (event.dataTransfer?.types.includes('application/x-wos-kanban')) { event.preventDefault(); event.dataTransfer.dropEffect = 'move'; }
        });
        section.addEventListener('drop', event => {
          event.preventDefault();
          const item = frame.items.find(item => key(item.ref) === event.dataTransfer?.getData('application/x-wos-kanban'));
          if (item && kanbanLane(item) !== stage) frame.pendingDrop = {ref:item.ref,stage};
        });
        const matching = items.filter((item) => kanbanLane(item) === stage);
        section.append(el("h3", `${label} · ${matching.length} loaded`));
        for (const item of matching) {
          const card = button(
            "",
            () => inspect(item.ref),
            "workspace-kanban-card",
          );
          card.dataset.kanbanRef = key(item.ref);
          card.setAttribute('aria-pressed', String(frame.selected?.subject.ref.id === item.ref.id));
          card.draggable = true;
          card.setAttribute("aria-description", "Open to inspect; use the Actions menu to prepare a move without dragging.");
          card.append(el("span", "⠿", "workspace-drag-handle"));
          card.addEventListener('dragstart', event => {
            frame.dragged = key(item.ref);
            event.dataTransfer.setData('application/x-wos-kanban', key(item.ref));
            event.dataTransfer.effectAllowed = 'move';
            const setHint = (column, state, message) => {
              column.dataset.dropIntent = state;
              const hint = column.querySelector('.workspace-drop-meaning'); hint.hidden = false; hint.textContent = message;
            };
            for (const column of columns.querySelectorAll('[data-column]'))
              setHint(column, 'requires_preparation', column.dataset.column === kanbanLane(item) ? 'Current lane — no change' : 'Prepare operation · requirements checked on drop');
            const scope = sessionScope;
            query({kind:'entity',ref:item.ref}).then(view => {
              if (scope !== scopeKey() || frame.dragged !== key(item.ref)) return;
              for (const column of columns.querySelectorAll('[data-column]')) {
                if (column.dataset.column === kanbanLane(item)) continue;
                const intent = kanbanOperation(view,column.dataset.column);
                const state = !workspaceActions(view).length ? 'read_only' : intent.kind === 'command' ? 'eligible' : intent.kind;
                setHint(column,state,`${display(state)}${intent.capability ? ` · ${humanActions[workspaceCommandName(intent.capability.action_id)]?.label || 'Check current authority'}` : ''}`);
              }
            }).catch(() => { /* Optional hints never authorize or block a drop. */ });
          });
          let dragged = false;
          card.addEventListener('pointerdown', () => { dragged = false; });
          card.addEventListener('keydown', () => { dragged = false; });
          card.addEventListener('dragstart', () => { dragged = true; });
          card.addEventListener('click', event => { if (dragged && event.detail) { event.preventDefault(); event.stopImmediatePropagation(); } }, true);
          card.addEventListener('dragend', () => {
            frame.dragged = null;
            for (const column of columns.querySelectorAll('[data-column]')) {
              delete column.dataset.dropIntent;
              column.querySelector('.workspace-drop-meaning').hidden = true;
            }
            const pending = frame.pendingDrop; frame.pendingDrop = null;
            // Keep the native dragged node mounted until the browser finishes
            // the gesture. A synchronous render in drop can strand mouseup.
            if (pending) requestAnimationFrame(() => {
              if (active() && current() === frame && sessionScope === scopeKey()) move(pending.ref,pending.stage);
            });
          });
          card.append(
            el("strong", item.title),
            el("small", display(item.lifecycle)),
          );
          if (['high','critical'].includes(item.priority)) card.append(el('small', `${display(item.priority)} priority`));
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
          for (const person of (item.participants || []).slice(0, 1))
            card.append(
              el(
                "small",
                `Declared responsibility: ${person.actor_ref?.display_name || person.actor_ref?.id || "Not recorded"}`,
              ),
            );
          if (item.attention_reason)
            card.append(el("small", display(item.attention_reason)));
          const wrapper = el('article', undefined, 'workspace-kanban-card-shell');
          const menu = el('details', undefined, 'workspace-card-menu');
          const summary = el('summary', '…'); summary.setAttribute('aria-label', `Actions for ${item.title}`);
          menu.append(summary);
          const choices = el('div'); choices.setAttribute('aria-label', `Move ${item.title}`);
          for (const [value, title] of laneKinds) {
            if (value === kanbanLane(item)) continue;
            choices.append(button(`Move to ${title}`, () => { menu.open = false; move(item.ref, value); }, 'quiet'));
          }
          menu.append(choices); wrapper.append(card, menu); section.append(wrapper);

        }
        if (!matching.length)
          section.append(el("p", "No matching loaded items.", "field-help"));
        columns.append(section);
        section.scrollTop = frame.scroll.get(`${frame.content}/${stage}`) || 0;
      }
    }
    if (!keepColumns) renderColumns();
    for (const card of columns.querySelectorAll('[data-kanban-ref]'))
      card.setAttribute('aria-pressed', String(card.dataset.kanbanRef === (frame.selected && key(frame.selected.subject.ref))));
    if (frame.selected) {
      body.dataset.inspecting = "true";
      const selected = frame.selected,
        aside = el("aside", undefined, "workspace-kanban-inspection");
      aside.setAttribute("aria-label", "Item inspector");
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
          : selected.subject.ref.kind === 'objective' ? ['achieve_objective','start_objective']
            : selected.subject.ref.kind === 'issue' ? ['resolve_issue','investigate_issue']
              : selected.subject.ref.kind === 'blocker' ? ['resolve_blocker'] : ['activate_outcome','achieve_outcome'];
      const candidates = primaryNames.map(name => available.find(cap => workspaceCommandName(cap.action_id) === name)).filter(Boolean);
      let next = candidates.find(cap => cap.availability === 'available' || cap.transport === 'signed_host') || candidates[0];
      if (next?.reason_codes?.some(code => ['missing_required_criterion','required_active_criterion_missing'].includes(code)))
        next = available.find(cap => workspaceCommandName(cap.action_id) === 'add_criterion') || next;
      const continuation = !next && selected.subject.kind === 'entity'
        && selected.subject.ref.kind === 'work_item' && selected.entity?.contracts_enabled
        && selected.header.lifecycle === 'in_progress';
      if (next || continuation) {
        const guide = el('section', undefined, 'workspace-next-operation');
        guide.append(el('h4', 'Next operation'));
        const signed = continuation || next.transport === 'signed_host';
        const blocked = !signed && next.availability !== 'available';
        const reviewNeeded = blocked && (next.reason_codes || []).some(reason => /criterion|assessment|proof/.test(reason));
        guide.append(el('p', signed
          ? 'Continue acquisition, delivery or independent review in your authorized agent host. WOS records coordination; the browser does not sign or execute work.'
          : next.availability === 'available'
            ? 'Review and confirm the next operation. Assignment, execution, proof and acceptance are separate records.'
            : `Requirements are not satisfied: ${(next.reason_codes || []).map(display).join(', ')}. Inspect the impediments and acceptance criteria before continuing.`));
        const control = button(signed ? 'Continue in authorized host'
          : blocked ? reviewNeeded ? 'Review acceptance criteria' : 'Inspect requirements'
            : humanActions[workspaceCommandName(next.action_id)]?.label || 'Check requirements', () => {
          if (continuation) act('inspect_execution_handoff', null, selected);
          else if (reviewNeeded && selected.subject.ref.kind === 'work_item') showPreparation(selected,{kind:'inspection',review:true});
          else if (reviewNeeded) { const node = aside.querySelector('[data-operational-group=criteria]'); if(node) {node.open=true;node.scrollIntoView({block:'nearest'});} }
          else if (blocked) {
            const target = aside.querySelector('[data-workspace-facet="attention"], [data-workspace-facet="criteria"]');
            if (target) { target.tabIndex = -1; target.focus({preventScroll:true}); target.scrollIntoView({block:'start'}); }
          }
          else act(next.action_id, next, selected);
        }, 'workspace-primary-action');
        control.dataset.actionId = continuation ? 'inspect_execution_handoff' : blocked ? 'inspect_requirements' : workspaceCommandName(next.action_id);
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
        operational: "compact",
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
          const group = aside.querySelector('[data-operational-group=criteria]');
          if (group) { group.open = true; group.scrollIntoView({block:'nearest'}); }
        },
        onMaterial: (view) => {
          showPreparation(view, {kind:'inspection',remedyGroup:'material',message:'An Artifact is a produced deliverable; Evidence is an observation. Neither approves a criterion. File upload is not available.'});
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
    {
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
        if (labels[name] && name === (frame.content === "objectives" ? "create_objective" : "create_work_item"))
          actions.append(
            button(
              labels[name],
              () => act(capability.action_id, capability, frame.view),
              "quiet",
            ),
          );
      }
      controls.append(actions);
    }

    footer.hidden = !footer.children.length;
    dialog.replaceChildren(header, controls, status, body, footer);
    const detail = dialog.querySelector('.workspace-kanban-inspection');
    if (detail) detail.scrollTop = frame.detailScroll.get(detail.dataset.subject) || 0;
    dialog.querySelector('.workspace-kanban-columns').scrollLeft = frame.horizontalScroll || 0;
    const context = `${key(frame.view.subject.ref)}/${frame.content}`;
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
    setActive(true);
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
  async function finishOperation({committed = false} = {}) {
    if (!operation) return;
    const intended = current()?.intent;
    restoreHost();
    onOperationEnd?.();
    if (operationDialog.open) operationDialog.close();
    preparation = null;
    if (active() && !dialog.open) dialog.showModal();
    render();
    if (current().mode) {
      const token = ++epoch;
      try { const view = await query(current().view.subject); if (valid(token)) { current().view = view; current().selected = null; render(); } }
      catch(error) { fail(error, token); }
    } else await load();
    if (committed && current().error) {
      current().intent = null;
      current().error = 'Saved, but the refreshed view could not be loaded. Refresh reads the state; do not repeat the write.';
      render();
    } else if (committed && intended) {
      const card = current().items.find(item => key(item.ref) === key(intended.ref));
      if (card && kanbanLane(card) !== intended.stage) { await move(intended.ref, intended.stage); return; }
    }
    current().intent = null;
    if (proofReturn && !current().error) {
      const subject=proofReturn; proofReturn=null;
      const token=++epoch;
      try {const view=await query(subject); if(valid(token)){ assertFrameRevision(view); showPreparation(view,{kind:'inspection',review:true}); return; }} catch(error){fail(error,token);}
    }
    proofReturn=null;
    focusSelection();
  }
  return {
    open,
    close,
    reset,
    resume,
    finishOperation,
    get operating() { return operation || Boolean(preparation); },
    get suspended() {
      return suspended;
    },
    get isOpen() {
      return active();
    },
  };
}
