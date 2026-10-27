import { expect } from "./ux-fixture.mjs";

/** Drag a real Kanban card to a lane, including lanes outside the horizontal viewport. */
export async function dragKanbanCardToLane(page, card, lane) {
  const columns = page.locator(".workspace-kanban-columns");
  await card.scrollIntoViewIfNeeded();
  let container = await columns.boundingBox();
  let cardBounds = await card.boundingBox();
  let laneBounds = await lane.boundingBox();
  if (!container || !cardBounds || !laneBounds)
    throw Error("Kanban drag geometry is unavailable.");

  const visible = (bounds) => bounds.x >= container.x
    && bounds.x + bounds.width <= container.x + container.width;
  if (visible(laneBounds)) {
    await card.dragTo(lane);
    return;
  }

  const start = {
    x: cardBounds.x + cardBounds.width / 2,
    y: cardBounds.y + cardBounds.height / 2,
  };
  await page.mouse.move(start.x, start.y);
  await page.mouse.down();
  await page.mouse.move(start.x + 12, start.y + 12, { steps: 2 });
  await expect.poll(() => columns.getAttribute("data-dragging"), {
    message: "The native drag must start before scrolling the Board lanes.",
  }).toBe("true");

  for (let attempt = 0; attempt < 80 && !visible(laneBounds); attempt++) {
    container = await columns.boundingBox();
    laneBounds = await lane.boundingBox();
    if (!container || !laneBounds) break;
    const edgeX = laneBounds.x < container.x
      ? container.x + 3
      : container.x + container.width - 3;
    await page.mouse.move(edgeX, start.y, { steps: 2 });
    await page.waitForTimeout(75);
    container = await columns.boundingBox();
    laneBounds = await lane.boundingBox();
    if (!container || !laneBounds) break;
  }

  if (!container || !laneBounds || !visible(laneBounds)) {
    await page.mouse.up();
    throw Error("Dragging at the Board edge did not reveal the requested lane.");
  }

  // Drop away from the edge threshold so another autoscroll step cannot move
  // the intended lane underneath a stationary pointer before the drop.
  const edge = Math.min(64, container.width / 4);
  const minX = Math.max(laneBounds.x + 16, container.x + edge + 2);
  const maxX = Math.min(laneBounds.x + laneBounds.width - 16,
    container.x + container.width - edge - 2);
  const x = minX <= maxX
    ? Math.min(maxX, Math.max(minX, laneBounds.x + laneBounds.width / 2))
    : laneBounds.x + laneBounds.width / 2;
  const y = laneBounds.y + Math.min(60, laneBounds.height / 2);
  await page.mouse.move(x, y, { steps: 8 });
  await page.mouse.up();
}
