import { test, expect, openOutcome } from "./ux-fixture.mjs";
import { seedCanvas } from "./canvas-fixture.mjs";

const authoredDescription = "Review the recorded installation steps against a clean environment. Preserve the exact commands and document each observed result so a second participant can review the delivery without recovering lost context from another session.";

async function initialReading(page) {
  return page.locator("#canvas-details").evaluate((panel) => {
    const prose = panel.querySelector('.workspace-compact-intro .detail-description');
    const bounds = panel.getBoundingClientRect(), rect = prose.getBoundingClientRect();
    const lineHeight = parseFloat(getComputedStyle(prose).lineHeight);
    return {
      panel: { y: bounds.y, bottom: bounds.bottom, height: bounds.height, scrollTop: panel.scrollTop },
      prose: { y: rect.y, bottom: rect.bottom, height: rect.height, lineHeight,
        visibleHeight: Math.min(rect.bottom, bounds.bottom, innerHeight) - Math.max(rect.top, bounds.top) },
      scrolls: [...panel.querySelectorAll(".canvas-inspector-detail, #detail-content")].map((element) => element.scrollTop),
      context: [...panel.querySelectorAll("nav.workspace-path > button")].map((button) => {
        const box = button.getBoundingClientRect();
        return { name: button.textContent, height: box.height, width: box.width };
      }),
    };
  });
}

for (const viewport of [{ width: 320, height: 740 }, { width: 390, height: 844 }, { width: 430, height: 932 }]) {
  test(`a nested Task initially exposes authored prose and retains reachable actions at ${viewport.width}px`, async ({ page, wos }) => {
    await page.setViewportSize(viewport);
    const data = await seedCanvas(wos, { title: "Plan, deliver, and verify together" });
    const task = data.tasks.ready;
    await wos.command("update_work_item", { scope: data.scope, work_item_id: task.id,
      expected_version: task.version, title: task.title, description: authoredDescription, priority: task.priority });
    const errors = [], mutations = [];
    page.on("pageerror", (error) => errors.push(error.message));
    page.on("request", (request) => {
      if (request.method() === "POST" && request.url().includes("/commands/")) mutations.push(request.url());
    });
    await wos.login(page);
    await openOutcome(page, data.outcome.title, { area: null });
    await page.getByRole("button", { name: "Map view", exact: true }).click();
    const layer = page.getByRole("combobox", { name: "Map layers", exact: true });
    const layerSize = await layer.evaluate((select) => {
      const style = getComputedStyle(select), rect = select.getBoundingClientRect();
      const drawing = document.createElement("canvas").getContext("2d");
      drawing.font = style.font;
      return { height: rect.height, fontSize: parseFloat(style.fontSize),
        selectedText: drawing.measureText(select.selectedOptions[0].textContent).width,
        // Chromium retains this native inset in addition to CSS padding;
        // use the same measured allowance as the native-zoom regression.
        textSpace: select.clientWidth - parseFloat(style.paddingLeft) - parseFloat(style.paddingRight) - 12 };
    });
    expect(layerSize.height).toBeGreaterThanOrEqual(44);
    expect(layerSize.fontSize).toBeGreaterThanOrEqual(14);
    expect(layerSize.textSpace, "The selected Work map label fits beside its native arrow").toBeGreaterThanOrEqual(layerSize.selectedText);
    await page.getByRole("combobox", { name: "Search Outcome Board", exact: true }).fill(task.title);
    await page.getByRole("option").filter({ hasText: task.title }).first().click();
    const panel = page.locator("#canvas-details");
    await expect(panel.locator("#detail-title")).toHaveText(task.title);
    const prose = panel.locator('.workspace-compact-intro .detail-description');
    await expect(prose).toHaveText(authoredDescription);
    await page.evaluate(() => document.fonts.ready);
    const before = await initialReading(page);
    await test.info().attach("initial-nested-task-geometry", { body: JSON.stringify(before, null, 2), contentType: "application/json" });
    await page.screenshot({ path: test.info().outputPath("initial-nested-task.png"), fullPage: false });
    expect(before.panel.scrollTop).toBe(0);
    expect(before.scrolls.every((top) => top === 0)).toBe(true);
    expect(before.context.map((item) => item.name)).toEqual([data.outcome.title, data.child.title]);
    for (const control of before.context) expect(control.height).toBeGreaterThanOrEqual(44);
    expect(before.prose.visibleHeight, "The actual initial nested Task view exposes at least three authored lines")
      .toBeGreaterThanOrEqual(Math.min(before.prose.height, before.prose.lineHeight * 3) - 2);

    const more = panel.locator("#entity-actions");
    await more.focus();
    await expect(more).toBeFocused();
    expect(await more.evaluate((button) => {
      const box = button.getBoundingClientRect(), hit = document.elementFromPoint(box.x + box.width / 2, box.y + box.height / 2);
      return box.height >= 44 && box.width >= 44 && box.top >= 0 && box.bottom <= innerHeight && button.contains(hit);
    })).toBe(true);
    await more.click();
    await expect(page.locator(".canvas-inspector-actions")).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(panel).toHaveAttribute("data-surface", "detail");
    await panel.locator("#close-detail").focus();
    await panel.hover();
    await page.mouse.wheel(0, -10000);
    await expect.poll(async () => (await initialReading(page)).panel.scrollTop).toBe(0);
    const returned = await initialReading(page);
    expect(returned.prose.visibleHeight).toBeGreaterThanOrEqual(Math.min(returned.prose.height, returned.prose.lineHeight * 3) - 2);
    await test.info().attach("returned-nested-task-geometry", { body: JSON.stringify(returned, null, 2), contentType: "application/json" });
    await panel.locator("#close-detail").click();
    await expect(panel).toBeHidden();
    expect(await page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(0);
    expect(mutations).toEqual([]);
    expect(errors).toEqual([]);
  });
}
