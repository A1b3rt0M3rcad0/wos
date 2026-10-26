import { test } from "node:test";
import assert from "node:assert/strict";
import { workspaceActions, workspaceCommandName, workspaceHeaderTitle } from "../../../packages/wos-api/web/assets/workspace-view.js";

const ref = { namespace_id: "workspace", outcome_id: "outcome", kind: "work_item", id: "task" };
const subject = { kind: "entity", ref };
const cap = (action_id, presentation_priority, extra = {}) => ({
  action_id, subject, presentation_priority, availability: "available", transport: "browser_command", ...extra,
});

test("action presentation retains observed blocking and authority channels without inventing availability", () => {
  const view = Object.freeze({ subject, capabilities: Object.freeze([
    cap("UpdateWorkItem", 25),
    cap("ActivateWorkItem", 10, { availability: "blocked", reason_codes: ["blocked"] }),
    cap("AcquireSignedWorkContract", 15, { transport: "signed_host" }),
  ]) });
  const actions = workspaceActions(view);
  assert.deepEqual(actions.map((entry) => entry.action_id), ["ActivateWorkItem", "AcquireSignedWorkContract", "UpdateWorkItem"]);
  assert.equal(actions[0].availability, "blocked");
  assert.equal(actions[1].transport, "signed_host");
  assert.equal(workspaceCommandName("SetWorkItemExecutionSpec"), "set_work_item_execution_spec");
});

test("unknown commands, foreign scopes and different nested selectors do not become runnable UI actions", () => {
  const view = { subject, capabilities: [
    cap("RunArbitraryRemoteCommand", 1),
    cap("UpdateWorkItem", 2, { subject: { ...subject, ref: { ...ref, namespace_id: "foreign" } } }),
    cap("RecordCriterionAssessment", 3, { subject: { kind: "criterion", ref, criterion_id: "condition" } }),
    cap("UpdateWorkItem", 25),
  ] };
  assert.deepEqual(workspaceActions(view).map((entry) => entry.action_id), ["UpdateWorkItem"]);
  const nested = { kind: "criterion", ref, criterion_id: "condition" };
  assert.deepEqual(workspaceActions({ subject: nested, capabilities: [
    cap("RecordCriterionAssessment", 1, { subject: { ...nested, criterion_id: "other-condition" } }),
    cap("RecordCriterionAssessment", 2, { subject: nested }),
  ] }).map((entry) => entry.presentation_priority), [2]);
});

test("repeated capability hints do not duplicate the same action in a compact footer", () => {
  const actions = workspaceActions({ subject, capabilities: [cap("UpdateWorkItem", 40), cap("UpdateWorkItem", 25)] });
  assert.equal(actions.length, 1);
  assert.equal(actions[0].presentation_priority, 25);
});

test("Task creation capabilities retain separate Backlog and To do targets", () => {
  const objective = { kind: "entity", ref: { ...ref, kind: "objective", id: "objective" } };
  const actions = workspaceActions({ subject: objective, capabilities: [
    cap("CreateWorkItem", 10, { subject: objective, target_lifecycle: "backlog" }),
    cap("CreateWorkItem", 11, { subject: objective, target_lifecycle: "todo", availability: "unknown", reason_codes: ["target_definition_required"] }),
    cap("CreateWorkItem", 12, { subject: objective, target_lifecycle: "todo", availability: "blocked", reason_codes: ["target_definition_required"] }),
  ] });
  assert.deepEqual(actions.map((entry) => [entry.target_lifecycle, entry.availability]), [
    ["backlog", "available"], ["todo", "unknown"],
  ]);
});

test("Blocker workspace headings use the entity name while preserving the recorded description", () => {
  const view = {
    subject: { kind: "entity", ref: { ...ref, kind: "blocker", id: "blocker" } },
    header: { kind: "blocker", title: "Wait for verified environment access", description: "Wait for verified environment access" },
  };
  assert.equal(workspaceHeaderTitle(view), "Blocker");
  assert.equal(view.header.description, "Wait for verified environment access");
  assert.equal(workspaceHeaderTitle({
    subject: { kind: "entity", ref }, header: { kind: "work_item", title: "Install the package" },
  }), "Install the package");
});
