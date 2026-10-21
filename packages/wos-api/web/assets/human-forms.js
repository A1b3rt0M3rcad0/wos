import { message } from "./en-US.js";
import { copy } from "./en-US.js";
import {
  externalContextEditor,
  participantEditor,
} from "./advanced-editors.js";
import { el, inputField, listField, details } from "./ui.js";
import { display } from "./presentation.js";
import { referencePicker, referenceKinds } from "./reference-picker.js";
import { roadmapEditor, roadmapDiff } from "./roadmap-editor.js";
import {
  criteriaEditor, criterionModes, executionSpecEditor, localDateTime, placementField,
} from "./workspace-form-fields.js";
import { planEditor } from "./workspace-plan-editor.js";
import { contextualEvidenceForm } from "./workspace-evidence-form.js";
import { conclusionImpactEditor } from "./workspace-conclusion-form.js";
import { blockingImpactEditor } from "./workspace-blocking-form.js";
import { structuralMoveEditor } from "./structural-form-preview.js";
const priority = ["normal", "high", "critical", "low"].map((x) => [
  x,
  display(x),
]);
export function humanForm(
  name,
  {
    scope,
    entity = {},
    criterion,
    request,
    permissions = [],
    protocol,
    submission,
    contractView,
    mode = "full",
    contextTitle,
    outcomeTitle,
    eligibility,
    workspaceView,
    targetLifecycle,
    batchKind = "work_item",
    note,
  },
) {
  const node = el("div", undefined, "human-form"),
    advanced = details(copy.text.advanced),
    getters = {},
    fixed = { scope: { ...scope } },
    disposers = [];
  const kind = entity._kind || "outcome";
  const ref = { ...scope, kind, id: entity.id || scope.outcome_id };
  let commandName = () => name;
  let conclusionEditor;
  function f(key, label, options = {}) {
    const field = inputField(label, options);
    node.append(field.node);
    getters[key] = field.get;
    return field;
  }
  function a(key, label, options = {}) {
    const field = inputField(label, options);
    advanced.append(field.node);
    getters[key] = field.get;
    return field;
  }
  function p(key, label, kinds, opts = {}) {
    const picker = referencePicker({ label, kinds, scope, request, ...opts });
    node.append(picker.node);
    disposers.push(picker.dispose);
    getters[key] = () => {
      const v = picker.get();
      return opts.ids ? (Array.isArray(v) ? v.map((x) => x.id) : v?.id) : v;
    };
    return picker;
  }
  function id(k = kind) {
    if (k !== "outcome") fixed[`${k}_id`] = entity.id;
    fixed.expected_version = entity.version;
  }
  function exactObservedVersion() {
    const value = workspaceView?.entity_version ?? entity.version;
    if (typeof value === "string" && /^[1-9][0-9]*$/.test(value)) return value;
    if (Number.isSafeInteger(value) && value > 0) return String(value);
    throw Error(copy.text.formReviewTargets);
  }
  function title(label = copy.text.title, value) {
    f("title", label, { required: true, value });
  }
  function description(value) {
    f("description", copy.text.description, { type: "textarea", value });
  }
  function reason(label = copy.text.reason) {
    f("reason", label, { type: "textarea", required: true });
  }
  function criteriaOwner() {
    fixed.owner = ref;
    fixed.expected_version = entity.version;
  }
  function initialCriteria(parent = node) {
    const editor = criteriaEditor();
    parent.append(editor.node);
    getters.initial_criteria = () => {
      const value = editor.get();
      return value.length ? value : undefined;
    };
  }
  function placement(key, parent, label, parentRequired = false) {
    const creating = name === "create_objective" || name === "create_work_item";
    const editor = placementField({
      scope, request, parent, label, outcomeTitle,
      title: creating && kind === "objective" ? entity.title
        : workspaceView?.placement?.parent?.title || workspaceView?.placement?.objective?.title,
      allowChange: mode !== "quick" || (parentRequired && !parent), parentRequired,
    });
    node.append(editor.node);
    getters[key] = editor.get;
    disposers.push(editor.dispose);
  }
  function contextualTarget(label = copy.text.formCurrentPlacement) {
    const node = el("p", undefined, "field-context");
    node.append(el("small", label), el("strong", contextTitle || entity.title || entity.name || display(kind)));
    return node;
  }
  function nestFields(key, names) {
    const fields = names.map((name) => [name, getters[name]]).filter(([, get]) => get);
    for (const [name] of fields) delete getters[name];
    getters[key] = () => Object.fromEntries(fields.map(([name, get]) => [name, get()]));
  }
  const help = el("p", undefined, "flow-help");
  node.append(help);
  switch (name) {
    case "achieve_outcome":
    case "achieve_objective": {
      id();
      conclusionEditor = conclusionImpactEditor(workspaceView);
      node.append(conclusionEditor.node);
      getters._editor = () => conclusionEditor.get();
      reason();
      help.textContent = copy.text.thisIsAnExplicitLifecycleDecisionTheServerChecks;
      break;
    }
    case "replace_external_context": {
      id("outcome");
      const editor = externalContextEditor(entity.external_context || {});
      node.append(editor.node);
      getters.external_context = editor.get;
      break;
    }
    case "set_work_item_assignees": {
      id("work_item");
      const editor = participantEditor(entity.assignee_refs || []);
      node.append(editor.node);
      getters.assignee_refs = editor.get;
      break;
    }
    case "set_outcome_owners":
    case "set_objective_owners": {
      id(kind);
      const editor = participantEditor(entity.owner_refs || []);
      node.append(editor.node);
      getters.owner_refs = editor.get;
      break;
    }

    case "create_outcome":
      delete fixed.scope;
      fixed.namespace_id = scope.namespace_id;
      title(copy.text.outcomeName);
      f("desired_state", copy.text.whatDoesSuccessLookLike, {
        type: "textarea",
        required: true,
      });
      a("description", copy.text.description, { type: "textarea" });
      f("completion_policy", "Completion policy", {options: [["strict_work", "Strict work — required work must be resolved"], ["compatible", "Compatible — criteria and existing completion rules"]], value: "strict_work",
        help: "Strict work also checks Tasks and subobjectives. Cancelled work requires an explicit scope acknowledgement; it is never counted as accepted."});
      initialCriteria();
      a("priority", copy.text.priority, { options: priority, value: "normal" });
      help.textContent = copy.text.anOutcomeIsAResultToVerifySaveA;
      break;
    case "update_outcome":
      fixed.expected_version = entity.version;
      title(copy.text.outcomeName, entity.title);
      f("desired_state", copy.text.whatDoesSuccessLookLike, {
        type: "textarea",
        required: true,
        value: entity.desired_state,
      });
      description(entity.description);
      a("priority", copy.text.priority, {
        options: priority,
        value: entity.priority,
      });
      break;
    case "create_objective": {
      title(copy.text.objectiveTitle);
      placement("parent_objective_id", kind === "objective" ? entity.id : null);
      f("required_for_outcome", copy.text.requiredForOutcome, {
        type: "checkbox",
        value: true,
        help: copy.text.formRequiredOutcomeHelp,
      });
      const extra = details(copy.text.formDetails);
      const text = f("description", copy.text.description, { type: "textarea" });
      extra.append(text.node);
      initialCriteria(extra);
      node.append(extra);
      a("priority", copy.text.priority, { options: priority, value: "normal" });
      help.textContent =
        copy.text.anObjectiveDescribesAnIntermediateConditionNotAnExecution;
      break;
    }
    case "update_objective":
      id();
      title(copy.text.objectiveTitle, entity.title);
      description(entity.description);
      f("required_for_outcome", copy.text.requiredForOutcome, {
        type: "checkbox",
        value: entity.required_for_outcome,
      });
      a("priority", copy.text.priority, {
        options: priority,
        value: entity.priority,
      });
      break;
    case "create_work_item": {
      title(copy.text.taskTitle);
      placement("objective_id", kind === "objective" ? entity.id : null, undefined, true);
      node.append(el("p", copy.text.formPlannedTask, "status-badge"),
        el("p", copy.text.formPlannedTaskHelp, "field-help"));
      const extra = details(copy.text.formDetails);
      const text = f("description", copy.text.description, { type: "textarea" });
      extra.append(text.node);
      initialCriteria(extra);
      const taskPriority = a("priority", copy.text.priority, {
        options: priority,
        value: "normal",
      });
      extra.append(taskPriority.node);
      fixed.lifecycle = targetLifecycle || "backlog";
      if (mode !== "quick" && !targetLifecycle) {
        const available = inputField(copy.text.formCreateAvailable, {
          type: "checkbox", value: false, help: copy.text.formCreateAvailableHelp,
        });
        advanced.append(available.node);
        getters.lifecycle = () => available.get() ? "todo" : "backlog";
      }
      const spec = executionSpecEditor({ scope, request });
      extra.append(spec.node);
      disposers.push(spec.dispose);
      node.append(extra);
      getters.execution_spec = () => {
        const value = spec.get();
        return Object.values(value).some((v) => v.length) ? value : undefined;
      };
      const date = a("not_before", copy.text.scheduledStartLocalTime, {
        type: "datetime-local",
        help: copy.text.theTaskRemainsScheduledUntilThisTime,
      });
      getters.not_before = () =>
        date.get() ? new Date(date.get()).toISOString() : undefined;
      help.textContent =
        copy.text.aTaskDescribesBoundedWorkCompletingItNeverCertifies;
      break;
    }
    case "create_plan_items": {
      const editor = planEditor({
        kind: kind === "outcome" ? "objective" : batchKind, parent: kind === "objective" ? entity.id : undefined,
        parentTitle: contextTitle || entity.title, outcomeTitle,
      });
      node.append(editor.node);
      getters.items = editor.get;
      break;
    }
    case "set_work_item_execution_spec": {
      id("work_item");
      fixed.expected_version = exactObservedVersion();
      const editor = executionSpecEditor({ initial: entity.execution_spec, scope, request });
      node.append(editor.node);
      getters.execution_spec = editor.get;
      disposers.push(editor.dispose);
      help.textContent = copy.text.formReplaceSpecHelp;
      break;
    }
    case "set_work_item_not_before": {
      id("work_item");
      fixed.expected_version = exactObservedVersion();
      const date = f("not_before", copy.text.scheduledStartLocalTime, {
        type: "datetime-local", value: localDateTime(entity.not_before),
      });
      getters.not_before = () => date.get() ? new Date(date.get()).toISOString() : null;
      help.textContent = copy.text.formClearDateHelp;
      break;
    }
    case "set_work_item_objective":
    case "set_objective_parent": {
      id(kind);
      const editor = structuralMoveEditor({ scope, entity, kind, workspaceView, request, outcomeTitle });
      node.append(editor.node);
      getters._editor = editor.get;
      disposers.push(editor.dispose);
      help.textContent = kind === "work_item" ? copy.text.formMoveTaskHelp : copy.text.formMoveObjectiveHelp;
      break;
    }
    case "update_work_item":
      id();
      title(copy.text.taskTitle, entity.title);
      description(entity.description);
      f("priority", copy.text.priority, {
        options: priority,
        value: entity.priority,
      });
      break;
    case "add_criterion":
      criteriaOwner();
      delete fixed.scope;
      title(copy.text.formSuccessQuestion);
      description();
      f("required", copy.text.required, { type: "checkbox", value: true });
      f("verification_mode", copy.text.howWillItBeVerified, {
        options: criterionModes(), value: "evidence_review",
      });
      help.textContent =
        copy.text.requiredCriteriaMustBeAssessedBeforeAnExplicitConclusion;
      break;
    case "revise_criterion":
    case "retire_criterion":
      criteriaOwner();
      delete fixed.scope;
      fixed.criterion_id = criterion.id;
      fixed.criterion_revision = criterion.criterion_revision;
      if (name === "revise_criterion") {
        title(copy.text.criterionTitle, criterion.title);
        description(criterion.description);
        f("required", copy.text.required, {
          type: "checkbox",
          value: criterion.required,
        });
        f("verification_mode", copy.text.howWillItBeVerified, {
          value: criterion.verification_mode,
          options: criterionModes(),
        });
      }
      reason();
      break;
    case "record_criterion_assessment":
    case "attest_criterion": {
      if (submission?.id) {
        fixed.submission_id = submission.id;
        node.append(
          el(
            "p",
            message(
              "reviewingSubmission",
              submission.summary ||
                submission.material?.summary ||
                submission.id,
            ),
          ),
        );
      }
      criteriaOwner();
      delete fixed.scope;
      fixed.criterion_id = criterion.id;
      fixed.criterion_revision = criterion.criterion_revision;
      help.textContent = message(
        "assessingDefinitionRevisionAWaiverIsNotVerification",
        criterion.title,
        criterion.criterion_revision,
      );
      const results = [
        ["met", copy.text.met],
        ["not_met", copy.text.notMet],
        ["inconclusive", copy.text.inconclusive],
      ];
      const result = f("result", copy.text.assessmentResult, { options: results });
      if (permissions.includes("assessment:waive")) {
        const waiver = a("_waiver", "Record an authorized waiver instead of verification", {
          type:"checkbox", help:"An exception requires a rationale. It is not a met criterion or verified success.",
        });
        getters.result = () => waiver.get() ? "waived" : result.get();
        waiver.input.addEventListener("change", () => { result.node.hidden=Boolean(waiver.get()); });
      }
      f("rationale", copy.text.rationale, { type: "textarea", required: true });
      p("evidence_ids", copy.text.evidenceUsed, ["evidence"], {
        multiple: true,
        ids: true,
      });
      if (criterion.verification_mode === "external_evaluation") {
        const provider = f("_evaluator_provider", copy.text.evaluatorProvider, {
            required: true,
          }),
          id = f("_evaluator_id", copy.text.evaluatorIdentity, {
            required: true,
          }),
          version = f("_evaluator_version", copy.text.evaluatorVersion, {
            required: true,
          });
        getters.evaluator_ref = () => ({
          provider: provider.get(),
          id: id.get(),
          version: version.get(),
        });
      }
      break;
    }
    case "append_entity_note": {
      fixed.subject_ref = ref;
      if (criterion?.id) fixed.criterion_id = criterion.id;
      if (note?.id) fixed.supersedes_note_id = note.id;
      node.append(contextualTarget());
      f("type", copy.text.formNoteKind, {
        options: [
          ["update", copy.text.formNoteUpdate], ["discovery", copy.text.formNoteDiscovery],
          ["observation", copy.text.formNoteObservation],
        ], value: note?.type || "update",
      });
      const body = f("body", copy.text.formNoteBody, {
        type: "textarea", required: true, value: note?.body,
      });
      body.input.rows = 5;
      getters.body = () => {
        const value = body.get();
        if (new TextEncoder().encode(value || "").length > 16384)
          throw Error(copy.text.formNoteLimit);
        return value;
      };
      const occurred = a("occurred_at", copy.text.formOccurredAt, {
        type: "datetime-local", value: localDateTime(note?.occurred_at),
      });
      getters.occurred_at = () => occurred.get() ? new Date(occurred.get()).toISOString() : undefined;
      for (const [key, label, recordKind] of [
        ["artifact_refs", copy.text.formReferencedArtifacts, "artifact"],
        ["evidence_refs", copy.text.formReferencedEvidence, "evidence"],
      ]) {
        const picker = p(key, label, [recordKind], { multiple: true, ids: true });
        advanced.append(picker.node);
      }
      help.textContent = copy.text.formNoteHelp;
      break;
    }
    case "register_evidence": {
      f("evidence_type", copy.text.evidenceType, {
        options: [
          "test_result",
          "inspection",
          "measurement",
          "attestation",
          "source",
          "external_evaluation",
        ].map((x) => [x, display(x)]),
      });
      description();
      const provider = f("_provider", copy.text.sourceProvider, {
          required: true,
        }),
        uri = f("_uri", copy.text.sourceURL, { type: "url" }),
        external = f("_source_id", copy.text.sourceIdentifier);
      node.append(
        el(
          "p",
          copy.text.provideASourceURLOrExternalIdentifierAtLeast,
          "field-help",
        ),
      );
      getters.source_ref = () => {
        if (!uri.get() && !external.get())
          throw Error(copy.text.provideASourceURLOrExternalSourceIdentifier);
        return { provider: provider.get(), uri: uri.get(), id: external.get() };
      };
      const captured = f("captured_at", copy.text.observedAtLocalTime, {
        required: true,
        type: "datetime-local",
        value: new Date(Date.now() - new Date().getTimezoneOffset() * 60000)
          .toISOString()
          .slice(0, 16),
      });
      getters.captured_at = () => new Date(captured.get()).toISOString();
      p("artifact_id", copy.text.supportingArtifact, ["artifact"], {
        ids: true,
      });
      const measurement = details(copy.text.measurementDetails);
      const fields = {};
      for (const [key, label] of Object.entries({
        value: copy.text.measuredValue,
        unit: copy.text.unit,
        method: copy.text.method,
        conditions: copy.text.conditions,
      })) {
        const v = inputField(label);
        measurement.append(v.node);
        fields[key] = v.get;
      }
      node.append(measurement);
      getters.measurement = () =>
        getters.evidence_type() === "measurement"
          ? Object.fromEntries(Object.entries(fields).map(([k, g]) => [k, g()]))
          : undefined;
      help.textContent =
        copy.text.registerAnObservationHereAssessASuccessCriterionSeparately;
      break;
    }
    case "register_evidence_with_links": {
      const editor = contextualEvidenceForm({ scope, entity, criterion, request,
        contextTitle, eligibility: eligibility || workspaceView?.eligibility });
      node.append(editor.node);
      getters._editor = editor.get;
      disposers.push(editor.dispose);
      break;
    }
    case "register_artifact":
    case "register_artifact_in_context":
      if (name === "register_artifact_in_context") {
        fixed.subject_ref = ref;
        node.append(contextualTarget());
        f("relationship", copy.text.formContextLink, {
          options: [
            ["relates_to", copy.text.formContextArtifact],
            ...(["outcome", "objective", "work_item"].includes(kind)
              ? [["produces", copy.text.formProducedArtifact]] : []),
          ], value: "relates_to",
        });
        help.textContent = copy.text.formContextRecordHelp;
      }
      f("name", copy.text.artifactName, { required: true });
      f("artifact_type", copy.text.artifactType, { required: true });
      f("uri", "URL", { type: "url", required: true });
      if (name === "register_artifact_in_context")
        nestFields("artifact", ["name", "artifact_type", "uri"]);
      break;
    case "create_evidence_link":
      fixed.evidence_id = kind === "evidence" ? entity.id : undefined;
      if (!fixed.evidence_id)
        p("evidence_id", copy.text.evidence, ["evidence"], {
          ids: true,
          required: true,
        });
      {
        const kinds = eligibility?.evidence_link_target_kinds
          || workspaceView?.eligibility?.evidence_link_target_kinds || [];
        if (kinds.includes(kind)) {
          fixed.target_ref = ref;
          node.append(contextualTarget(copy.text.relatedItem));
          if (criterion?.id) fixed.criterion_id = criterion.id;
        } else {
          const picker = p("target_ref", copy.text.relatedItem, kinds, { required: true });
          if (!kinds.length) node.append(el("p", copy.text.formEvidenceTargetsUnavailable, "field-help"));
          getters.target_ref = () => {
            if (!kinds.length) throw Error(copy.text.formEvidenceTargetsUnavailable);
            return picker.get();
          };
        }
      }
      f("stance", copy.text.evidenceStance, {
        options: ["supports", "contradicts", "context"].map((x) => [
          x,
          x === "context" ? copy.text.formEvidenceContext : display(x),
        ]),
        value: criterion ? "supports" : "context",
      });
      f("rationale", copy.text.rationale, { required: true, type: "textarea" });
      help.textContent =
        copy.text.linkingEvidenceRecordsARelationshipItDoesNotAssess;
      break;
    case "create_issue":
    case "report_issue_with_blocker": {
      title(copy.text.issueTitle);
      description();
      f("severity", copy.text.severity, {
        options: ["major", "critical", "minor", "informational"].map((x) => [
          x,
          display(x),
        ]),
      });
      const isContextual = ["outcome", "objective", "work_item"].includes(kind);
      if (isContextual) {
        node.append(contextualTarget(copy.text.affectedItems));
        fixed.affected_refs = [ref];
      } else p("affected_refs", copy.text.affectedItems,
        ["outcome", "objective", "work_item"], { multiple: true });
      const blocks = f("_blocks", copy.text.formPreventsContinuing, {
        type: "checkbox", value: name === "report_issue_with_blocker",
      });
      const impact = el("section", undefined, "nested-editor workspace-issue-impact");
      let target;
      if (!isContextual) {
        target = p("_blocked_ref", copy.text.blockedItem,
          ["outcome", "objective", "work_item"]);
        impact.append(target.node);
      }
      const impactDescription = inputField(copy.text.blockingImpact, { type: "textarea" });
      const propagation = inputField(copy.text.propagation, { options: [] });
      const propagationMessage = el("p", undefined, "field-help");
      impact.append(impactDescription.node, propagation.node, propagationMessage);
      const blockingPreview = blockingImpactEditor({ scope, request,
        getTarget: () => isContextual ? ref : target?.get(), getPropagation: propagation.get });
      impact.append(blockingPreview.node);
      disposers.push(blockingPreview.dispose);
      getters._editor = () => blocks.get() ? blockingPreview.get() : {};
      node.append(impact);
      const propagationByKind = eligibility?.blocker_propagation
        || workspaceView?.eligibility?.blocker_propagation;
      function updateImpact() {
        const blockedKind = isContextual ? kind : target?.get()?.kind;
        const allowed = propagationByKind?.[blockedKind] || [];
        const previous = propagation.get();
        propagation.input.replaceChildren(...allowed.map((value) =>
          Object.assign(el("option", value === "direct" ? copy.text.onlyThisItem
            : copy.text.thisItemAndItsDescendants), { value })));
        if (allowed.includes(previous)) propagation.input.value = previous;
        propagationMessage.textContent = allowed.length ? "" : copy.text.formPropagationUnavailable;
        impact.hidden = !blocks.get();
        impact.inert = !blocks.get();
        impactDescription.input.required = blocks.get();
        propagation.input.required = blocks.get();
        blockingPreview.sync();
      }
      blocks.input.addEventListener("change", updateImpact);
      propagation.input.addEventListener("change", () => blockingPreview.sync());
      target?.node.addEventListener("click", updateImpact);
      target?.node.addEventListener("keydown", () => queueMicrotask(updateImpact));
      updateImpact();
      commandName = () => blocks.get() ? "report_issue_with_blocker" : "create_issue";
      getters.blocked_ref = () => blocks.get() ? isContextual ? ref : target.get() : undefined;
      getters.blocker_description = () => blocks.get() ? impactDescription.get() : undefined;
      getters.propagation = () => {
        if (!blocks.get()) return undefined;
        const blockedKind = isContextual ? kind : target?.get()?.kind;
        if (!propagationByKind?.[blockedKind]?.includes(propagation.get()))
          throw Error(copy.text.formPropagationUnavailable);
        return propagation.get();
      };
      help.textContent = copy.text.formIssueContextHelp;
      if (["done", "achieved", "cancelled", "failed", "abandoned"].includes(entity.lifecycle))
        node.append(el("p", copy.text.formIssueTerminalHelp, "field-help"));
      break;
    }
    case "create_blocker": {
      const byKind = eligibility?.blocker_propagation || workspaceView?.eligibility?.blocker_propagation || {};
      const contextual = Object.hasOwn(byKind, kind);
      let picker;
      if (contextual) {
        fixed.blocked_ref = ref;
        node.append(contextualTarget(copy.text.blockedItem));
      } else picker = p("blocked_ref", copy.text.blockedItem, Object.keys(byKind), { required: true });
      description();
      p("cause_ref", copy.text.blockingCause, [
        "issue",
        "work_item",
        "objective",
        "decision",
      ]);
      const propagation = f("propagation", copy.text.propagation, { options: [] });
      const warning = el("p", undefined, "field-help");
      node.append(warning);
      function selectedKind() {
        try { return contextual ? kind : picker?.get()?.kind; } catch { return undefined; }
      }
      function syncPropagation() {
        const previous = propagation.get();
        const values = byKind[selectedKind()] || [];
        propagation.input.replaceChildren(...values.map((value) =>
          Object.assign(el("option", value === "direct" ? copy.text.onlyThisItem
            : copy.text.thisItemAndItsDescendants), { value })));
        if (values.includes(previous)) propagation.input.value = previous;
        warning.textContent = values.length ? "" : copy.text.formPropagationUnavailable;
      }
      picker?.node.addEventListener("click", syncPropagation);
      picker?.node.addEventListener("keydown", () => queueMicrotask(syncPropagation));
      syncPropagation();
      getters.propagation = () => {
        if (!byKind[selectedKind()]?.includes(propagation.get()))
          throw Error(copy.text.formPropagationUnavailable);
        return propagation.get();
      };
      help.textContent = copy.text.aBlockerRecordsImpactOnWorkResolvingItsIssue;
      break;
    }
    case "update_issue":
      id();
      title(copy.text.issueTitle, entity.title);
      description(entity.description);
      f("severity", copy.text.severity, {
        value: entity.severity,
        options: ["major", "critical", "minor", "informational"].map((x) => [
          x,
          display(x),
        ]),
      });
      break;
    case "update_blocker_description":
      id("blocker");
      description(entity.description);
      break;
    case "resolve_issue":
    case "resolve_blocker":
      id();
      f("resolution_summary", copy.text.resolutionSummary, {
        type: "textarea",
        required: true,
      });
      if (name === "resolve_blocker")
        f("release_confirmed", copy.text.confirmWorkCanProceed, {
          type: "checkbox",
          value: false,
        });
      help.textContent =
        name === "resolve_issue"
          ? copy.text.resolvesTheIssueOnlyActiveBlockersRemainUntilExplicitly
          : copy.text.releasesThisImpactOnlyItsCauseMayRemainUnresolved;
      break;
    case "resolve_issue_and_blockers": {
      fixed.issue_id = entity.id;
      fixed.issue_expected_version = entity.version;
      f("issue_resolution_summary", copy.text.issueResolutionSummary, {
        required: true,
        type: "textarea",
      });
      const contextBlockers = [];
      const known = new Set();
      const refs = (workspaceView?.relationships?.items || [])
        .filter((edge) => edge.kind === "causes" && edge.source_ref?.kind === "issue"
          && edge.source_ref.id === entity.id && edge.target_ref?.kind === "blocker"
          && edge.target_ref.namespace_id === scope.namespace_id
          && edge.target_ref.outcome_id === scope.outcome_id)
        .map((edge) => edge.target_ref).filter((ref) => {
          if (known.has(ref.id)) return false;
          known.add(ref.id); return true;
        }).slice(0, 20);
      const contextList = el("section", undefined, "workspace-release-impacts");
      contextList.append(el("h3", copy.text.blockersToRelease));
      node.append(contextList);
      let disposed = false;
      const controller = new AbortController();
      disposers.push(() => { disposed = true; controller.abort(); });
      const headings = workspaceView?.relationships?.nodes || [];
      for (const ref of refs) {
        const entry = { loading: true };
        contextBlockers.push(entry);
        const status = el("p", copy.text.formImpactLoading, "field-help");
        status.setAttribute("role", "status");
        contextList.append(status);
        (async () => {
          try {
            const response = await request(`/namespaces/${scope.namespace_id}/outcomes/${scope.outcome_id}/blockers/${ref.id}`, { signal: controller.signal });
            const blocker = response.value || response;
            if (disposed) return;
            if (blocker.cause_ref?.kind !== "issue" || blocker.cause_ref.id !== entity.id
              || blocker.scope?.namespace_id !== scope.namespace_id || blocker.scope?.outcome_id !== scope.outcome_id)
              throw Error(copy.text.selectOnlyBlockersCausedByThisIssue);
            entry.blocker = blocker;
            entry.loading = false;
            if (blocker.lifecycle !== "active") { status.remove(); return; }
            const target = blocker.blocked_ref;
            let title = headings.find((value) => value.ref?.kind === target.kind && value.ref.id === target.id)?.title;
            if (!title) {
              const query = new URLSearchParams({ kind: target.kind, id: target.id, limit: "1" });
              const matches = await request(`/namespaces/${scope.namespace_id}/outcomes/${scope.outcome_id}/references?${query}`, { signal: controller.signal });
              if (disposed) return;
              title = matches.items?.find((value) => value.ref?.id === target.id)?.title;
            }
            const field = inputField(title || copy.text.blockedItem, { type: "checkbox", value: false,
              help: blocker.description });
            entry.field = field;
            status.replaceWith(field.node);
          } catch (error) {
            if (disposed) return;
            entry.loading = false;
            entry.error = error;
            status.textContent = error.message;
          }
        })();
      }
      if (!refs.length) contextList.append(el("p", copy.text.formNoContextImpacts, "field-help"));
      const partial = !workspaceView || workspaceView.coverage?.relationships?.status !== "complete";
      let picker;
      if (partial) {
        contextList.append(el("p", copy.text.formMoreImpactsMayExist, "field-help"));
        picker = p("_blockers", copy.text.blockersToRelease, ["blocker"], {
          multiple: true, lifecycle: "active",
        });
        advanced.append(picker.node);
      }
      const confirmed = f(
        "_confirmed",
        copy.text.confirmEachSelectedImpactIsReleased,
        {
          type: "checkbox",
        },
      );
      getters.blockers = () => {
        if (!confirmed.get())
          throw Error(copy.text.confirmTheBlockingImpactsAreReleased);
        if (contextBlockers.some((entry) => entry.loading)) throw Error(copy.text.formImpactLoading);
        const selected = new Map(contextBlockers.filter((entry) => entry.field?.get())
          .map((entry) => [entry.blocker.id, { blocker_id: entry.blocker.id,
            expected_version: entry.blocker.version,
            resolution_summary: getters.issue_resolution_summary(), release_confirmed: true }]));
        for (const item of (picker?.get() || []).map((r) => ({
          blocker_id: r.id,
          expected_version: picker
            .getCandidates()
            .find((c) => c.ref.id === r.id)?.version,
          resolution_summary: getters.issue_resolution_summary(),
          release_confirmed: true,
        }))) selected.set(item.blocker_id, item);
        return [...selected.values()];
      };
      help.textContent =
        copy.text.thisExplicitlyResolvesTheIssueAndTheSelectedBlockers;
      break;
    }
    case "create_roadmap":
      title(copy.text.roadmapName);
      const parent = p("_roadmap_scope", copy.text.objectiveScopeOptional, [
        "objective",
      ]);
      getters.plan_scope = () =>
        parent.get()
          ? { kind: "objective", id: parent.get().id }
          : { kind: "outcome", id: scope.outcome_id };
      help.textContent =
        copy.text.aRoadmapReferencesObjectivesAndTasksItNeverOwns;
      break;
    case "open_roadmap_draft":
      id("roadmap");
      f("base_revision_number", copy.text.startFromRevision, {
        options: [
          ["", copy.text.emptyDraft],
          ...(entity.revisions || []).map((r) => [
            String(r.revision_number),
            message(
              "revision",
              r.revision_number,
              r.reason || copy.text.published,
            ),
          ]),
        ],
        value: entity.revisions?.at(-1)?.revision_number || "",
      });
      getters.base_revision_number = () => {
        const v = node.querySelector("select").value;
        return v ? Number(v) : undefined;
      };
      break;
    case "replace_roadmap_draft": {
      id("roadmap");
      fixed.expected_draft_version = entity.draft.draft_version;
      const editor = roadmapEditor({ entity, scope, request });
      node.append(editor.node);
      getters._editor = editor.get;
      disposers.push(editor.dispose);
      help.textContent =
        copy.text.editTheDraftPublicationAndActivationAreSeparateDecisions;
      break;
    }
    case "publish_roadmap_draft":
      id("roadmap");
      fixed.expected_draft_version = entity.draft.draft_version;
      node.append(roadmapDiff(entity));
      reason(copy.text.publicationReason);
      break;
    case "activate_roadmap_revision":
    case "deactivate_roadmap_revision":
      id("roadmap");
      f("revision_number", copy.text.publishedRevision, {
        options: (entity.revisions || []).map((r) => [
          String(r.revision_number),
          message(
            "revision",
            r.revision_number,
            r.reason || copy.text.published,
          ),
        ]),
        value: entity.revisions?.at(-1)?.revision_number,
      });
      getters.revision_number = () =>
        Number(node.querySelector("select").value);
      help.textContent = copy.text.changesTheActivePlanSlotItDoesNotComplete;
      break;
    case "add_dependency":
      fixed.source_ref = ref;
      p("target_ref", copy.text.dependsOn, ["work_item", "objective"], {
        required: true,
        initial: entity._canvasDependencyTarget
          ? [entity._canvasDependencyTarget]
          : [],
      });
      f("strength", "Dependency type", {
        options: [
          ["hard", "Hard prerequisite"],
          ["advisory", "Advisory dependency"],
        ],
        value: "hard",
        help: "Hard prerequisites affect readiness and completion. Advisory dependencies provide context without blocking transitions.",
      });
      reason(copy.text.whyIsThisPrerequisiteRequired);
      help.textContent =
        copy.text.operationalDependenciesAffectReadinessIndependentlyOfVisualRoadmapOrder;
      break;
    case "remove_dependency":
      if (
        kind !== "relation" ||
        entity.relation_type !== "depends_on" ||
        entity.lifecycle !== "active"
      )
        throw Error("Only an active dependency can be removed here.");
      id("relation");
      reason();
      help.textContent =
        "Remove this dependency with an explicit reason. Its source and target remain unchanged. The server checks the relation version and source lifecycle.";
      break;
    case "propose_decision":
    case "propose_decision_in_context":
      if (name === "propose_decision_in_context") {
        fixed.subject_ref = ref;
        node.append(contextualTarget());
        help.textContent = copy.text.formContextRecordHelp;
      }
      title(copy.text.decisionTitle);
      f("proposal", copy.text.proposal, { type: "textarea", required: true });
      f("chosen_alternative", copy.text.chosenAlternative, { required: true });
      f("rationale", copy.text.rationale, { type: "textarea", required: true });
      const alternatives = listField(copy.text.alternatives);
      node.append(alternatives.node);
      getters.alternatives = alternatives.get;
      if (name === "propose_decision_in_context")
        nestFields("decision", ["title", "proposal", "chosen_alternative", "rationale", "alternatives"]);
      break;
    case "revoke_work_contract":
      fixed.contract_id = contractView?.contract.id;
      fixed.expected_contract_version = contractView?.contract.version;
      reason();
      help.textContent =
        copy.text.administrativeRevocationEndsThisContractAuthorityItDoesNot;
      break;
    case "renew_work_item_lease":
      fixed.claim_id = entity.current_lease?.claim_id;
      fixed.fencing_token = entity.current_lease?.fencing_token;
    case "reclaim_work_item":
    case "claim_work_item":
      id("work_item");
      f("_duration", copy.text.reservationDurationMinutes, {
        type: "number",
        value: String(
          (protocol?.lease_policy?.default_ttl_seconds || 900) / 60,
        ),
        required: true,
        help: copy.text.defaultComesFromTheWorkspaceLeasePolicyTheServer,
      });
      getters.ttl_seconds = () => Number(getters._duration()) * 60;
      help.textContent =
        copy.text.legacyReservationTheServerChecksReadinessAndExclusiveAuthority;
      break;
    case "complete_work_item":
    case "release_work_item":
      id("work_item");
      fixed.claim_id = entity.current_lease?.claim_id;
      fixed.fencing_token = entity.current_lease?.fencing_token;
      if (name === "complete_work_item")
        f("result_summary", copy.text.taskResult, {
          type: "textarea",
          required: true,
        });
      reason();
      break;
    default:
      id();
      reason();
      help.textContent =
        copy.text.thisIsAnExplicitLifecycleDecisionTheServerChecks;
  }
  if (advanced.children.length > 1) node.append(advanced);
  return {
    node,
    commandName: () => commandName(),
    updateConclusionImpact(view) { conclusionEditor?.update(view); },
    async get() {
      const result = { ...fixed };
      for (const [key, get] of Object.entries(getters)) {
        if (!key.startsWith("_")) {
          const value = get();
          if (value !== undefined) result[key] = value;
        }
      }
      if (getters._editor) Object.assign(result, await getters._editor());
      if (result.blockers)
        for (const b of result.blockers) {
          const current = await request(
            `/namespaces/${scope.namespace_id}/outcomes/${scope.outcome_id}/blockers/${b.blocker_id}`,
          );
          if (current.value?.cause_ref?.id !== entity.id)
            throw Error(copy.text.selectOnlyBlockersCausedByThisIssue);
          if (!b.expected_version)
            throw Error(copy.text.refreshTheSelectedBlockerBeforeSaving);
        }
      return result;
    },
    dispose() {
      disposers.forEach((f) => f());
    },
  };
}
