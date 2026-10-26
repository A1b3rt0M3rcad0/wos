import { test } from "node:test";
import assert from "node:assert/strict";
import { canvasWorkLayer, canvasLinkedRecords, canvasRecordExplorer, canvasCardPresentation, MAX_CONTEXT_RECORDS } from "../../../packages/wos-api/web/assets/workspace-card-summary.js";

const fixture = () => ({
  rootKey: "outcome:one", partial: true,
  nodes: [
    { key: "outcome:one", kind: "outcome" },
    { key: "objective:one", kind: "objective", parentKey: "outcome:one" },
    { key: "work_item:one", kind: "work_item", parentKey: "objective:one" },
    { key: "work_item:two", kind: "work_item", parentKey: "outcome:one" },
    { key: "evidence:shared", kind: "evidence" },
    { key: "issue:one", kind: "issue" },
    { key: "blocker:one", kind: "blocker" },
    { key: "artifact:unrelated", kind: "artifact" },
  ],
  edges: [
    { key: "one", sourceKey: "evidence:shared", targetKey: "work_item:one", kind: "supports" },
    { key: "two", sourceKey: "evidence:shared", targetKey: "work_item:two", kind: "contradicts" },
    { key: "blocks", sourceKey: "blocker:one", targetKey: "work_item:one", kind: "blocks" },
    { key: "cause", sourceKey: "issue:one", targetKey: "blocker:one", kind: "causes" },
  ],
});

test("work layer retains the full loaded model and does not turn documentary relations into ownership", () => {
  const model = fixture(), before = structuredClone(model);
  const work = canvasWorkLayer(model);
  assert.deepEqual(work.model.nodes.map((entry) => entry.key), ["outcome:one", "objective:one", "work_item:one", "work_item:two"]);
  assert.equal(work.hiddenRecordCount, 4);
  assert.equal(work.model.partial, true);
  assert.deepEqual(model, before);
  const expanded = canvasWorkLayer(model, { recordRoots: new Set(["work_item:one"]) });
  assert.deepEqual(canvasLinkedRecords(model, "work_item:one"), new Set(["evidence:shared", "blocker:one", "issue:one"]));
  assert.equal(expanded.model.nodes.filter((entry) => entry.key === "evidence:shared").length, 1);
  assert.equal(expanded.model.nodes.some((entry) => entry.key === "artifact:unrelated"), false);
  assert.equal(expanded.model.nodes.find((entry) => entry.kind === "issue").parentKey, undefined);
  assert.equal(expanded.hiddenRecordCount, 1);
  assert.equal(canvasWorkLayer(model, { layer: "records" }).model, model);
});

test("selection and edge inspection reveal their actual endpoints without enabling every record", () => {
  const model = fixture();
  const selected = canvasWorkLayer(model, { selectedKey: "artifact:unrelated" });
  assert.equal(selected.model.nodes.some((entry) => entry.key === "artifact:unrelated"), true);
  assert.equal(selected.model.nodes.some((entry) => entry.kind === "evidence"), false);
  const edge = canvasWorkLayer(model, { selectedEdgeKey: "one" });
  assert.equal(edge.model.nodes.some((entry) => entry.kind === "evidence"), true);
  assert.equal(edge.model.edges.some((entry) => entry.key === "one"), true);
  assert.equal(edge.model.nodes.some((entry) => entry.kind === "issue"), false);
  const explorer = canvasRecordExplorer(model);
  assert.equal(explorer("work_item:one"), explorer("work_item:one"));
});

test("linked record previews stay bounded while every loaded record remains explicitly accessible", () => {
  const model = fixture();
  for (let index = 0; index < 30; index++) {
    model.nodes.push({ key: `evidence:${index}`, kind: "evidence" });
    model.edges.push({ key: `context:${index}`, sourceKey: `evidence:${index}`, targetKey: "work_item:one", kind: "context" });
  }
  const preview = canvasWorkLayer(model, { recordRoots: new Set(["work_item:one"]) });
  assert.equal(preview.model.nodes.length, 4 + MAX_CONTEXT_RECORDS);
  assert.equal(canvasWorkLayer(model, { layer: "records" }).model.nodes.length, model.nodes.length);
  assert.equal(canvasWorkLayer(model, { selectedKey: "evidence:29" }).model.nodes.some((entry) => entry.key === "evidence:29"), true);
});

test("card facts keep declared responsibility, verification and waivers distinct", () => {
  const facts = canvasCardPresentation({
    kind: "objective", lifecycle: "planned", authorityUnknown: true,
    cardSummary: {
      required_for_outcome: true, authority_state: "unknown",
      participants: [{ role: "declared_owner", actor_ref: { id: "owner" } }, { role: "authorized_executor", actor_ref: { id: "not-authorized" }, authority_state: "active" }],
      criteria: { active: 3, required: 2, met: 1, waived: 1, pending: 1 },
    },
  });
  assert.equal(facts.person, "Owner: owner");
  assert.equal(facts.required, "Required for Outcome");
  assert.equal(facts.summary, "1 of 3 criteria met · 1 waived");
  assert.equal(facts.lifecycle, "planned");
  assert.equal(facts.authorityUnknown, true);
  const unknown = canvasCardPresentation({ kind: "work_item", lifecycle: "todo", cardSummary: { participants: [{ role: "declared_assignee", actor_ref: { id: "worker" } }], authority_state: "unknown" } });
  assert.equal(unknown.person, "Assigned: worker");
  assert.equal(unknown.summary, "Criteria summary not loaded");
  assert.equal(unknown.state, "todo");
});
