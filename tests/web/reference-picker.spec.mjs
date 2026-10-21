import { readFileSync } from "node:fs";
import { test, expect } from "playwright/test";

const assets = new URL("../../packages/wos-api/web/assets/", import.meta.url);

async function fixture(page, { multiple = false } = {}) {
  await page.route("http://reference-picker.test/**", async (route) => {
    const pathname = new URL(route.request().url()).pathname;
    if (/^\/assets\/[a-zA-Z0-9.-]+\.js$/.test(pathname))
      await route.fulfill({ contentType: "text/javascript", body: readFileSync(new URL(pathname.slice(8), assets), "utf8") });
    else
      await route.fulfill({ contentType: "text/html", body: '<!doctype html><html lang="en"><title>Reference picker</title><body><button id="outside">Elsewhere</button><main id="picker"></main></body></html>' });
  });
  await page.goto("http://reference-picker.test/");
  await page.evaluate(async ({ multiple }) => {
    const { referencePicker } = await import("/assets/reference-picker.js");
    const scope = { namespace_id: "namespace", outcome_id: "outcome" };
    const candidate = (id, title) => ({ ref: { ...scope, kind: "work_item", id }, title, lifecycle: "todo" });
    const choices = [candidate("first", "First task"), candidate("second", "Second task")];
    window.queries = [];
    window.holdNext = false;
    window.picker = referencePicker({
      label: "Depends on", kinds: ["work_item"], scope, multiple,
      request: async (path, options) => {
        window.queries.push(new URL(path, location.origin).searchParams.get("query"));
        if (window.holdNext) {
          window.holdNext = false;
          window.heldSignal = options.signal;
          // The response may already have arrived when abort is requested.
          // Deliberately deliver it anyway to exercise the generation fence.
          return new Promise((resolve) => {
            window.releaseRead = () => resolve({ items: [candidate("late", "Obsolete late result")] });
          });
        }
        return { items: choices };
      },
    });
    document.querySelector("#picker").append(window.picker.node);
  }, { multiple });
  const instant = new Date("2026-10-19T12:00:00Z");
  await page.clock.install({ time: instant });
  // Pause ahead of the installed time so transport/runner scheduling cannot
  // turn this deterministic debounce fixture into a backwards clock jump.
  await page.clock.pauseAt(new Date(instant.getTime() + 60_000));
}

test("choosing a reference cancels a queued debounce and fences older results that still arrive", async ({ page }) => {
  await fixture(page);
  const input = page.getByRole("combobox", { name: "Depends on", exact: true });
  await input.focus();
  await expect(page.getByRole("option", { name: /^First task/ })).toBeVisible();
  await input.fill("queued search");
  await page.getByRole("option", { name: /^First task/ }).click();
  await page.clock.runFor(500);
  expect(await page.evaluate(() => window.queries)).toEqual([""]);
  await expect(input).toHaveAttribute("aria-expanded", "false");
  expect(await page.evaluate(() => window.picker.get().id)).toBe("first");

  await page.locator("#outside").focus();
  await input.focus();
  await expect(page.getByRole("option", { name: /^Second task/ })).toBeVisible();
  await page.evaluate(() => { window.holdNext = true; });
  await input.fill("already dispatched");
  await page.clock.runFor(180);
  expect(await page.evaluate(() => typeof window.releaseRead)).toBe("function");
  await page.getByRole("option", { name: /^Second task/ }).click();
  expect(await page.evaluate(() => window.heldSignal.aborted)).toBe(true);
  await page.evaluate(() => window.releaseRead());
  await page.clock.runFor(500);
  await expect(input).toHaveAttribute("aria-expanded", "false");
  await expect(page.getByRole("option")).toHaveCount(0);
  expect(await page.evaluate(() => window.queries)).toEqual(["", "", "already dispatched"]);
  expect(await page.evaluate(() => window.picker.get().id)).toBe("second");
});

test("Escape cancels queued and in-flight suggestions without clearing the search or reopening the popup", async ({ page }) => {
  await fixture(page);
  const input = page.getByRole("combobox", { name: "Depends on", exact: true });
  await input.focus();
  await expect(page.getByRole("option", { name: /^First task/ })).toBeVisible();
  await input.fill("queued search");
  await page.keyboard.press("Escape");
  await page.clock.runFor(500);
  await expect(input).toHaveValue("queued search");
  await expect(input).toHaveAttribute("aria-expanded", "false");
  expect(await page.evaluate(() => window.queries)).toEqual([""]);

  await page.keyboard.press("ArrowDown");
  await expect(page.getByRole("option", { name: /^First task/ })).toBeVisible();
  await page.evaluate(() => { window.holdNext = true; });
  await input.fill("in-flight search");
  await page.clock.runFor(180);
  expect(await page.evaluate(() => typeof window.releaseRead)).toBe("function");
  await page.keyboard.press("Escape");
  expect(await page.evaluate(() => window.heldSignal.aborted)).toBe(true);
  await page.evaluate(() => window.releaseRead());
  await page.clock.runFor(500);
  await expect(input).toHaveValue("in-flight search");
  await expect(input).toHaveAttribute("aria-expanded", "false");
  await expect(page.getByRole("option")).toHaveCount(0);
  expect(await page.evaluate(() => window.queries)).toEqual(["", "queued search", "in-flight search"]);
});

test("multiple selection keeps its explicit fresh suggestion search and does not replay an old timer", async ({ page }) => {
  await fixture(page, { multiple: true });
  const input = page.getByRole("combobox", { name: "Depends on", exact: true });
  await input.focus();
  await expect(page.getByRole("option", { name: /^First task/ })).toBeVisible();
  await input.fill("queued before selection");
  await page.getByRole("option", { name: /^First task/ }).click();
  await expect(input).toHaveAttribute("aria-expanded", "true");
  await expect(page.getByRole("option", { name: /^Second task/ })).toBeVisible();
  await page.clock.runFor(500);
  expect(await page.evaluate(() => window.queries)).toEqual(["", ""]);
  await page.getByRole("option", { name: /^Second task/ }).click();
  await expect(page.locator(".reference-chip")).toHaveCount(2);
  await expect(input).toHaveAttribute("aria-expanded", "true");
  expect(await page.evaluate(() => window.picker.get().map((value) => value.id))).toEqual(["first", "second"]);
  expect(await page.evaluate(() => window.queries)).toEqual(["", "", ""]);
});
