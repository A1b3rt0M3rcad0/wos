import { test } from "node:test";
import assert from "node:assert/strict";
import { signedCheckoutHandoff } from "../../../packages/wos-api/web/assets/contracts-view.js";

const id = "019a0180-0000-7000-8000-000000000001";
const scope = { namespace_id: "019a0180-0000-7000-8000-000000000002",
  outcome_id: "019a0180-0000-7000-8000-000000000003" };

test("signed executor and reviewer checkout pin exact contextual scope independently of host defaults", () => {
  for (const kind of ["work", "review"]) {
    const text = signedCheckoutHandoff({ kind, id, scope, version: "9007199254740993" });
    assert.ok(text.includes(`${kind} checkout ${id} --version 9007199254740993`));
    assert.ok(text.includes(`--namespace ${scope.namespace_id} --outcome ${scope.outcome_id}`));
    assert.equal(text.split("\n").length, 3);
    assert.ok(text.includes(`${kind} show ${kind === "work" ? "CONTRACT_UUID" : "REVIEW_CONTRACT_UUID"} --for-agent`));
  }
  assert.ok(signedCheckoutHandoff({ kind: "work", id, scope, version: 7,
    previousReviewCaseId: id }).includes(`--previous-review ${id}`));
});

test("handoffs never interpolate hostile authored values or rounded versions into a shell", () => {
  const base = { kind: "work", id, scope, version: "3" };
  assert.ok(!signedCheckoutHandoff({ ...base, title: "$(touch /tmp/unexpected)",
    description: "`env`" }).includes("touch"));
  for (const value of ["$(touch /tmp/unexpected)", "`id`", "abc; echo secret", "'\nwhoami", "x y"])
    for (const input of [{ ...base, id: value }, { ...base, previousReviewCaseId: value },
      { ...base, scope: { ...scope, namespace_id: value } }, { ...base, scope: { ...scope, outcome_id: value } }])
      assert.throws(() => signedCheckoutHandoff(input));
  for (const version of [9007199254740992, "18446744073709551616", "3; env", "0", "03"])
    assert.throws(() => signedCheckoutHandoff({ ...base, version }));
});
