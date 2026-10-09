import { test, expect, openOutcome, intent } from "./ux-fixture.mjs";
async function provision(wos, principal, permissions) {
  const snapshot = await wos.api(`/namespaces/${wos.namespace}/administration`);
  const grant = await wos.api("/security/commands", {
    namespace_id: wos.namespace,
    expected_namespace_version: snapshot.namespace_version,
    operation: "set_grant",
    principal_id: principal,
    permissions,
  });
  return wos.api("/security/commands", {
    namespace_id: wos.namespace,
    expected_namespace_version: grant.result.namespace_version,
    operation: "issue_credential",
    principal_id: principal,
    actor_ref: { kind: "human", provider: "ux-test", id: principal },
    expires_at: new Date(Date.now() + 3600000).toISOString(),
  });
}
test("read-only membership hides mutation and settings affordances", async ({
  page,
  wos,
}) => {
  await wos.outcome();
  const issued = await provision(wos, "viewer", ["state:read"]);
  await wos.login(page, issued.token);
  await openOutcome(page);
  await expect(page.locator("#create-outcome")).toBeDisabled();
  await expect(page.locator("#developer-tools")).not.toBeVisible();
  await expect(page.locator("#administration")).not.toBeVisible();
  await page.locator("#actions").click();
  await expect(page.locator("#action-options button")).toHaveCount(0);
});
test("revoked credential cannot retrieve references or dispatch edits", async ({
  page,
  wos,
}) => {
  await wos.outcome();
  const issued = await provision(wos, "planner", [
    "state:read",
    "outcome:write",
    "planning:write",
    "work:write",
  ]);
  await wos.login(page, issued.token);
  await openOutcome(page);
  await intent(page, "Edit outcome");
  await page
    .getByLabel("Outcome name", { exact: true })
    .fill("Forbidden after revocation");
  let dispatched = 0;
  await page.route("**/commands/update_outcome", (route) => {
    dispatched++;
    return route.continue();
  });
  const snapshot = await wos.api(`/namespaces/${wos.namespace}/administration`);
  const credential = snapshot.credentials.find(
    (c) => c.principal_id === "planner",
  );
  await wos.api("/security/commands", {
    namespace_id: wos.namespace,
    expected_namespace_version: snapshot.namespace_version,
    operation: "revoke_credential",
    credential_id: credential.id,
  });
  const denied = await page.evaluate(
    async (path) => {
      const r = await fetch(path);
      return { status: r.status, body: await r.json() };
    },
    `/api/v1/namespaces/${wos.namespace}/outcomes/${(await wos.api(`/namespaces/${wos.namespace}/outcomes`)).items[0].id}/references?kind=work_item`,
  );
  expect([401, 403]).toContain(denied.status);
  expect(denied.body.items).toBeUndefined();
  await page.locator("#submit-command").click();
  await expect(page.locator("#command-error")).not.toBeEmpty();
  expect(dispatched).toBe(0);
  await expect(page.locator("#retry-command")).not.toBeVisible();
});
test("a late failed Outcome read cannot replace the selected Outcome or its notice", async ({
  page,
  wos,
}) => {
  const first = await wos.outcome("Slow Outcome");
  await wos.outcome("Current Outcome");
  await wos.login(page);
  let release;
  const pending = new Promise((done) => (release = done));
  await page.route(
    `**/outcomes/${first.value.id}/continuity?*`,
    async (route) => {
      await pending;
      await route.fulfill({
        status: 503,
        contentType: "application/json",
        body: '{"error":{"message":"obsolete failure"}}',
      });
    },
  );
  await page
    .locator("#outcomes")
    .getByRole("button")
    .filter({ hasText: "Slow Outcome" })
    .click();
  await page
    .locator("#outcomes")
    .getByRole("button")
    .filter({ hasText: "Current Outcome" })
    .click();
  await expect(page.locator("#title")).toHaveText("Current Outcome");
  release();
  await expect(page.locator("#title")).toHaveText("Current Outcome");
  await expect(page.locator("#notice")).not.toContainText("obsolete failure");
});
test("revocation prevents replay dispatch of an already frozen uncertain intent", async ({
  page,
  wos,
}) => {
  await wos.outcome();
  const issued = await provision(wos, "retry-planner", [
    "state:read",
    "outcome:write",
  ]);
  await wos.login(page, issued.token);
  await openOutcome(page);
  await intent(page, "Edit outcome");
  await page
    .getByLabel("Outcome name", { exact: true })
    .fill("One committed update");
  let dispatched = 0;
  await page.route("**/commands/update_outcome", async (route) => {
    dispatched++;
    await route.fetch();
    await route.fulfill({
      status: 503,
      contentType: "application/json",
      body: '{"error":{"message":"Response lost after commit"}}',
    });
  });
  await page.locator("#submit-command").click();
  await expect(page.locator("#retry-command")).toBeVisible();
  const snapshot = await wos.api(`/namespaces/${wos.namespace}/administration`);
  await wos.api("/security/commands", {
    namespace_id: wos.namespace,
    expected_namespace_version: snapshot.namespace_version,
    operation: "revoke_credential",
    credential_id: issued.result.credential.id,
  });
  await page.locator("#retry-command").click();
  await expect(page.locator("#command-error")).not.toBeEmpty();
  expect(dispatched).toBe(1);
});
