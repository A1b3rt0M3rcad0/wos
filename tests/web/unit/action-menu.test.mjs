import { test } from "node:test";
import assert from "node:assert/strict";
import { groupActions } from "../../../packages/wos-api/web/assets/workspace-action-menu.js";
import { humanActions, availableActions } from "../../../packages/wos-api/web/assets/actions.js";
import { copy } from "../../../packages/wos-api/web/assets/en-US.js";

test("action grouping preserves the authorized list exactly and separates advanced operations", () => {
  const actions = Object.entries(humanActions).map(([name, action]) => Object.freeze({ name, ...action }));
  Object.freeze(actions);
  const grouped = groupActions(actions);
  const result = grouped.flatMap((group) => group.actions);
  assert.equal(result.length, actions.length);
  assert.deepEqual(new Set(result), new Set(actions));
  for (const group of grouped) {
    assert.ok(copy.text[`actionGroup${group.category}`]);
    assert.ok(group.actions.every((action) => Boolean(action.advanced) === group.advanced));
  }
  assert.deepEqual(groupActions([]), []);
});

test("presentation cannot restore commands removed by permissions, lifecycle or protocol", () => {
  const registry = Object.fromEntries(Object.keys(humanActions).map((name) => [name, { permission: `test:${name}` }]));
  const permitted = availableActions({
    context: "evidence", entity: { lifecycle: "registered" }, protocol: "signed-v2",
    permissions: ["test:create_evidence_link"], principal: "viewer", inventory: registry,
  });
  assert.deepEqual(groupActions(permitted).flatMap((group) => group.actions.map((action) => action.name)), ["create_evidence_link"]);
  assert.deepEqual(groupActions(availableActions({
    context: "evidence", entity: { lifecycle: "registered" }, protocol: "signed-v2",
    permissions: [], principal: "viewer", inventory: registry,
  })), []);
  const signedWork = availableActions({
    context: "work_item", entity: { lifecycle: "todo", _operational_state: { display_state: "ready" } },
    protocol: { phase: "signed_contracts_v2" }, permissions: Object.values(registry).map((entry) => entry.permission),
    principal: "executor", inventory: registry,
  });
  assert.ok(signedWork.length > 0);
  assert.ok(!groupActions(signedWork).flatMap((group) => group.actions)
    .some((action) => ["claim_work_item", "complete_work_item", "release_work_item"].includes(action.name)));
});
