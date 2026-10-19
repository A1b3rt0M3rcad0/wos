import { expect } from "playwright/test";

// Follow the shared inspector's compact primary/secondary/More hierarchy.
// The helper only operates visible controls; it never invokes app internals.
export async function clickRecordAction(page, label) {
  const inspector = page.locator("#canvas-details:visible, #detail-dialog[open]").first();
  const scope = await inspector.count() ? inspector : page;
  const direct = scope.getByRole("button", { name: label, exact: true });
  if (await direct.first().isVisible()) await direct.first().click();
  else {
    const more = scope.locator("[data-action-id=more_actions], #entity-actions").filter({ visible: true }).first();
    await more.click();
    const target = page.locator("#action-options").getByRole("button", { name: label, exact: true });
    if (!(await target.isVisible())) {
      const advanced = page.locator("#action-options details").filter({ has: page.locator("summary").getByText("Advanced", { exact: true }) });
      if (await advanced.count() && !(await advanced.evaluate((element) => element.open)))
        await advanced.locator(":scope > summary").click();
    }
    await target.click();
  }
  await expect(page.locator("#command-form")).toBeVisible();
}
