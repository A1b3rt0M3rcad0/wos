import { createHash } from "node:crypto";
import { execFileSync } from "node:child_process";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { resolve } from "node:path";

export const reviewViewports = [
  { width: 1440, height: 1000 },
  { width: 768, height: 1024 },
  { width: 360, height: 800 },
];

export async function captureReviewState(page, name, state) {
  const directory = process.env.WOS_HUMAN_REVIEW_CAPTURE;
  if (!directory) return;
  const root = resolve("../..");
  const source = execFileSync("git", ["rev-parse", "HEAD"], {
    cwd: root,
    encoding: "utf8",
  }).trim();
  const binary = resolve(root, "bin/wos");
  const apiServer = JSON.parse(execFileSync(binary, ["--output", "json", "version"], {
    encoding: "utf8",
  })).data;
  const apiServerSha256 = createHash("sha256").update(await readFile(binary)).digest("hex");
  await mkdir(directory, { recursive: true });
  const bytes = await page.screenshot({
    path: resolve(directory, `${name}.png`),
    animations: "disabled",
  });
  await writeFile(resolve(directory, `${name}.json`), JSON.stringify({
    source,
    apiServer,
    apiServerSha256,
    frontendDelivery: "bin/wos embeds the checked-out frontend; HTTP API and isolated SQLite state use the same binary.",
    viewport: page.viewportSize(),
    browser: page.context().browser().version(),
    pngSha256: createHash("sha256").update(bytes).digest("hex"),
    state,
  }, null, 2) + "\n");
}
