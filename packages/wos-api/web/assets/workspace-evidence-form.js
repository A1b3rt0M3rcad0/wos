import { copy, message } from "./en-US.js";
import { el, inputField, details } from "./ui.js";
import { referencePicker } from "./reference-picker.js";
import { localDateTime } from "./workspace-form-fields.js";
import { display } from "./presentation.js";

function exactVersion(value) {
  if (typeof value === "string" && /^[1-9][0-9]*$/.test(value)) return value;
  if (Number.isSafeInteger(value) && value > 0) return String(value);
  throw Error(copy.text.formReviewTargets);
}

// A single typed command saves the observation and its explicit relationship.
// No write is performed by this editor and no evidence implies acceptance.
export function contextualEvidenceForm({ scope, entity, criterion, request, contextTitle, eligibility }) {
  const node = el("section", undefined, "workspace-evidence-form");
  const target = { ...scope, kind: entity._kind || "outcome", id: entity.id || scope.outcome_id };
  let boundEntity = entity, boundCriterion = criterion, boundContextTitle = contextTitle;
  const context = el("p", undefined, "field-context");
  function renderContext() {
    context.replaceChildren(el("small", copy.text.formEvidenceTarget),
      el("strong", boundContextTitle || boundEntity.title || boundEntity.name || copy.text.outcome));
    if (boundCriterion) context.append(el("small", copy.text.formEvidenceReviewDefinition), el("span", boundCriterion.title),
      el("small", message("workspaceDefinitionRevision", boundCriterion.criterion_revision), "field-help"));
  }
  renderContext();
  node.append(context);
  const origin = inputField(copy.text.formEvidenceOrigin, { options: [
    ["external", copy.text.formEvidenceExternal], ["artifact", copy.text.formEvidenceArtifact],
    ["observation", copy.text.formEvidenceObservation], ["existing", copy.text.formEvidenceReuse],
  ], value: "external" });
  const description = inputField(copy.text.description, { type: "textarea", required: true });
  const source = el("section", undefined, "nested-editor workspace-evidence-source");
  const provider = inputField(copy.text.sourceProvider, { value: "external" });
  const url = inputField(copy.text.sourceURL, { type: "url" });
  const externalId = inputField(copy.text.sourceIdentifier);
  source.append(provider.node, url.node, externalId.node);
  const artifact = referencePicker({
    label: copy.text.supportingArtifact, kinds: ["artifact"], scope, request,
  });
  const existing = referencePicker({
    label: copy.text.evidence, kinds: ["evidence"], scope, request, lifecycle: "registered",
  });
  const observationHelp = el("p", copy.text.formEvidenceObservationHelp, "field-help");
  const captured = inputField(copy.text.observedAtLocalTime, {
    type: "datetime-local", value: localDateTime(new Date()),
  });
  const type = inputField(copy.text.evidenceType, { options: [
    "source", "inspection", "test_result", "attestation", "external_evaluation", "measurement",
  ].map((value) => [value, display(value)]) });
  const measure = details(copy.text.measurementDetails);
  const measurements = {};
  for (const [key, label] of Object.entries({ value: copy.text.measuredValue,
    unit: copy.text.unit, method: copy.text.method, conditions: copy.text.conditions })) {
    const field = inputField(label);
    measurements[key] = field;
    measure.append(field.node);
  }
  const stance = inputField(copy.text.evidenceStance, { options: [
    ["supports", copy.text.supports], ["contradicts", copy.text.contradicts], ["context", copy.text.formEvidenceContext],
  ], value: criterion ? "supports" : "context" });
  const rationale = inputField(copy.text.formEvidenceRelevance, { type: "textarea", required: true });
  node.append(origin.node, description.node, source, artifact.node, existing.node,
    observationHelp, captured.node, type.node, measure, stance.node, rationale.node,
    el("p", copy.text.formEvidenceNoAcceptance, "field-help"));
  function sync() {
    const selected = origin.get();
    for (const [element, visible] of [
      [source, selected === "external"], [artifact.node, selected === "artifact"],
      [existing.node, selected === "existing"], [observationHelp, selected === "observation"],
      [description.node, selected !== "existing"], [captured.node, selected !== "existing"],
      [type.node, !["observation", "existing"].includes(selected)],
      [measure, !["observation", "existing"].includes(selected) && type.get() === "measurement"],
    ]) {
      element.hidden = !visible;
      element.inert = !visible;
    }
    description.input.required = selected !== "existing";
    provider.input.required = selected === "external";
    captured.input.required = selected !== "existing";
  }
  origin.input.addEventListener("change", sync);
  type.input.addEventListener("change", sync);
  sync();
  return {
    node,
    updateBinding({ entity: latestEntity, criterion: latestCriterion, contextTitle: latestContextTitle }) {
      if (!latestEntity || latestEntity.id !== target.id || (latestEntity._kind && latestEntity._kind !== target.kind))
        throw Error(copy.text.formEvidenceOwnerChanged);
      if (!latestCriterion || latestCriterion.id !== boundCriterion?.id || latestCriterion.status === "retired")
        throw Error(copy.text.formEvidenceCriterionUnavailable);
      boundEntity = { ...latestEntity, _kind: target.kind };
      boundCriterion = latestCriterion;
      if (latestContextTitle) boundContextTitle = latestContextTitle;
      renderContext();
    },
    async get() {
      if (!eligibility?.evidence_link_target_kinds?.includes(target.kind))
        throw Error(copy.text.formEvidenceTargetsUnavailable);
      const link = { target_ref: target, stance: stance.get(), rationale: rationale.get() };
      if (boundCriterion) {
        link.criterion_id = boundCriterion.id;
        link.criterion_revision = exactVersion(boundCriterion.criterion_revision);
        link.target_expected_version = exactVersion(boundEntity.version);
      }
      const result = { scope: { ...scope }, links: [link] };
      const selected = origin.get();
      if (selected === "existing") {
        const evidence = existing.get();
        if (!evidence) throw Error(copy.text.formEvidenceOriginMissing);
        result.existing_evidence_id = evidence.id;
        return result;
      }
      const capturedAt = captured.get() ? new Date(captured.get()).toISOString() : undefined;
      result.evidence = {
        evidence_type: selected === "observation" ? "attestation" : type.get(),
        description: description.get(), captured_at: capturedAt,
      };
      if (selected === "external") {
        if (!url.get() && !externalId.get())
          throw Error(copy.text.provideASourceURLOrExternalSourceIdentifier);
        result.evidence.source_ref = {
          provider: provider.get(), ...(url.get() ? { uri: url.get() } : {}),
          ...(externalId.get() ? { id: externalId.get() } : {}),
        };
      } else if (selected === "artifact") {
        const selectedArtifact = artifact.get();
        if (!selectedArtifact) throw Error(copy.text.formEvidenceOriginMissing);
        result.evidence.artifact_id = selectedArtifact.id;
        result.evidence.source_ref = { provider: "artifact", id: selectedArtifact.id };
      } else if (selected === "observation") {
        result.observation = {
          subject_ref: target, body: description.get(), occurred_at: capturedAt,
          ...(boundCriterion ? { criterion_id: boundCriterion.id } : {}),
        };
      } else throw Error(copy.text.formEvidenceOriginMissing);
      if (selected !== "observation" && type.get() === "measurement")
        result.evidence.measurement = Object.fromEntries(
          Object.entries(measurements).map(([key, field]) => [key, field.get() || ""]));
      return result;
    },
    dispose() { artifact.dispose(); existing.dispose(); },
  };
}
