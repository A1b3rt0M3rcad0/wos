import { readFileSync } from "node:fs";
import { test, expect } from "playwright/test";

const moduleSource = readFileSync(
  new URL("../../packages/wos-api/web/assets/workspace-inspector.js", import.meta.url),
  "utf8",
);

async function fixture(page) {
  await page.setContent(`
    <button id="canvas-node">Selected task</button>
    <aside id="canvas-details" hidden></aside>
    <dialog id="detail-dialog">
      <header class="dialog-heading"><h2 id="detail-title">Task details</h2></header>
      <div class="dialog-body" id="detail-content"><button id="edit-task">Edit task</button></div>
      <footer><button id="detail-close">Close details</button></footer>
    </dialog>
    <dialog id="command-dialog">
      <form id="command-form" class="dialog-form">
        <header class="dialog-heading"><h2 id="command-title">Edit task</h2></header>
        <div class="dialog-body" id="fields"><input id="draft" value="Original title"></div>
        <footer><button id="save-task">Save</button></footer>
      </form>
    </dialog>
    <dialog id="action-dialog">
      <header class="dialog-heading"><h2>Task actions</h2></header>
      <div class="dialog-body"><button id="choose-edit">Edit</button></div>
    </dialog>
  `);
  await page.evaluate(async (source) => {
    const { createWorkspaceInspector } = await import(
      `data:text/javascript;charset=utf-8,${encodeURIComponent(source)}`
    );
    window.closeEvents = [];
    window.nativeCloses = 0;
    window.submits = 0;
    window.originalForm = document.querySelector("#command-form");
    window.originalDetail = document.querySelector("#detail-content");
    window.originalForm.addEventListener("submit", (event) => {
      event.preventDefault();
      window.submits++;
    });
    const dialogs = {
      detail: document.querySelector("#detail-dialog"),
      command: document.querySelector("#command-dialog"),
      action: document.querySelector("#action-dialog"),
    };
    for (const dialog of Object.values(dialogs))
      dialog.addEventListener("close", () => window.nativeCloses++);
    window.inspector = createWorkspaceInspector({
      host: document.querySelector("#canvas-details"),
      dialogs,
      onClose: (surface, state) => window.closeEvents.push({ surface, ...state }),
    });
  }, moduleSource);
}

test("inline surfaces preserve exact form nodes, listeners and uncertain draft state", async ({ page }) => {
  await fixture(page);
  expect(await page.evaluate(() => window.inspector.showDetails())).toBe(false);
  await page.evaluate(() => {
    document.querySelector("#draft").value = "Unsaved authored title";
    document.querySelector("#fields").inert = true;
    document.querySelector("#save-task").disabled = true;
    window.inspector.setEnabled(true);
    window.inspector.showDetails();
    window.inspector.showForm();
  });
  await expect(page.locator("#canvas-details")).toBeVisible();
  await expect(page.locator(".canvas-inspector-form")).toBeVisible();
  await expect(page.locator(".canvas-inspector-detail")).toBeHidden();
  await expect(page.locator("#draft")).toHaveValue("Unsaved authored title");
  expect(await page.evaluate(() => ({
    sameForm: document.querySelector("#command-form") === window.originalForm,
    sameDetail: document.querySelector("#detail-content") === window.originalDetail,
    inert: document.querySelector("#fields").inert,
    disabled: document.querySelector("#save-task").disabled,
    forms: document.querySelectorAll("#command-form").length,
    rejectsReplacement: !window.inspector.showDetails() && !window.inspector.showActions(),
  }))).toEqual({ sameForm: true, sameDetail: true, inert: true, disabled: true, forms: 1, rejectsReplacement: true });
  await page.evaluate(() => {
    window.originalForm.dispatchEvent(new Event("submit", { cancelable: true }));
    window.inspector.closeAll();
    window.inspector.setEnabled(false);
  });
  await expect(page.locator("#command-dialog #command-form")).toHaveCount(1);
  await expect(page.locator("#detail-dialog #detail-content")).toHaveCount(1);
  expect(await page.evaluate(() => ({
    submits: window.submits,
    nativeCloses: window.nativeCloses,
    title: document.querySelector("#draft").value,
    inert: document.querySelector("#fields").inert,
  }))).toEqual({ submits: 1, nativeCloses: 0, title: "Unsaved authored title", inert: true });
  await page.evaluate(() => {
    window.inspector.setEnabled(true);
    window.inspector.dispose();
  });
  await expect(page.locator(".canvas-inspector-slot")).toHaveCount(0);
  await expect(page.locator("#command-dialog #command-form")).toHaveCount(1);
  expect(await page.evaluate(() => window.nativeCloses)).toBe(0);
});

test("local Escape returns from actions and forms to details, then to the canvas trigger", async ({ page }) => {
  await fixture(page);
  await page.locator("#canvas-node").focus();
  await page.evaluate(() => {
    window.inspector.setEnabled(true);
    window.inspector.showDetails();
  });
  await page.locator("#edit-task").focus();
  await page.evaluate(() => window.inspector.showActions());
  await page.keyboard.press("Escape");
  await expect(page.locator(".canvas-inspector-detail")).toBeVisible();
  await expect(page.locator("#edit-task")).toBeFocused();
  await page.evaluate(() => window.inspector.showForm());
  await page.keyboard.press("Escape");
  await expect(page.locator(".canvas-inspector-detail")).toBeVisible();
  await expect(page.locator("#edit-task")).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(page.locator("#canvas-details")).toBeHidden();
  await expect(page.locator("#canvas-node")).toBeFocused();
  expect(await page.evaluate(() => window.inspector.isVisible())).toBe(false);
  expect(await page.evaluate(() => window.closeEvents)).toEqual([
    { surface: "actions", reason: "escape", nextSurface: "detail" },
    { surface: "form", reason: "escape", nextSurface: "detail" },
    { surface: "detail", reason: "escape", nextSurface: null },
  ]);
});

test("autocomplete and nested dialogs keep their Escape without hiding the inspector", async ({ page }) => {
  await fixture(page);
  await page.evaluate(() => {
    window.inspector.setEnabled(true);
    window.inspector.showForm();
    const input = document.querySelector("#draft");
    input.setAttribute("role", "combobox");
    input.setAttribute("aria-expanded", "true");
  });
  await page.locator("#draft").focus();
  await page.keyboard.press("Escape");
  expect(await page.evaluate(() => window.inspector.isVisible("form"))).toBe(true);
  await page.evaluate(() => {
    document.querySelector("#draft").setAttribute("aria-expanded", "false");
    document.querySelector("#draft").addEventListener("keydown", (event) => {
      if (event.key === "Escape") event.preventDefault();
    });
  });
  await page.keyboard.press("Escape");
  expect(await page.evaluate(() => window.inspector.isVisible("form"))).toBe(true);
  await page.evaluate(() => {
    const nested = document.createElement("dialog");
    nested.innerHTML = '<button id="nested-close">Confirm action</button>';
    document.querySelector("#fields").append(nested);
    nested.showModal();
  });
  await page.keyboard.press("Escape");
  expect(await page.evaluate(() => window.inspector.isVisible("form"))).toBe(true);
  expect(await page.evaluate(() => window.closeEvents.length)).toBe(0);
  await page.evaluate(() => {
    document.querySelector("#draft").setAttribute("aria-expanded", "true");
    document.querySelector("#save-task").focus();
  });
  await page.keyboard.press("Escape");
  expect(await page.evaluate(() => window.inspector.isVisible("form"))).toBe(false);
  expect(await page.evaluate(() => window.closeEvents)).toEqual([
    { surface: "form", reason: "escape", nextSurface: null },
  ]);
});

test("enabling declines an open native modal without changing its top layer or firing close", async ({ page }) => {
  await fixture(page);
  await page.evaluate(() => document.querySelector("#command-dialog").showModal());
  expect(await page.evaluate(() => window.inspector.setEnabled(true))).toBe(false);
  await expect(page.locator("#command-dialog")).toBeVisible();
  await expect(page.locator("#command-dialog #command-form")).toHaveCount(1);
  expect(await page.evaluate(() => window.nativeCloses)).toBe(0);
});
