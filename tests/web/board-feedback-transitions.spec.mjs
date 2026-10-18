import { test, expect, openOutcome, chooseArea } from "./ux-fixture.mjs";

const notice = (page) => page.locator("#notice");

async function rememberNotice(page) {
  await page.evaluate(() => { window.originalFeedbackNode = document.querySelector("#notice"); });
  await expect(page.locator("#canvas-feedback > #notice")).toHaveCount(1);
}

async function visibleTransitionError(page, message) {
  await expect(notice(page)).toHaveCount(1);
  await expect(notice(page)).toHaveAttribute("role", "status");
  await expect(notice(page)).toHaveAttribute("aria-live", "polite");
  await expect(notice(page)).toHaveClass(/error/);
  await expect(notice(page)).toHaveText(message);
  await expect(notice(page)).toBeVisible();
  await expect(page.locator(".workspace-main > #notice")).toHaveCount(1);
  await expect(page.locator("#canvas-feedback > #notice")).toHaveCount(0);
  expect(await page.evaluate(() => document.querySelector("#notice") === window.originalFeedbackNode),
    "Transitions preserve the original single live region").toBe(true);
  await expect(page.locator("#outcome")).toBeHidden();
  await expect(page.locator("#empty")).toBeVisible();
}

// Preserve the real server read and its scope; only delivery of that completed
// read fails. No fixture response fabricates a different Workspace or Outcome.
async function failRead(page, pattern, message, check) {
  const completed = [];
  await page.route(pattern, async (route) => {
    const response = await route.fetch();
    expect(response.ok(), "The underlying authenticated server read succeeds").toBe(true);
    const value = await response.json();
    check(value);
    completed.push(route.request().url());
    await route.fulfill({
      status: 503,
      contentType: "application/json",
      body: JSON.stringify({ error: { code: "temporarily_unavailable", message } }),
    });
  });
  return completed;
}

test("a failed Outcome transition from Board keeps its error in the original visible live region", async ({ page, wos }) => {
  const previous = await wos.outcome("A previous context to resume");
  const current = await wos.outcome("A Board that was already open");
  await wos.login(page);
  await openOutcome(page, previous.value.title, { area: null });
  await chooseArea(page, "overview");
  await openOutcome(page, current.value.title, { area: null });
  await expect(page.locator("#canvas-view")).toBeVisible();
  await rememberNotice(page);

  const message = "This Outcome could not be loaded. Try opening it again.";
  const reads = await failRead(page, `**/outcomes/${previous.value.id}/continuity?**`, message,
    (snapshot) => expect(snapshot.outcome.ref.id).toBe(previous.value.id));
  // Use actual same-document browser history. The previous Outcome's route is
  // restored while the current Board still owns the feedback node.
  await page.goBack();
  await visibleTransitionError(page, message);
  expect(reads).toHaveLength(1);
});

test("Workspace selection reset after Board exposes read errors beside the empty-workspace state", async ({ page, wos }) => {
  const current = await wos.outcome("A Board before Workspace selection resets");
  await wos.login(page);
  await openOutcome(page, current.value.title, { area: null });
  await expect(page.locator("#canvas-view")).toBeVisible();
  await rememberNotice(page);

  const message = "The Workspace list could not be loaded. Try searching again.";
  const reads = await failRead(page, `**/namespaces/${wos.namespace}/outcomes?**`, message,
    (result) => expect(result.items.some((item) => item.id === current.value.id)).toBe(true));
  // This fixture's credential authorizes one Workspace. Exercise the existing
  // select change handler with that real authorized selection rather than
  // fabricate another Namespace, permissions or an empty API response.
  await page.locator("#namespace").evaluate((element) => {
    element.dispatchEvent(new Event("change", { bubbles: true }));
  });
  await visibleTransitionError(page, message);
  expect(reads).toHaveLength(1);
  await expect(page.locator("#namespace")).toHaveValue(wos.namespace);
  await expect(page.locator("#outcome-search-status")).toHaveText(message);
});
