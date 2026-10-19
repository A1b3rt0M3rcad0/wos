import { test } from "node:test";
import assert from "node:assert/strict";
import { validatePlanItems, planSummary } from "../../../packages/wos-api/web/assets/workspace-plan-editor.js";

const objective = (local_key, fields = {}) => ({
  local_key, kind: "objective", title: `Condition ${local_key}`, required_for_outcome: true, ...fields,
});
const task = (local_key, fields = {}) => ({ local_key, kind: "work_item", title: `Work ${local_key}`, ...fields });

test("a mixed new plan keeps explicit Outcome obligations and parents without mutating its draft", () => {
  const draft = [
    task("endpoint", { parent_local_key: "refunds" }),
    objective("refunds"),
    objective("docs", { parent_local_key: "refunds", required_for_outcome: false }),
    task("release"),
  ];
  for (const item of draft) Object.freeze(item);
  Object.freeze(draft);
  assert.deepEqual(validatePlanItems(draft), draft);
  assert.deepEqual(planSummary(draft), { required: 1, optional: 1, tasks: 2 });
  const copy = validatePlanItems(draft);
  copy[0].title = "An independent local edit";
  assert.equal(draft[0].title, "Work endpoint");
  assert.equal(Object.hasOwn(copy[0], "lifecycle"), false,
    "the plan endpoint, rather than the draft, fixes Task lifecycle to backlog");
});

test("ambiguous, absent and incompatible local parents cannot enter the preview", () => {
  assert.throws(() => validatePlanItems([task("same"), objective("same")]));
  assert.throws(() => validatePlanItems([task("missing", { parent_local_key: "unknown" })]));
  assert.throws(() => validatePlanItems([task("parent"), task("child", { parent_local_key: "parent" })]));
  assert.throws(() => validatePlanItems([objective("parent"), task("child", {
    parent_local_key: "parent", parent_objective_id: "persisted-parent",
  })]));
  assert.throws(() => validatePlanItems([{ local_key: "evidence", kind: "evidence", title: "Not plan work" }]));
  assert.throws(() => validatePlanItems([task("blank", { title: "  " })]));
});

test("self cycles and long cycles are rejected while forward parent references remain valid", () => {
  assert.throws(() => validatePlanItems([objective("a", { parent_local_key: "a" })]));
  assert.throws(() => validatePlanItems([
    objective("a", { parent_local_key: "b" }),
    objective("b", { parent_local_key: "c" }),
    objective("c", { parent_local_key: "a" }),
  ]));
  assert.doesNotThrow(() => validatePlanItems([
    objective("child", { parent_local_key: "parent" }), objective("parent"),
  ]));
});

test("the draft honors the specific endpoint's item and criterion limits", () => {
  assert.throws(() => validatePlanItems([]));
  assert.doesNotThrow(() => validatePlanItems(Array.from({ length: 50 }, (_, i) => task(`task-${i}`))));
  assert.throws(() => validatePlanItems(Array.from({ length: 51 }, (_, i) => task(`task-${i}`))));
  const condition = { title: "Expected response", required: true, verification_mode: "evidence_review" };
  assert.doesNotThrow(() => validatePlanItems([objective("a", { initial_criteria: Array(100).fill(condition) })]));
  assert.throws(() => validatePlanItems([
    objective("a", { initial_criteria: Array(100).fill(condition) }),
    task("b", { initial_criteria: [condition] }),
  ]));
});
