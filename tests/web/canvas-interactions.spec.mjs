import { test, expect } from "playwright/test";
import { createServer } from "node:http";
import { readFile } from "node:fs/promises";
import { dirname, extname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { createHash } from "node:crypto";
import AxeBuilder from "@axe-core/playwright";

// These are renderer tests, independent of transport/domain integration in
// canvas.spec.mjs. Serve the real assets under the production CSP and observe
// callbacks: moving presentation data must never create an implicit command.
const assets = resolve(
  dirname(fileURLToPath(import.meta.url)),
  "../../packages/wos-api/web/assets",
);
let server, origin;
test.beforeAll(async () => {
  server = createServer(async (request, response) => {
    response.setHeader(
      "Content-Security-Policy",
      "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self'; connect-src 'self'; object-src 'none'",
    );
    if (request.url === "/") {
      response.setHeader("Content-Type", "text/html");
      response.end(
        `<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Canvas renderer</title>${["colors", "style", "editorial", "canvas"].map((name) => `<link rel="stylesheet" href="/app/${name}.css">`).join("")}<body><main><div class="canvas-workspace-shell"><div id="operational-canvas"></div></div></main></body></html>`,
      );
      return;
    }
    try {
      const file = resolve(
        assets,
        decodeURIComponent(
          new URL(request.url, "http://localhost").pathname,
        ).replace(/^\/app\//, ""),
      );
      if (!file.startsWith(assets + "/")) throw Error("Outside asset root");
      response.setHeader(
        "Content-Type",
        {
          ".js": "text/javascript",
          ".css": "text/css",
          ".svg": "image/svg+xml",
          ".woff2": "font/woff2",
        }[extname(file)] || "application/octet-stream",
      );
      response.end(await readFile(file));
    } catch {
      response.writeHead(404);
      response.end();
    }
  });
  await new Promise((done) => server.listen(0, "127.0.0.1", done));
  origin = `http://127.0.0.1:${server.address().port}`;
});
test.afterAll(async () => new Promise((done) => server.close(done)));

async function mount(page) {
  await page.goto(origin);
  await page.evaluate(async () => {
    const { mountWorkspaceCanvas } = await import("/app/workspace-canvas.js");
    const ref = (kind, id) => ({ kind, id });
    window.canvasEvents = {
      select: [],
      edge: [],
      create: [],
      connect: [],
      expand: [],
      search: [],
    };
    window.searchPending = [];
    window.canvasModel = {
      scopeKey: "tenant/outcome",
      rootKey: "outcome:root",
      revision: 1,
      partial: true,
      hasMore: false,
      nodes: [
        {
          key: "outcome:root",
          ref: ref("outcome", "root"),
          kind: "outcome",
          title: "Ship a reliable service",
          lifecycle: "active",
          canCreate: ["objective", "work_item"],
          canConnect: true,
        },
        {
          key: "objective:one",
          ref: ref("objective", "one"),
          kind: "objective",
          title: "Handle long work safely",
          parentKey: "outcome:root",
          lifecycle: "active",
          canCreate: ["objective", "work_item"],
        },
        {
          key: "work_item:a",
          ref: ref("work_item", "a"),
          kind: "work_item",
          title: "Remote Alpha",
          parentKey: "objective:one",
          lifecycle: "planned",
          canConnect: true,
        },
        {
          key: "work_item:b",
          ref: ref("work_item", "b"),
          kind: "work_item",
          title: "Review integration",
          parentKey: "objective:one",
          lifecycle: "planned",
        },
        {
          key: "issue:one",
          ref: ref("issue", "one"),
          kind: "issue",
          title: "A documented risk",
          lifecycle: "open",
        },
      ],
      edges: [
        {
          key: "h1",
          sourceKey: "outcome:root",
          targetKey: "objective:one",
          kind: "hierarchy",
        },
        {
          key: "o1",
          sourceKey: "objective:one",
          targetKey: "work_item:a",
          kind: "ownership",
        },
        {
          key: "o2",
          sourceKey: "objective:one",
          targetKey: "work_item:b",
          kind: "ownership",
        },
        {
          key: "graph:relation:dep",
          ref: ref("relation", "dep"),
          sourceKey: "work_item:a",
          targetKey: "work_item:b",
          kind: "depends_on",
          mode: "hard",
        },
        {
          key: "documentary",
          sourceKey: "issue:one",
          targetKey: "work_item:a",
          kind: "context",
        },
      ],
    };
    window.canvasHandle = mountWorkspaceCanvas(
      document.querySelector("#operational-canvas"),
      {
        model: window.canvasModel,
        onSelect: (ref, item) => {
          canvasEvents.select.push([ref, item]);
          if (window.canvasSelectionError) {
            window.canvasRejectedSelection =
              Promise.reject(canvasSelectionError);
            return canvasRejectedSelection;
          }
          if (window.mirrorPlanningSelection && item.roadmapRef) {
            canvasEvents.inspector = item.roadmapRef;
            canvasHandle.update(canvasModel, {
              selectedKey: `${item.roadmapRef.kind}:${item.roadmapRef.id}`,
            });
          }
        },
        onSelectEdge: (edge) => canvasEvents.edge.push(edge),
        onCreate: (...args) => canvasEvents.create.push(args),
        onConnect: (value) => canvasEvents.connect.push(value),
        onExpand: (...args) => canvasEvents.expand.push(args),
        onSearch: async (query, { signal, cursor }) => {
          canvasEvents.search.push({ query, cursor });
          if (query === "pending")
            return new Promise((done) => searchPending.push({ done, signal }));
          const item = (id, title) => ({
            key: `evidence:${id}`,
            ref: ref("evidence", id),
            kind: "evidence",
            title,
          });
          if (query === "remote")
            return cursor
              ? {
                  items: [
                    item("one", "Remote Beta"),
                    item("two", "Remote Gamma"),
                  ],
                  next_cursor: "",
                }
              : {
                  items: [canvasModel.nodes[2], item("one", "Remote Beta")],
                  next_cursor: "next-page",
                };
          return { items: [], next_cursor: "" };
        },
      },
    );
  });
  await expect(page.locator(".canvas-world")).toHaveCSS("transform", /matrix/);
}
const card = (page, key) => page.locator(`[data-canvas-node="${key}"]`);
const search = (page) => page.getByRole("combobox", { name: "Search canvas" });
const result = (page, title) =>
  page
    .getByRole("listbox", { name: "Matching items" })
    .getByRole("option", { name: title, exact: true });

test("native canvas controls preserve selection, keyboard focus and explicit command boundaries", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1440, height: 1100 });
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await mount(page);
  await card(page, "outcome:root").locator(".canvas-node-main").click();
  expect(await page.evaluate(() => canvasEvents.select.length)).toBe(1);
  const zoom = await page.getByLabel("Canvas zoom").textContent();
  await page.getByRole("button", { name: "Zoom in", exact: true }).click();
  await expect(page.getByLabel("Canvas zoom")).not.toHaveText(zoom);
  await page.evaluate(() => canvasHandle.focus("objective:one"));
  await card(page, "objective:one").locator(".canvas-expand").click();
  await expect(card(page, "work_item:a")).toHaveCount(0);
  await expect(
    card(page, "objective:one").locator(".canvas-expand"),
  ).toBeFocused();
  await card(page, "objective:one").locator(".canvas-expand").click();
  await expect(card(page, "work_item:a")).toHaveCount(1);
  await expect(
    card(page, "objective:one").locator(".canvas-expand"),
  ).toBeFocused();
  expect(
    await page.evaluate(() => canvasEvents.expand.map((entry) => entry[1])),
  ).toEqual([false, true]);
  await page.evaluate(() => canvasHandle.focus("outcome:root"));
  await card(page, "outcome:root").locator(".canvas-add").click();
  await card(page, "outcome:root")
    .getByRole("button", { name: "New task", exact: true })
    .click();
  expect(await page.evaluate(() => canvasEvents.create)).toEqual([
    ["work_item", { kind: "outcome", id: "root" }],
  ]);
  await card(page, "outcome:root").locator(".canvas-add").click();
  await page.keyboard.press("Escape");
  await expect(card(page, "outcome:root").locator(".canvas-add")).toBeFocused();
  await page.getByLabel("Connections", { exact: true }).selectOption("all");
  await page.evaluate(() => canvasHandle.focus("relation:dep"));
  const edge = page.locator('[data-edge-key="graph:relation:dep"]');
  await edge.press("Enter");
  await expect(edge).toBeFocused();
  // A coherent model refresh rebuilds the SVG ancestor. Preserve the exact
  // focused relationship before detaching it, not only in renderEdges().
  await page.evaluate(() => canvasHandle.update({ ...canvasModel }));
  await expect(edge).toBeFocused();
  expect(
    await page.evaluate(() => canvasEvents.edge.map((entry) => entry.ref)),
  ).toEqual([{ kind: "relation", id: "dep" }]);
  await page.evaluate(() => canvasHandle.focus("work_item:a"));
  const before = await card(page, "work_item:a").evaluate((element) => ({
    left: element.style.left,
    top: element.style.top,
  }));
  const box = await card(page, "work_item:a")
    .locator(".canvas-node-main")
    .boundingBox();
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
  await page.mouse.down();
  await page.mouse.move(
    box.x + box.width / 2 + 80,
    box.y + box.height / 2 + 40,
    { steps: 4 },
  );
  await page.mouse.up();
  const after = await card(page, "work_item:a").evaluate((element) => ({
    left: element.style.left,
    top: element.style.top,
  }));
  expect(after).not.toEqual(before);
  await page.evaluate(() =>
    canvasHandle.update({ ...canvasModel, revision: 2 }),
  );
  expect(
    await card(page, "work_item:a").evaluate((element) => ({
      left: element.style.left,
      top: element.style.top,
    })),
  ).toEqual(after);
  expect(await page.evaluate(() => canvasEvents.connect)).toEqual([]);
  await page.evaluate(() => canvasHandle.focus("work_item:b"));
  const releaseBox = await card(page, "work_item:b")
    .locator(".canvas-node-main")
    .boundingBox();
  await page.mouse.move(releaseBox.x + 2, releaseBox.y + 2);
  await page.mouse.down();
  // Simulate losing the pointer outside before any captured movement.
  await page.evaluate(() =>
    window.dispatchEvent(new PointerEvent("pointerup", { pointerId: 1 })),
  );
  await page.mouse.up();
  await page.evaluate(() =>
    canvasHandle.update({
      ...canvasModel,
      revision: 3,
      nodes: [
        ...canvasModel.nodes,
        { key: "evidence:later", kind: "evidence", title: "Newly loaded item" },
      ],
    }),
  );
  await expect(card(page, "evidence:later")).toHaveCount(1);
  expect(errors).toEqual([]);
});

test("canvas search paginates and deduplicates without reopening stale queries or scopes", async ({
  page,
}) => {
  await mount(page);
  await search(page).fill("remote");
  await result(page, "Task · Remote Alpha").waitFor();
  await page
    .getByRole("button", { name: "Load more search results", exact: true })
    .click();
  await result(page, "Evidence · Remote Gamma").waitFor();
  await expect(
    page.getByRole("listbox", { name: "Matching items" }).getByRole("option"),
  ).toHaveCount(3);
  expect(
    await page.evaluate(() =>
      canvasEvents.search
        .filter((entry) => entry.query === "remote")
        .map((entry) => entry.cursor),
    ),
  ).toEqual(["", "next-page"]);
  await search(page).press("ArrowDown");
  await search(page).press("Enter");
  expect(await page.evaluate(() => canvasEvents.select.at(-1)[0])).toEqual({
    kind: "work_item",
    id: "a",
  });
  await search(page).fill("pending");
  await expect.poll(() => page.evaluate(() => searchPending.length)).toBe(1);
  await search(page).fill("review");
  await page.evaluate(() =>
    searchPending[0].done({
      items: [
        {
          key: "evidence:stale",
          kind: "evidence",
          title: "Wrong old response",
        },
      ],
      next_cursor: "stale-page",
    }),
  );
  await result(page, "Task · Review integration").waitFor();
  await expect(
    page.getByText("Wrong old response", { exact: false }),
  ).toHaveCount(0);
  await search(page).fill("pending");
  await search(page).press("Escape");
  await page.waitForTimeout(300); // Longer than the renderer's pending debounce.
  expect(await page.evaluate(() => searchPending.length)).toBe(1);
  await expect(search(page)).toHaveAttribute("aria-expanded", "false");
  await search(page).fill("pending");
  await expect.poll(() => page.evaluate(() => searchPending.length)).toBe(2);
  await search(page).press("Escape");
  expect(await page.evaluate(() => searchPending[1].signal.aborted)).toBe(true);
  await page.evaluate(() =>
    searchPending[1].done({ items: [canvasModel.nodes[2]], next_cursor: "" }),
  );
  await expect(search(page)).toHaveAttribute("aria-expanded", "false");
  await search(page).press("Enter");
  expect(await page.evaluate(() => canvasEvents.select.length)).toBe(1);
  await search(page).fill("pending");
  await expect.poll(() => page.evaluate(() => searchPending.length)).toBe(3);
  await page.evaluate(() =>
    canvasHandle.update({ ...canvasModel, revision: 2 }),
  );
  await expect.poll(() => page.evaluate(() => searchPending.length)).toBe(4);
  expect(await page.evaluate(() => searchPending[2].signal.aborted)).toBe(true);
  await page.evaluate(() =>
    searchPending[3].done({ items: [canvasModel.nodes[2]], next_cursor: "" }),
  );
  await result(page, "Task · Remote Alpha").waitFor();
  await search(page).press("Escape");
  await page.evaluate(() =>
    canvasHandle.update({ ...canvasModel, revision: 3 }),
  );
  await expect(search(page)).toHaveAttribute("aria-expanded", "false");
  expect(await page.evaluate(() => searchPending.length)).toBe(4);
  await expect(page.locator(".canvas-status")).toContainText("revision 3");
  await search(page).fill("pending");
  await expect.poll(() => page.evaluate(() => searchPending.length)).toBe(5);
  await page.evaluate(() => {
    canvasHandle.update({ ...canvasModel, scopeKey: "different-scope" });
    searchPending[4].done({ items: [canvasModel.nodes[2]], next_cursor: "" });
  });
  await expect(search(page)).toHaveValue("");
  await expect(search(page)).toHaveAttribute("aria-expanded", "false");
  await search(page).fill("pending");
  await expect.poll(() => page.evaluate(() => searchPending.length)).toBe(6);
  await page.evaluate(() => {
    canvasHandle.dispose();
    searchPending[5].done({ items: [canvasModel.nodes[2]], next_cursor: "" });
  });
  await expect(page.locator(".workspace-canvas")).toHaveCount(0);
});

test("responsive UI scaling remains separate from camera and mobile has a native list alternative", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1440, height: 1100 });
  await mount(page);
  await page.evaluate(() => canvasHandle.focus("work_item:b"));
  const zoom = await page.getByLabel("Canvas zoom").textContent();
  const normal = await card(page, "work_item:b").boundingBox();
  await page.setViewportSize({ width: 2560, height: 1440 });
  await expect(page.getByLabel("Canvas zoom")).toHaveText(zoom);
  const larger = await card(page, "work_item:b").boundingBox();
  expect(larger.width / normal.width).toBeCloseTo(2560 / 1440, 2);
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(page.locator(".workspace-canvas")).toHaveAttribute(
    "data-view",
    "list",
  );
  await page
    .locator('[data-canvas-list-node="work_item:b"] .canvas-list-main')
    .click();
  expect(await page.evaluate(() => canvasEvents.select.at(-1)[0])).toEqual({
    kind: "work_item",
    id: "b",
  });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
  await page.getByRole("button", { name: "Map view", exact: true }).click();
  await expect(page.locator(".canvas-viewport")).toBeVisible();
  await page.getByRole("button", { name: "List view", exact: true }).click();
  await expect(page.locator(".canvas-list")).toBeVisible();
});

test("overview zoom preserves accessible controls through item selection and the list alternative", async ({
  page,
}) => {
  await mount(page);
  for (let index = 0; index < 10; index++)
    await page.getByRole("button", { name: "Zoom out", exact: true }).click();
  await expect(page.getByLabel("Canvas zoom")).toHaveText("25%");
  await expect(page.locator(".workspace-canvas")).toHaveAttribute(
    "data-detail",
    "overview",
  );
  await expect(
    card(page, "objective:one").locator(".canvas-expand"),
  ).not.toBeVisible();
  await page.evaluate(() => canvasHandle.focus("work_item:a"));
  await card(page, "work_item:a").locator(".canvas-node-main").click();
  expect(await page.evaluate(() => canvasEvents.select.at(-1)[0])).toEqual({
    kind: "work_item",
    id: "a",
  });
  const audit = await new AxeBuilder({ page })
    .include(".workspace-canvas")
    .withTags(["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"])
    .analyze();
  expect(
    audit.violations.filter((entry) =>
      ["critical", "serious"].includes(entry.impact),
    ),
  ).toEqual([]);
  await page.getByRole("button", { name: "List view", exact: true }).click();
  await expect(
    page.locator('[data-canvas-list-node="objective:one"] .canvas-expand'),
  ).toBeVisible();
});

test("a planning card remains selected when its inspector opens the real Roadmap", async ({
  page,
}) => {
  await mount(page);
  const key = "plan:roadmap-one:draft1:phase-one";
  await page.evaluate((key) => {
    window.canvasModel = {
      ...canvasModel,
      nodes: [
        ...canvasModel.nodes,
        {
          key,
          kind: "phase",
          title: "First delivery phase",
          ref: null,
          roadmapRef: { kind: "roadmap", id: "roadmap-one" },
          sourceOfTruth: "roadmap_draft",
          lifecycle: "draft",
        },
      ],
    };
    window.mirrorPlanningSelection = true;
    canvasHandle.update(canvasModel);
    canvasHandle.focus(key);
  }, key);
  await card(page, key).locator(".canvas-node-main").click();
  await expect(card(page, key)).toHaveAttribute("data-selected", "true");
  await expect(card(page, key).locator(".canvas-node-main")).toHaveAttribute(
    "aria-pressed",
    "true",
  );
  expect(await page.evaluate(() => canvasEvents.select.length)).toBe(1);
  expect(await page.evaluate(() => canvasEvents.select[0][0])).toBeNull();
  expect(await page.evaluate(() => canvasEvents.inspector)).toEqual({
    kind: "roadmap",
    id: "roadmap-one",
  });
  await expect(card(page, "roadmap:roadmap-one")).toHaveCount(0);
  expect(
    await page.evaluate(() => [canvasEvents.create, canvasEvents.connect]),
  ).toEqual([[], []]);

  await page.evaluate(() =>
    canvasHandle.update(canvasModel, { selectedKey: null }),
  );
  await expect(card(page, key)).toHaveAttribute("data-selected", "false");
  await card(page, key).locator(".canvas-node-main").click();
  await page.evaluate(() =>
    canvasHandle.update(canvasModel, { selectedKey: "roadmap:different" }),
  );
  await expect(card(page, key)).toHaveAttribute("data-selected", "false");
  await card(page, key).locator(".canvas-node-main").click();
  await page.evaluate(() =>
    canvasHandle.update(
      { ...canvasModel, scopeKey: "another-scope" },
      { selectedKey: "roadmap:roadmap-one" },
    ),
  );
  await expect(card(page, key)).toHaveAttribute("data-selected", "false");
});

test("discarded callback failures stay silent while current failures remain announced", async ({
  page,
}) => {
  await mount(page);
  const status = page.locator(".canvas-status");
  const select = card(page, "outcome:root").locator(".canvas-node-main");
  await page.evaluate(() => {
    window.canvasSelectionError = Object.assign(
      new Error("Obsolete selection failed"),
      { discarded: true },
    );
  });
  await select.click();
  await page.evaluate(() => canvasRejectedSelection.catch(() => {}));
  await expect(status).not.toContainText("Obsolete selection failed");
  await page.evaluate(() => {
    window.canvasSelectionError = new DOMException(
      "Selection request was cancelled",
      "AbortError",
    );
  });
  await select.click();
  await page.evaluate(() => canvasRejectedSelection.catch(() => {}));
  await expect(status).not.toContainText("Selection request was cancelled");
  await page.evaluate(() => {
    window.canvasSelectionError = new Error(
      "Current selection could not be loaded",
    );
  });
  await select.click();
  await page.evaluate(() => canvasRejectedSelection.catch(() => {}));
  await expect(status).toContainText("Current selection could not be loaded");
  await page.evaluate(() => {
    window.canvasSelectionError = Object.assign(
      new Error("Late obsolete failure"),
      { discarded: true },
    );
  });
  await select.click();
  await page.evaluate(() => canvasRejectedSelection.catch(() => {}));
  await expect(status).toContainText("Current selection could not be loaded");
  await expect(status).not.toContainText("Late obsolete failure");
});

test("all map cards display the original supplied paper behind live accessible content", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1440, height: 1100 });
  const papers = [
    {
      kind: "outcome",
      format: "wide",
      style: "pinned",
      color: "surface",
      width: 336,
      top: 22,
    },
    {
      kind: "objective",
      format: "square",
      style: "tabbed",
      color: "workspace",
      width: 252,
      top: 22,
    },
    {
      kind: "work_item",
      format: "wide",
      style: "curled",
      color: "surface",
      width: 284,
      top: 16,
      bottom: 17,
    },
    {
      kind: "issue",
      format: "wide",
      style: "taped",
      color: "workspace",
      width: 284,
      top: 22,
    },
    {
      kind: "blocker",
      format: "wide",
      style: "torn",
      color: "workspace",
      width: 284,
      top: 16,
    },
    {
      kind: "evidence",
      format: "wide",
      style: "notebook",
      color: "surface",
      width: 284,
      top: 22,
    },
    {
      kind: "artifact",
      format: "wide",
      style: "stacked",
      color: "surface",
      width: 284,
      top: 16,
    },
    {
      kind: "decision",
      format: "wide",
      style: "dogear",
      color: "workspace",
      width: 284,
      top: 16,
    },
    {
      kind: "phase",
      format: "wide",
      style: "rounded",
      color: "workspace",
      width: 252,
      top: 16,
    },
    {
      kind: "milestone",
      format: "wide",
      style: "pinned",
      color: "surface",
      width: 252,
      top: 22,
    },
    {
      kind: "plan_reference",
      format: "wide",
      style: "classic",
      color: "workspace",
      width: 252,
      top: 16,
    },
    {
      kind: "annotation",
      format: "wide",
      style: "rounded",
      color: "workspace",
      width: 284,
      top: 16,
    },
  ];
  const provenance = JSON.parse(
    await readFile(
      resolve(
        assets,
        "../../../../docs/design/operational-canvas/postits/assets.json",
      ),
      "utf8",
    ),
  );
  const originalHashes = new Map(
    provenance.assets.map((entry) => [entry.source_path, entry.sha256]),
  );
  await mount(page);
  await page.evaluate((papers) => {
    canvasModel = {
      ...canvasModel,
      nodes: papers.map(({ kind }) => ({
        key: `${kind}:paper`,
        kind,
        title: `${kind.replaceAll("_", " ")} on original paper`,
        lifecycle: kind === "outcome" ? "active" : "draft",
        ref: ["phase", "milestone", "plan_reference", "annotation"].includes(
          kind,
        )
          ? null
          : { kind, id: "paper" },
        ...(kind === "outcome"
          ? {
              summary: "A result people and agents can verify",
              canCreate: ["objective", "work_item"],
            }
          : {}),
      })),
      edges: [],
      rootKey: "outcome:paper",
    };
    canvasHandle.update(canvasModel, { selectedKey: null });
  }, papers);
  await expect(page.locator(".canvas-world .canvas-node-paper")).toHaveCount(
    papers.length,
  );
  const checked = new Set();
  async function originalImage(image, sourcePath) {
    await expect(image).toHaveAttribute(
      "src",
      `${origin}/app/postits/${sourcePath}`,
    );
    const decoded = await image.evaluate(async (element) => {
      await element.decode();
      return {
        complete: element.complete,
        width: element.naturalWidth,
        height: element.naturalHeight,
      };
    });
    expect(decoded.complete).toBe(true);
    expect(decoded.width / decoded.height).toBeCloseTo(
      sourcePath.startsWith("svg/square/") ? 1 : 1000 / 730,
      5,
    );
    if (!checked.has(sourcePath)) {
      const response = await page.request.get(
        `${origin}/app/postits/${sourcePath}`,
      );
      expect(response.status()).toBe(200);
      expect(response.headers()["content-type"]).toContain("image/svg+xml");
      expect(
        createHash("sha256")
          .update(await response.body())
          .digest("hex"),
      ).toBe(originalHashes.get(sourcePath));
      checked.add(sourcePath);
    }
  }
  for (const paper of papers) {
    const key = `${paper.kind}:paper`,
      current = card(page, key),
      image = current.locator("img.canvas-node-paper");
    await page.evaluate((key) => canvasHandle.focus(key), key);
    await expect(current).toBeVisible();
    await expect(current).toHaveAttribute("data-paper", paper.style);
    await expect(image).toHaveAttribute("alt", "");
    await expect(image).toHaveAttribute("aria-hidden", "true");
    await expect(image).toHaveAttribute("draggable", "false");
    await expect(current.getByRole("img")).toHaveCount(0);
    await expect(image).toHaveCSS("pointer-events", "none");
    await expect(image).toHaveCSS("object-fit", "contain");
    await originalImage(
      image,
      `svg/${paper.format}/${paper.style}/${paper.color}.svg`,
    );
    const geometry = await current.evaluate((element) => {
      const bounds = element.getBoundingClientRect(),
        content = element
          .querySelector(".canvas-node-content")
          .getBoundingClientRect(),
        image = element
          .querySelector(".canvas-node-paper")
          .getBoundingClientRect();
      return {
        width: bounds.width,
        height: bounds.height,
        imageWidth: image.width,
        imageHeight: image.height,
        top: (100 * (content.top - bounds.top)) / bounds.height,
        right: (100 * (bounds.right - content.right)) / bounds.width,
        bottom: (100 * (bounds.bottom - content.bottom)) / bounds.height,
        left: (100 * (content.left - bounds.left)) / bounds.width,
        backgrounds: [
          element,
          element.querySelector(".canvas-node-content"),
          element.querySelector(".canvas-node-main"),
        ].map((item) => getComputedStyle(item).backgroundColor),
      };
    });
    const camera = Number(
      await page.locator(".canvas-viewport").getAttribute("data-scale"),
    );
    expect(geometry.width / camera).toBeCloseTo(paper.width, 1);
    expect(geometry.width / geometry.height).toBeCloseTo(
      paper.format === "square" ? 1 : 1000 / 730,
      3,
    );
    expect(geometry.imageWidth).toBeCloseTo(geometry.width, 1);
    expect(geometry.imageHeight).toBeCloseTo(geometry.height, 1);
    expect(geometry.top).toBeCloseTo(paper.top, 1);
    expect(geometry.right).toBeCloseTo(14, 1);
    expect(geometry.bottom).toBeCloseTo(paper.bottom || 14, 1);
    expect(geometry.left).toBeCloseTo(14, 1);
    expect(geometry.backgrounds).toEqual([
      "rgba(0, 0, 0, 0)",
      "rgba(0, 0, 0, 0)",
      "rgba(0, 0, 0, 0)",
    ]);
    const main = current.locator(".canvas-node-main");
    await expect(main).toBeFocused();
    await main.press("Enter");
    await expect(main).toBeFocused();
    await expect(main).toHaveAttribute("aria-pressed", "true");
    await expect(current).toHaveAttribute("data-selected", "true");
    await expect(current).toHaveCSS("outline-style", "none");
    await originalImage(
      image,
      `svg/${paper.format}/${paper.style}/selected.svg`,
    );
  }
  expect(await page.evaluate(() => canvasEvents.select.length)).toBe(
    papers.length,
  );
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(page.locator(".workspace-canvas")).toHaveAttribute(
    "data-view",
    "list",
  );
  await expect(page.locator(".canvas-list img.canvas-node-paper")).toHaveCount(
    0,
  );
  await page.getByRole("button", { name: "Map view", exact: true }).click();
  await page.evaluate(() => canvasHandle.focus("work_item:paper"));
  const task = card(page, "work_item:paper");
  await expect(task.locator(".canvas-node-paper")).toBeVisible();
  await task.locator(".canvas-node-main").press("Enter");
  await originalImage(
    task.locator(".canvas-node-paper"),
    "svg/wide/curled/selected.svg",
  );
  await page.getByRole("button", { name: "List view", exact: true }).click();
  const listTask = page.locator(
    '[data-canvas-list-node="work_item:paper"] .canvas-list-main',
  );
  await expect(listTask).toBeVisible();
  await listTask.focus();
  await listTask.press("Enter");
  await expect(listTask).toHaveAttribute("aria-pressed", "true");
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
});
