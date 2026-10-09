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
  },
) {
  const node = el("div", undefined, "human-form"),
    advanced = details(copy.text.advanced),
    getters = {},
    fixed = { scope: { ...scope } },
    disposers = [];
  const kind = entity._kind || "outcome";
  const ref = { ...scope, kind, id: entity.id || scope.outcome_id };
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
  const help = el("p", undefined, "flow-help");
  node.append(help);
  switch (name) {
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

    case "create_outcome":
      delete fixed.scope;
      fixed.namespace_id = scope.namespace_id;
      title(copy.text.outcomeName);
      f("desired_state", copy.text.whatDoesSuccessLookLike, {
        type: "textarea",
        required: true,
      });
      description();
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
    case "create_objective":
      title(copy.text.objectiveTitle);
      description();
      p("parent_objective_id", copy.text.parentObjective, ["objective"], {
        ids: true,
        initial: kind === "objective" ? [ref] : [],
      });
      f("required_for_outcome", copy.text.requiredForOutcome, {
        type: "checkbox",
        value: false,
        help: copy.text.thisObjectiveMustBeAchievedBeforeTheOutcomeCan,
      });
      a("priority", copy.text.priority, { options: priority, value: "normal" });
      help.textContent =
        copy.text.anObjectiveDescribesAnIntermediateConditionNotAnExecution;
      break;
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
      description();
      p("objective_id", copy.text.objective, ["objective"], {
        ids: true,
        initial: kind === "objective" ? [ref] : [],
      });
      f("priority", copy.text.priority, { options: priority, value: "normal" });
      f("lifecycle", copy.text.initialState, {
        options: [
          ["todo", copy.text.toDo],
          ["backlog", copy.text.backlog],
        ],
        value: "todo",
        help: copy.text.toDoCanBecomeReadyWhenTheOutcomeDependencies,
      });
      const spec = details(copy.text.executionInstructions);
      const lists = {};
      for (const [key, label] of Object.entries({
        instructions: copy.text.instructions,
        constraints: copy.text.constraints,
        deliverables: copy.text.deliverables,
        scope_hints: copy.text.scopeHints,
      })) {
        const list = listField(label);
        spec.append(list.node);
        lists[key] = list.get;
      }
      const context = referencePicker({
        label: copy.text.contextReferences,
        kinds: referenceKinds,
        scope,
        request,
        multiple: true,
      });
      spec.append(context.node);
      lists.context_refs = context.get;
      disposers.push(context.dispose);
      node.append(spec);
      getters.execution_spec = () => {
        const value = Object.fromEntries(
          Object.entries(lists).map(([k, get]) => [k, get()]),
        );
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
      title(copy.text.criterionTitle);
      description();
      f("required", copy.text.required, { type: "checkbox", value: true });
      f("verification_mode", copy.text.howWillItBeVerified, {
        options: [
          ["attestation", copy.text.attestationAuthorizedDeclaration],
          ["evidence_review", copy.text.evidenceReviewAssessRegisteredProof],
          [
            "external_evaluation",
            copy.text.externalEvaluationIndependentEvaluator,
          ],
        ],
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
          options: [
            ["attestation", copy.text.attestation],
            ["evidence_review", copy.text.evidenceReview],
            ["external_evaluation", copy.text.externalEvaluation],
          ],
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
      if (permissions.includes("assessment:waive"))
        results.push(["waived", copy.text.waivedNotVerifiedecbb]);
      f("result", copy.text.assessmentResult, { options: results });
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
    case "register_artifact":
      f("name", copy.text.artifactName, { required: true });
      f("artifact_type", copy.text.artifactType, { required: true });
      f("uri", "URL", { type: "url", required: true });
      description();
      fixed.producer_ref = {
        kind: "human",
        provider: "wos-web",
        id: "browser",
      };
      break;
    case "create_evidence_link":
      fixed.evidence_id = kind === "evidence" ? entity.id : undefined;
      if (!fixed.evidence_id)
        p("evidence_id", copy.text.evidence, ["evidence"], {
          ids: true,
          required: true,
        });
      p(
        "target_ref",
        copy.text.relatedItem,
        ["outcome", "objective", "work_item", "issue", "blocker"],
        { required: true },
      );
      f("stance", copy.text.evidenceStance, {
        options: ["supports", "contradicts", "neutral"].map((x) => [
          x,
          display(x),
        ]),
      });
      f("rationale", copy.text.rationale, { required: true, type: "textarea" });
      help.textContent =
        copy.text.linkingEvidenceRecordsARelationshipItDoesNotAssess;
      break;
    case "create_issue":
    case "report_issue_with_blocker":
      title(copy.text.issueTitle);
      description();
      f("severity", copy.text.severity, {
        options: ["major", "critical", "minor", "informational"].map((x) => [
          x,
          display(x),
        ]),
      });
      p(
        "affected_refs",
        copy.text.affectedItems,
        ["outcome", "objective", "work_item"],
        { multiple: true },
      );
      if (name === "report_issue_with_blocker") {
        p(
          "blocked_ref",
          copy.text.blockedItem,
          ["outcome", "objective", "work_item"],
          { required: true },
        );
        f("blocker_description", copy.text.blockingImpact, {
          required: true,
          type: "textarea",
        });
        f("propagation", copy.text.propagation, {
          options: [
            ["direct", copy.text.onlyThisItem],
            ["subtree", copy.text.thisItemAndItsDescendants],
          ],
        });
      }
      help.textContent =
        name === "create_issue"
          ? copy.text.anIssueRecordsAProblemItDoesNotBlock
          : copy.text.thisExplicitCompoundActionReportsAProblemAndA;
      break;
    case "create_blocker":
      p(
        "blocked_ref",
        copy.text.blockedItem,
        ["outcome", "objective", "work_item"],
        { required: true, initial: kind === "work_item" ? [ref] : [] },
      );
      description();
      p("cause_ref", copy.text.blockingCause, [
        "issue",
        "work_item",
        "objective",
        "decision",
      ]);
      f("propagation", copy.text.propagation, {
        options: [
          ["direct", copy.text.onlyThisItem],
          ["subtree", copy.text.thisItemAndItsDescendants],
        ],
      });
      help.textContent = copy.text.aBlockerRecordsImpactOnWorkResolvingItsIssue;
      break;
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
      const picker = p("_blockers", copy.text.blockersToRelease, ["blocker"], {
        multiple: true,
        lifecycle: "active",
      });
      const confirmed = f(
        "_confirmed",
        copy.text.confirmEachSelectedImpactIsReleased,
        {
          type: "checkbox",
        },
      );
      getters.blockers = async () => []; // get below resolves versions on an authorized read.
      getters.blockers = () => {
        if (!confirmed.get())
          throw Error(copy.text.confirmTheBlockingImpactsAreReleased);
        return picker.get().map((r) => ({
          blocker_id: r.id,
          expected_version: picker
            .getCandidates()
            .find((c) => c.ref.id === r.id)?.version,
          resolution_summary: getters.issue_resolution_summary(),
          release_confirmed: true,
        }));
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
      });
      fixed.strength = "hard";
      reason(copy.text.whyIsThisPrerequisiteRequired);
      help.textContent =
        copy.text.operationalDependenciesAffectReadinessIndependentlyOfVisualRoadmapOrder;
      break;
    case "propose_decision":
      title(copy.text.decisionTitle);
      f("proposal", copy.text.proposal, { type: "textarea", required: true });
      f("chosen_alternative", copy.text.chosenAlternative, { required: true });
      f("rationale", copy.text.rationale, { type: "textarea", required: true });
      const alternatives = listField(copy.text.alternatives);
      node.append(alternatives.node);
      getters.alternatives = alternatives.get;
      break;
    case "revoke_work_contract":
      fixed.contract_id = contractView?.contract.id;
      fixed.expected_contract_version = contractView?.contract.version;
      reason();
      help.textContent =
        copy.text.administrativeRevocationEndsThisContractAuthorityItDoesNot;
      break;
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
    async get() {
      const result = { ...fixed };
      for (const [key, get] of Object.entries(getters)) {
        if (!key.startsWith("_")) {
          const value = get();
          if (value !== undefined) result[key] = value;
        }
      }
      if (getters._editor) Object.assign(result, getters._editor());
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
