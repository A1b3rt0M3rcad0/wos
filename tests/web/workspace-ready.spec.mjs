import { expandWorkspaceDetails } from './workspace-action-fixture.mjs';
import { test, expect, openOutcome, chooseArea } from "./ux-fixture.mjs";
import { seedCanvas } from "./canvas-fixture.mjs";

test("workspace mutations remain unavailable until the real authenticated identity finishes loading", async ({ page, wos }) => {
  let releaseIdentity, identityArrived;
  const held = new Promise((resolve) => { releaseIdentity = resolve; });
  const arrived = new Promise((resolve) => { identityArrived = resolve; });
  const mutations = [];
  page.on("request", (request) => {
    if (request.method() === "POST" && request.url().includes("/commands/")) mutations.push(request.url());
  });
  await page.route("**/api/v1/identity", async (route) => {
    // Delay transport of a genuine authorized response, never its authority.
    const response = await route.fetch();
    expect(response.ok()).toBe(true);
    identityArrived();
    await held;
    await route.fulfill({ response });
  });
  try {
    await page.goto(`${wos.url}/app/`);
    await page.getByLabel("Access credential", { exact: true }).fill(wos.token);
    await page.getByRole("button", { name: "Sign in", exact: true }).click();
    await arrived;
    await expect(page.locator("#workspace")).toBeVisible();
    await expect(page.locator("#workspace")).toHaveAttribute("aria-busy", "true");
    expect(await page.locator("#workspace").evaluate((element) => element.inert)).toBe(true);
    await expect(page.locator("#workspace-loading")).toBeVisible();
    await expect(page.locator("#empty-create")).toBeDisabled();
    await expect(page.locator("#create-outcome")).toBeDisabled();
    await page.locator("#logout").focus();
    await page.keyboard.press("Tab");
    expect(await page.locator("#workspace").evaluate((element) => element.contains(document.activeElement))).toBe(false);
    expect(mutations).toEqual([]);

    releaseIdentity();
    await expect(page.locator("#workspace-loading")).not.toBeVisible();
    await expect(page.locator("#workspace")).toHaveAttribute("aria-busy", "false");
    expect(await page.locator("#workspace").evaluate((element) => element.inert)).toBe(false);
    await expect(page.locator("#empty-create")).toBeEnabled();
    await page.locator("#empty-create").click();
    await expect(page.locator("#command-form")).toBeVisible();
    await page.getByLabel("Outcome name", { exact: true }).fill("An Outcome created after authenticated readiness");
    await page.getByLabel("What does success look like?", { exact: true }).fill("A real participant can plan once identity and permissions are loaded.");
    const creation = page.waitForResponse((response) => response.request().method() === "POST"
      && response.url().endsWith("/commands/create_outcome"));
    await page.locator("#submit-command").click();
    const response = await creation;
    const result = await response.json();
    expect(response.ok(), JSON.stringify(result)).toBe(true);
    expect(result.value.lifecycle).toBe("draft");
    await expect(page.locator("#title")).toHaveText(result.value.title);
    expect(mutations).toHaveLength(1);
    expect(mutations[0]).toMatch(/\/commands\/create_outcome$/);
  } finally {
    releaseIdentity();
    await page.unroute("**/api/v1/identity");
  }
});

test("native Escape invalidates a genuine pending criterion read instead of reopening its closed dialog", async ({ page, wos }) => {
  const data = await seedCanvas(wos);
  const criterion = await wos.command("add_criterion", {
    owner: data.ref("work_item", data.tasks.ready), expected_version: data.tasks.ready.version,
    title: "The recorded delivery satisfies the agreed condition", required: true, verification_mode: "evidence_review",
  });
  await wos.login(page);
  await openOutcome(page, data.outcome.title);
  await chooseArea(page, "work");
  await page.locator("#content").getByRole("button").filter({ hasText: data.tasks.ready.title }).first().click();
  const dialog = page.locator("#detail-dialog");
  await expect(dialog).toBeVisible();
  await expect(dialog.locator("#detail-title")).toHaveText(data.tasks.ready.title);
  await expandWorkspaceDetails(dialog, {history:false});
  let release, arrived, delivered;
  const gate = new Promise((resolve) => { release = resolve; });
  const reached = new Promise((resolve) => { arrived = resolve; });
  const finished = new Promise((resolve) => { delivered = resolve; });
  await page.route("**/workspace-view", async (route) => {
    const subject = route.request().postDataJSON()?.subject;
    if (subject?.kind !== "criterion" || subject.criterion_id !== criterion.id) return route.continue();
    try {
      const response = await route.fetch();
      expect(response.ok()).toBe(true);
      arrived();
      await gate;
      await route.fulfill({ response });
    } finally { delivered(); }
  });
  try {
    await dialog.locator(`[data-criterion-id="${criterion.id}"] [data-action-id="open_criterion"]`).click();
    await reached;
    await page.keyboard.press("Escape");
    await expect(dialog).not.toBeVisible();
    release();
    await finished;
    await page.evaluate(async () => {
      if (!(await fetch("/livez")).ok) throw Error("The real response fence failed");
    });
    await expect(dialog).not.toBeVisible();
    await expect(page.locator("#canvas-details")).not.toBeVisible();
    await expect(page).not.toHaveURL(/item=/);
    await expect(page.locator("#notice")).not.toHaveClass(/error/);
  } finally {
    release();
    await page.unroute("**/workspace-view");
  }
});
