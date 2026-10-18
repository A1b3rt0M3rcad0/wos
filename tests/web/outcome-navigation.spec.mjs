import { test, expect, openOutcome, chooseArea, openViews } from "./ux-fixture.mjs";

const rows = (page) => page.locator("#outcomes button[data-outcome-id]");
const listRead = (request, namespace) => request.method() === "GET" &&
  new URL(request.url()).pathname === `/api/v1/namespaces/${namespace}/outcomes`;
const queryIs = (request, namespace, text, cursor = false) =>
  listRead(request, namespace) && new URL(request.url()).searchParams.get("text") === text &&
  new URL(request.url()).searchParams.has("cursor") === cursor;
const frame = (page) => page.evaluate(() => new Promise((resolve) =>
  requestAnimationFrame(() => requestAnimationFrame(resolve))));

async function sidebar(page, wos, title) {
  await wos.login(page);
  await openOutcome(page, title);
  await expect(page.locator("#outcomes")).toHaveAttribute("aria-busy", "false");
  await expect(page.locator(".sidebar")).toBeVisible();
}
async function search(page, wos, value) {
  const response = page.waitForResponse((response) => queryIs(response.request(), wos.namespace, value));
  await page.getByRole("searchbox", { name: "Search outcomes", exact: true }).fill(value);
  await response;
  await expect(page.locator("#outcomes")).toHaveAttribute("aria-busy", "false");
}
function writes(page) {
  const captured = [];
  page.on("request", (request) => {
    if (request.method() === "POST" && request.url().includes("/commands/")) captured.push(request.url());
  });
  return captured;
}
// Hold delivery of a real completed read. A superseding query may abort the
// browser request; its already-completed server result must remain harmless.
async function holdRead(page, predicate) {
  let release, arrived, finished;
  const gate = new Promise((resolve) => { release = resolve; });
  const seen = new Promise((resolve) => { arrived = resolve; });
  const done = new Promise((resolve) => { finished = resolve; });
  const handler = async (route) => {
    if (!predicate(route.request())) return route.continue();
    const response = await route.fetch();
    arrived({ value: await response.json(), url: route.request().url() });
    await gate;
    try { await route.fulfill({ response }); }
    finally { finished(); }
  };
  await page.route("**/api/v1/**", handler);
  return { seen, async deliver() { release(); await done; await frame(page); await page.unroute("**/api/v1/**", handler); } };
}

test("sidebar search debounces typing, submits Enter immediately and never mutates Outcomes", async ({ page, wos }) => {
  const first = await wos.outcome("Alpha delivery");
  await wos.outcome("Beta delivery");
  await sidebar(page, wos, first.value.title);
  const before = (await wos.api(`/namespaces/${wos.namespace}/outcomes/${first.value.id}`)).value;
  const mutations = writes(page), reads = [];
  page.on("request", (request) => { if (listRead(request, wos.namespace)) reads.push(new URL(request.url())); });
  const input = page.getByRole("searchbox", { name: "Search outcomes", exact: true });
  await expect(input).toHaveCount(1);
  await expect(page.locator("#search button[type=submit], #search .search-submit")).toHaveCount(0);
  await expect(page.locator("#search").getByRole("button", { name: "Search", exact: true })).toHaveCount(0);
  await page.clock.install({ time: new Date("2026-10-18T12:00:00Z") });
  await page.clock.pauseAt(new Date("2026-10-18T12:00:01Z"));
  await input.fill("Al");
  await page.clock.runFor(200);
  await input.fill("Alpha");
  await page.clock.runFor(249);
  expect(reads).toHaveLength(0);
  const automatic = page.waitForResponse((response) => queryIs(response.request(), wos.namespace, "Alpha"));
  await page.clock.runFor(1);
  await automatic;
  await expect(rows(page)).toHaveCount(1);
  await expect(rows(page)).toContainText("Alpha delivery");
  expect(reads).toHaveLength(1);
  await input.fill("Beta");
  const immediate = page.waitForResponse((response) => queryIs(response.request(), wos.namespace, "Beta"));
  await input.press("Enter");
  await immediate;
  await expect(rows(page)).toHaveCount(1);
  await expect(rows(page)).toContainText("Beta delivery");
  await page.clock.runFor(1000);
  expect(reads).toHaveLength(2);
  expect(mutations).toEqual([]);
  expect((await wos.api(`/namespaces/${wos.namespace}/outcomes/${first.value.id}`)).value).toEqual(before);
});

test("a late sidebar response cannot overwrite a newer query or its accessible status", async ({ page, wos }) => {
  const first = await wos.outcome("Alpha delayed result");
  await wos.outcome("Beta current result");
  await sidebar(page, wos, first.value.title);
  const mutations = writes(page);
  const held = await holdRead(page, (request) => queryIs(request, wos.namespace, "Alpha"));
  try {
    await page.locator("#text").fill("Alpha");
    expect((await held.seen).value.items.map((item) => item.title)).toEqual(["Alpha delayed result"]);
    await search(page, wos, "Beta");
    await expect(rows(page)).toContainText("Beta current result");
  } finally { await held.deliver(); }
  await expect(rows(page)).toHaveCount(1);
  await expect(rows(page)).toContainText("Beta current result");
  await expect(page.locator("#outcome-search-status")).toHaveText(/^Outcomes shown: 1$/);
  await expect(page.locator("#notice")).not.toHaveClass(/error/);
  expect(mutations).toEqual([]);
});

test("changing a live query discards its old page cursor and keeps current pagination unique", async ({ page, wos }) => {
  let first;
  for (const prefix of ["Alpha", "Beta"])
    for (let index = 0; index < 31; index++) {
      const item = await wos.outcome(`${prefix} delivery ${String(index).padStart(2, "0")}`);
      if (!first) first = item;
    }
  await sidebar(page, wos, first.value.title);
  const mutations = writes(page);
  await search(page, wos, "Alpha");
  await expect(rows(page)).toHaveCount(25);
  const held = await holdRead(page, (request) => queryIs(request, wos.namespace, "Alpha", true));
  try {
    await page.locator("#more-outcomes").click();
    expect((await held.seen).value.items).toHaveLength(6);
    await search(page, wos, "Beta");
    await expect(rows(page)).toHaveCount(25);
  } finally { await held.deliver(); }
  await expect(rows(page)).toHaveCount(25);
  expect((await rows(page).allTextContents()).every((text) => text.includes("Beta delivery"))).toBe(true);
  const next = page.waitForResponse((response) => queryIs(response.request(), wos.namespace, "Beta", true));
  await page.locator("#more-outcomes").click();
  await next;
  await expect(rows(page)).toHaveCount(31);
  const ids = await rows(page).evaluateAll((buttons) => buttons.map((button) => button.dataset.outcomeId));
  expect(new Set(ids).size).toBe(31);
  expect((await rows(page).allTextContents()).every((text) => text.includes("Beta delivery"))).toBe(true);
  await expect(page.locator("#more-outcomes")).toBeHidden();
  expect(mutations).toEqual([]);
});

test("mobile specialist views retain local Refresh and immediate Outcome filters", async ({ page, wos }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  const first = await wos.outcome("Visible draft");
  const archived = await wos.outcome("Archived draft");
  await wos.command("archive_outcome", { scope: archived.scope, expected_version: archived.value.version });
  await sidebar(page, wos, first.value.title);
  const mutations = writes(page);
  for (const area of ["overview", "plan", "work", "issues", "evidence", "activity"]) {
    await chooseArea(page, area);
    await expect(page.locator("#outcome-tools")).toBeHidden();
    await expect(page.locator("#actions")).toBeHidden();
    await expect(page.locator("#refresh")).toHaveText("Refresh view");
    await expect(page.locator("#refresh")).toBeVisible();
  }
  await chooseArea(page, "overview");
  const toggle = page.locator("#navigation-toggle");
  if (await toggle.getAttribute("aria-expanded") === "false") await toggle.click();
  await page.locator("#search .search-filters > summary").click();
  // With time paused, filter requests must complete without advancing the
  // typing debounce. This distinguishes immediate filters from delayed input.
  await page.clock.install({ time: new Date("2026-10-18T13:00:00Z") });
  await page.clock.pauseAt(new Date("2026-10-18T13:00:01Z"));
  await page.locator("#text").fill("draft");
  const filtered = page.waitForResponse((response) => listRead(response.request(), wos.namespace) &&
    new URL(response.url()).searchParams.get("lifecycle") === "draft");
  await page.locator("#lifecycle").selectOption("draft");
  const filteredResponse = await filtered;
  expect(new URL(filteredResponse.url()).searchParams.has("cursor")).toBe(false);
  await expect(rows(page)).toHaveCount(1);
  const included = page.waitForResponse((response) => listRead(response.request(), wos.namespace) &&
    !new URL(response.url()).searchParams.has("archived"));
  await page.locator("#archived").check();
  expect(new URL((await included).url()).searchParams.has("cursor")).toBe(false);
  await expect(rows(page)).toHaveCount(2);
  await expect(rows(page).filter({ hasText: archived.value.title })).toHaveCount(1);
  await toggle.click();
  const refreshed = page.waitForResponse((response) => response.url().includes(`/outcomes/${first.value.id}/continuity?`));
  await page.locator("#refresh").click();
  await refreshed;
  await expect(page.locator('#area-tabs [data-area="overview"]')).toHaveAttribute("aria-pressed", "true");
  await expect(page.locator("#notice")).not.toHaveClass(/error/);
  await chooseArea(page, "canvas");
  await expect(page.locator("#outcome-tools")).toBeVisible();
  await expect(page.locator("#actions")).toBeVisible();
  await openViews(page);
  await page.keyboard.press("Escape");
  expect(mutations).toEqual([]);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
});
