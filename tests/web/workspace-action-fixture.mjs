import { expect } from "playwright/test";

// Only expand the real disclosure that contains an already authorized action.
// This never changes action availability or bypasses a hidden native dialog.
export async function revealMenuAction(target) {
  for (const disclosure of await target.locator("xpath=ancestor::details").all()) {
    if (!(await disclosure.evaluate((element) => element.open)))
      await disclosure.locator(":scope > summary").click();
  }
  await expect(target).toBeVisible();
}

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
    const target = page.locator("#action-options").getByRole("button", { name: label, exact: true, includeHidden: true });
    await revealMenuAction(target);
    await target.click();
  }
  await expect(page.locator("#command-form")).toBeVisible();
}

// Deeper regression journeys opt into the new progressive inspector through its
// real disclosures; compact-default assertions remain in contextual-kanban.spec.
export async function expandWorkspaceDetails(scope, { history = true } = {}) {
  const management = scope.locator('.workspace-management').first();
  if (await management.count() && !(await management.evaluate(node => node.open)))
    await management.locator(':scope > summary').click();
  if (history) {
    const activity = scope.locator('.workspace-history').first();
    if (await activity.count() && !(await activity.evaluate(node => node.open)))
      await activity.locator(':scope > summary').click();
  }
}
