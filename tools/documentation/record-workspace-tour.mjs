// Capture the real browser UI against an isolated server. No invented UI,
// overlays, remote services, or persistent credentials are used.
import assert from "node:assert/strict";
import { spawn, execFileSync } from "node:child_process";
import { createHash, randomBytes, randomUUID } from "node:crypto";
import { once } from "node:events";
import {
  readFile,
  writeFile,
  mkdir,
  mkdtemp,
  rm,
  copyFile,
  readdir,
} from "node:fs/promises";
import { createServer } from "node:net";
import { tmpdir } from "node:os";
import { dirname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { chromium } from "../../tests/web/node_modules/playwright/index.mjs";
import { expect, openOutcome } from "../../tests/web/ux-fixture.mjs";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const output = resolve(
  process.argv[2] || join(root, "docs/images/workspace-tour.gif"),
);
const binary = resolve(process.env.WOS_BINARY || join(root, "bin/wos"));
const sourceCommit = execFileSync("git", ["rev-parse", "HEAD"], {
  cwd: root,
  encoding: "utf8",
}).trim();
const directory = await mkdtemp(join(tmpdir(), "wos-workspace-tour-"));
const sha256 = (bytes) => createHash("sha256").update(bytes).digest("hex");
const socket = createServer();
socket.listen(0, "127.0.0.1");
await once(socket, "listening");
const port = socket.address().port;
await new Promise((done) => socket.close(done));
const origin = `http://127.0.0.1:${port}`;
const tenant = "0199d000-0000-7000-8000-000000000001";
const token = randomBytes(32).toString("base64url");
const environment = Object.fromEntries(
  Object.entries(process.env).filter(([key]) => !key.startsWith("WOS_")),
);
Object.assign(environment, {
  WOS_LISTEN: `127.0.0.1:${port}`,
  WOS_STORAGE_DRIVER: "sqlite",
  WOS_SQLITE_PATH: join(directory, "wos.db"),
  WOS_AUTH_MODE: "api_token",
  WOS_BOOTSTRAP_TOKEN: token,
  WOS_BOOTSTRAP_TENANT_ID: tenant,
  WOS_BOOTSTRAP_TENANT_NAME: "Refund delivery team",
  WOS_LOCAL_PRINCIPAL_ID: "illustrative-operator",
  WOS_MCP_ENABLED: "false",
  WOS_DELIVERY_WORKER_ENABLED: "false",
});
const server = spawn(binary, ["server"], {
  cwd: directory,
  env: environment,
  stdio: ["ignore", "ignore", "pipe"],
});
let serverError = "",
  browser;
server.stderr.on("data", (bytes) => {
  serverError = (serverError + bytes.toString()).slice(-4000);
});
async function api(path, command) {
  const response = await fetch(`${origin}/api/v1${path}`, {
    method: command ? "POST" : "GET",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
      "Idempotency-Key": randomUUID(),
    },
    body: command ? JSON.stringify({ command }) : undefined,
  });
  const value = await response.json();
  assert.ok(
    response.ok,
    `Public API returned ${response.status}: ${JSON.stringify(value)}`,
  );
  return value;
}
const command = async (name, input) =>
  (await api(`/commands/${name}`, input)).value;
async function run(program, args) {
  const child = spawn(program, args, {
    stdio: ["ignore", "ignore", "inherit"],
  });
  const [code] = await once(child, "exit");
  assert.equal(code, 0, `${program} exited ${code}`);
}

try {
  let ready = false;
  for (let attempt = 0; attempt < 200; attempt++) {
    if (server.exitCode !== null)
      throw Error(`Server stopped before readiness: ${serverError}`);
    try {
      if ((await fetch(`${origin}/readyz`)).ok) {
        ready = true;
        break;
      }
    } catch {}
    await new Promise((done) => setTimeout(done, 50));
  }
  assert.ok(ready, "Isolated server did not become ready");
  // Verify every embedded top-level web input against this checked-out source,
  // rather than treating a binary timestamp as source attestation.
  const webInputs = {};
  for (const name of await readdir(join(root, "packages/wos-api/web/assets"))) {
    if (!/\.(?:js|css|html|json)$/.test(name)) continue;
    const path = `packages/wos-api/web/assets/${name}`;
    const expected = await readFile(join(root, path));
    const response = await fetch(`${origin}/app/${name}`);
    assert.equal(response.status, 200, `Embedded web input ${name}`);
    assert.equal(
      sha256(Buffer.from(await response.arrayBuffer())),
      sha256(expected),
      `Embedded ${name} must match source`,
    );
    webInputs[path] = sha256(expected);
  }
  let outcome = await command("create_outcome", {
    tenant_id: tenant,
    title: "Ship verified partial refunds",
    priority: "high",
    desired_state:
      "Developers and agents can resume the same delivery, and a reviewer can verify the result against agreed criteria.",
  });
  const scope = { tenant_id: tenant, outcome_id: outcome.id };
  const ref = (kind, value) => ({ ...scope, kind, id: value.id });
  const base = `/tenants/${tenant}/outcomes/${outcome.id}`;
  const criterion = await command("add_criterion", {
    owner: ref("outcome", outcome),
    expected_version: outcome.version,
    title: "Partial refunds are correct, tested, and independently reviewed",
    description:
      "Verify partial amounts, repeated requests, and invalid requests against the implementation and test report. An authorized reviewer must assess the current criterion before explicit acceptance.",
    required: true,
    verification_mode: "evidence_review",
  });
  outcome = await command("activate_outcome", {
    scope,
    expected_version: outcome.version + 1,
  });
  const objective = await command("create_objective", {
    scope,
    title: "A refund API another participant can verify",
    priority: "high",
    required_for_outcome: false,
    description:
      "Keep the implementation, test observations, and review decision in the same Outcome.",
  });
  const tasks = [];
  for (const [title, instructions, deliverables] of [
    [
      "Implement POST /refunds",
      "Implement partial refunds in the external repository. Record the change and test results here.",
      "A reviewable change and a refund test report",
    ],
    [
      "Review refund behavior against the criteria",
      "Inspect the actual change and test report. Assess the agreed criterion using an authorized review identity.",
      "An explicit criterion assessment with rationale",
    ],
    [
      "Publish the verified delivery",
      "Wait for the review task, then publish through the external delivery system.",
      "A release reference linked to the verified delivery",
    ],
  ])
    tasks.push(
      await command("create_work_item", {
        scope,
        title,
        objective_id: objective.id,
        priority: "normal",
        lifecycle: "todo",
        description: instructions,
        execution_spec: {
          instructions: [instructions],
          constraints: [
            "Execution happens outside WOS",
            "Delivery does not automatically accept the Outcome",
          ],
          deliverables: [deliverables],
        },
      }),
    );
  for (const [source, target] of [
    [tasks[1], tasks[0]],
    [tasks[2], tasks[1]],
  ])
    await command("add_dependency", {
      scope,
      source_ref: ref("work_item", source),
      target_ref: ref("work_item", target),
      strength: "hard",
      reason: "Inspect the preceding delivery before moving to the next step.",
    });
  const claimed = await command("claim_work_item", {
    scope,
    work_item_id: tasks[0].id,
    expected_version: tasks[0].version,
    ttl_seconds: 900,
  });
  tasks[0] = await command("complete_work_item", {
    scope,
    work_item_id: claimed.id,
    expected_version: claimed.version,
    claim_id: claimed.current_lease.claim_id,
    fencing_token: claimed.current_lease.fencing_token,
    reason:
      "Illustrative delivery recorded for the product tour; independent review is still pending.",
    result_summary:
      "Implementation and test report are available for review. This example does not execute or validate a real refund service.",
  });
  const artifact = await command("register_artifact", {
    scope,
    name: "Refund implementation and test report (example)",
    artifact_type: "document",
    uri: "https://example.test/refunds/review-42",
    media_type: "text/markdown",
  });
  const evidence = await command("register_evidence", {
    scope,
    evidence_type: "inspection",
    description: "Refund test report: partial, repeated, and invalid requests",
    artifact_id: artifact.id,
    source_ref: {
      provider: "illustrative-test-report",
      id: "refunds-example-42",
    },
    captured_at: "2026-10-17T12:00:00Z",
  });
  await command("create_evidence_link", {
    scope,
    evidence_id: evidence.id,
    target_ref: ref("outcome", outcome),
    criterion_id: criterion.id,
    stance: "supports",
    rationale:
      "An example report is available to inspect. Evidence alone does not assess the criterion or accept the Outcome.",
  });
  const initial = (await api(base)).value;
  assert.equal(initial.lifecycle, "active");
  assert.ok(!initial.conclusion, "The tour must not imply acceptance");
  browser = await chromium.launch({
    ...(process.env.WOS_TEST_CHROMIUM_PATH
      ? { executablePath: process.env.WOS_TEST_CHROMIUM_PATH }
      : {}),
    headless: true,
    args: ["--no-sandbox", "--no-zygote", "--disable-dev-shm-usage"],
  });
  const page = await browser.newPage({
    viewport: { width: 1280, height: 1100 },
    deviceScaleFactor: 1,
  });
  const errors = [],
    browserMutations = [],
    failedResponses = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("request", (request) => {
    if (request.method() !== "GET" && request.url().includes("/api/v1/"))
      browserMutations.push(request.url().replace(origin, ""));
  });
  page.on("response", (response) => {
    if (response.status() >= 400 && response.url().startsWith(origin))
      failedResponses.push({
        status: response.status(),
        path: response.url().replace(origin, ""),
      });
  });
  await page.goto(`${origin}/app/`);
  await page.getByLabel("Access credential", { exact: true }).fill(token);
  await page.getByRole("button", { name: "Sign in", exact: true }).click();
  await expect(page.locator("#workspace")).toBeVisible();
  await expect(page.locator("#workspace-loading")).not.toBeVisible();
  // The initial unauthenticated session probe returns 401 before sign-in.
  // Capture and qualify authenticated product navigation only.
  failedResponses.length = 0;
  browserMutations.length = 0;
  await openOutcome(page, outcome.title, { area: null });
  const frames = [];
  async function frame(name, seconds = 4) {
    await page.evaluate(() => document.fonts.ready);
    await page
      .locator("img")
      .evaluateAll((images) =>
        Promise.all(
          images
            .filter((image) => image.checkVisibility())
            .map((image) => image.decode()),
        ),
      );
    assert.ok(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
      `No horizontal overflow in ${name}`,
    );
    await page.mouse.move(0, 0);
    const file = join(
      directory,
      `${String(frames.length).padStart(2, "0")}.png`,
    );
    await page.screenshot({ path: file, animations: "disabled" });
    frames.push({
      name,
      start_seconds: frames.reduce(
        (sum, item) => sum + item.duration_seconds,
        0,
      ),
      duration_seconds: seconds,
      file,
      sha256: sha256(await readFile(file)),
    });
  }
  await expect(page.locator("#operational-canvas")).toBeVisible();
  await page
    .locator("#operational-canvas")
    .getByRole("button", { name: "Fit view", exact: true })
    .click();
  // Use the native camera to make the primary work cards readable while the
  // Outcome title and page navigation stay visible above the working board.
  await page
    .locator("#operational-canvas")
    .getByRole("button", { name: "Zoom in", exact: true })
    .click();
  await page.evaluate(() => scrollTo(0, 0));
  const board = await page.locator(".canvas-viewport").boundingBox();
  await page.mouse.move(board.x + 24, board.y + 36);
  await page.mouse.down();
  await page.mouse.move(board.x + 24, board.y + 60, { steps: 6 });
  await page.mouse.up();
  await expect(
    page.locator(
      `[data-canvas-node="work_item:${tasks[0].id}"] .canvas-node-main`,
    ),
  ).toBeInViewport({ ratio: 1 });
  await frame("Canvas: one Outcome, related Tasks, and explicit dependencies");
  await page
    .locator("#operational-canvas")
    .getByRole("combobox", { name: "Search canvas", exact: true })
    .fill(tasks[0].title);
  await page
    .locator("#operational-canvas")
    .getByRole("option")
    .filter({ hasText: tasks[0].title })
    .first()
    .click();
  await expect(page.locator("#canvas-details #detail-title")).toHaveText(
    tasks[0].title,
  );
  await expect(page.locator("#canvas-details")).toContainText(
    "Execution happens outside WOS",
  );
  await frame("Task inspector: delivery scope and execution boundary");
  await page.locator('#area-tabs [data-area="work"]').click();
  await page.evaluate(() => scrollTo(0, 0));
  await expect(page.locator("#content .content-heading h2")).toHaveText(
    "Tasks",
  );
  await expect(page.locator("#content")).toContainText(tasks[0].title);
  await expect(page.locator("#content")).toContainText(tasks[1].title);
  await page.evaluate(() => scrollTo(0, 60));
  for (const task of tasks)
    await expect(
      page.locator(`#content [data-entity-id="${task.id}"]`),
    ).toBeInViewport({ ratio: 1 });
  await frame("Work: a completed implementation and separate review Task");
  await page.locator('#area-tabs [data-area="evidence"]').click();
  await page.evaluate(() => scrollTo(0, 0));
  await expect(page.locator("#content .content-heading h2")).toHaveText(
    "Evidence",
  );
  await expect(page.locator("#content")).toContainText(evidence.description);
  await frame("Evidence: a persisted observation, available for review");
  await page.locator(`#content [data-entity-id="${evidence.id}"]`).click();
  await expect(page.locator("#detail-dialog")).toBeVisible();
  await expect(page.locator("#detail-title")).toHaveText(evidence.description);
  await expect(page.locator("#detail-content")).toContainText(
    "illustrative-test-report",
  );
  await frame("Evidence detail: report provenance");
  await page.keyboard.press("Escape");
  await page.locator('#area-tabs [data-area="overview"]').click();
  await page.evaluate(() => scrollTo(0, 0));
  await expect(page.locator("#area-title")).toHaveText("The whole picture.");
  await expect(page.locator("#metrics")).toContainText("Criteria met");
  await expect(
    page.getByRole("progressbar", {
      name: "Criteria met: 0 of 1",
      exact: true,
    }),
  ).toBeVisible();
  await frame(
    "Overview: work progress does not automatically verify the Outcome",
  );
  await page.locator("#view-outcome").click();
  await expect(page.locator("#detail-dialog")).toBeVisible();
  await expect(page.locator("#detail-content")).toContainText(criterion.title);
  await expect(page.locator("#detail-content")).toContainText(
    "Review assessment",
  );
  await frame(
    "Success criteria: independent review and explicit acceptance remain pending",
  );
  assert.deepEqual(errors, [], "No browser script errors");
  assert.deepEqual(failedResponses, [], "No failed same-origin requests");
  assert.deepEqual(
    browserMutations,
    [],
    "Navigation must not mutate domain state",
  );
  const final = (await api(base)).value;
  assert.equal(final.lifecycle, "active");
  assert.ok(!final.conclusion);
  assert.deepEqual(
    Object.keys(final.criteria?.current_assessments || {}),
    [],
    "No criterion assessment is implied",
  );
  assert.equal(
    final.version,
    initial.version,
    "Read-only navigation must preserve the Outcome version",
  );
  await mkdir(dirname(output), { recursive: true });
  await copyFile(frames[0].file, output.replace(/\.gif$/, ".png"));
  // Hold genuine browser screenshots long enough to read. A single palette and
  // standard encoding resize are the only visual processing; no drawn frames.
  const concat = frames.flatMap((item) => [
    `file '${item.file}'`,
    `duration ${item.duration_seconds}`,
  ]);
  concat.push(`file '${frames.at(-1).file}'`);
  const list = join(directory, "frames.txt");
  await writeFile(list, concat.join("\n") + "\n");
  await run("ffmpeg", [
    "-hide_banner",
    "-loglevel",
    "error",
    "-y",
    "-f",
    "concat",
    "-safe",
    "0",
    "-i",
    list,
    "-filter_complex",
    "[0:v]fps=2,scale=1120:-1:flags=lanczos,split[a][b];[a]palettegen=stats_mode=diff[p];[b][p]paletteuse=dither=none",
    "-loop",
    "0",
    "-t",
    "28",
    output,
  ]);
  const metadata = {
    captured_at: new Date().toISOString(),
    source_commit: sourceCommit,
    method:
      "Seven genuine browser screenshots held for four seconds each. The browser navigates a real loopback WOS server and isolated SQLite database seeded through public command APIs; no invented UI, overlays, external execution, assessment, or acceptance.",
    source_inputs_sha256: webInputs,
    recorder: relative(root, fileURLToPath(import.meta.url)),
    recorder_sha256: sha256(await readFile(fileURLToPath(import.meta.url))),
    shared_navigation_helper: "tests/web/ux-fixture.mjs",
    shared_navigation_helper_sha256: sha256(
      await readFile(join(root, "tests/web/ux-fixture.mjs")),
    ),
    binary_sha256: sha256(await readFile(binary)),
    binary_source_attestation:
      "Every served top-level JS/CSS/HTML/JSON web input is byte-identical to source. The native binary hash is recorded; no stronger native binary-to-source build attestation is claimed.",
    chromium: browser.version(),
    playwright: JSON.parse(
      await readFile(
        join(root, "tests/web/node_modules/playwright/package.json"),
        "utf8",
      ),
    ).version,
    command:
      "WOS_TEST_CHROMIUM_PATH=/usr/bin/chromium node tools/documentation/record-workspace-tour.mjs",
    viewport_css: { width: 1280, height: 1100 },
    device_scale_factor: 1,
    gif: {
      file: relative(root, output),
      sha256: sha256(await readFile(output)),
      pixel_size: {
        width: (await readFile(output)).readUInt16LE(6),
        height: (await readFile(output)).readUInt16LE(8),
      },
      duration_seconds: 28,
      fps: 2,
      loop: true,
    },
    poster: {
      file: relative(root, output.replace(/\.gif$/, ".png")),
      sha256: sha256(await readFile(output.replace(/\.gif$/, ".png"))),
      pixel_size: { width: 1280, height: 1100 },
    },
    sequence: frames.map(({ file, ...item }) => item),
    assertions: [
      "Every embedded top-level web input matches source",
      "Every page and detail renders its expected persisted record",
      "No page-level horizontal overflow",
      "All visible images decode",
      "No browser script errors",
      "No failed same-origin HTTP responses during authenticated navigation",
      "No browser domain mutation requests during navigation",
      "Outcome remains Active, with unchanged version, no criterion assessment, and no Conclusion",
    ],
    limitations: [
      "Illustrative content, not a real refund implementation or test result",
      "One trusted local operator seeds unsigned example records; no signed independent-review demonstration",
      "Navigation only; WOS does not execute, test, review, publish, or bill on behalf of participants",
      "Readability demonstration in Chromium desktop; not a new cross-platform or accessibility qualification",
    ],
  };
  await writeFile(
    output.replace(/\.gif$/, ".json"),
    JSON.stringify(metadata, null, 2) + "\n",
  );
  console.log(
    JSON.stringify({
      gif: metadata.gif,
      poster: metadata.poster,
      stages: frames.length,
      assertions: "passed",
    }),
  );
} finally {
  await browser?.close();
  if (server.exitCode === null) {
    const stopped = once(server, "exit");
    server.kill("SIGTERM");
    await stopped;
  }
  await rm(directory, { recursive: true, force: true });
}
