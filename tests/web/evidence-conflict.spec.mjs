import { test, expect, openOutcome } from './ux-fixture.mjs';
import { expandWorkspaceDetails } from './workspace-action-fixture.mjs';
import { createHash } from 'node:crypto';
import { execFileSync } from 'node:child_process';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import AxeBuilder from '@axe-core/playwright';

async function expectNoSeriousA11yViolations(page, scope) {
  const result = await new AxeBuilder({ page }).include(scope)
    .withTags(['wcag2a', 'wcag2aa', 'wcag21aa', 'wcag22aa']).analyze();
  expect(result.violations.filter((violation) => ['critical', 'serious'].includes(violation.impact)),
    JSON.stringify(result.violations)).toEqual([]);
}

async function serveCheckoutAssets(page) {
  const root = resolve('../..');
  const names = [
    'app.js', 'actions.js', 'en-US.js', 'human-forms.js', 'workspace-evidence-form.js',
    'workspace-view.js', 'workspace-kanban.js', 'workspace-view.css', 'workspace-kanban.css', 'workspace-dialogs.css',
  ];
  const sources = new Map(names.map((name) => [name, resolve(root, 'packages/wos-api/web/assets', name)]));
  await page.route('**/app/*', async (route) => {
    const name = new URL(route.request().url()).pathname.split('/').at(-1);
    const path = sources.get(name);
    if (!path) return route.continue();
    const css = name.endsWith('.css');
    await route.fulfill({ body: await readFile(path), contentType: css ? 'text/css; charset=utf-8' : 'text/javascript; charset=utf-8' });
  });
}

async function capture(page, name, state) {
  const directory = process.env.WOS_EVIDENCE_CONFLICT_CAPTURE;
  if (!directory) return;
  const root = resolve('../..');
  const source = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: root, encoding: 'utf8' }).trim();
  const apiServer = JSON.parse(execFileSync(resolve(root, 'bin/wos'), ['--output', 'json', 'version'], { encoding: 'utf8' })).data;
  const apiServerSha256 = createHash('sha256').update(await readFile(resolve(root, 'bin/wos'))).digest('hex');
  await mkdir(directory, { recursive: true });
  const bytes = await page.screenshot({ path: resolve(directory, `${name}.png`), animations: 'disabled' });
  await writeFile(resolve(directory, `${name}.json`), JSON.stringify({
    source,
    apiServer,
    apiServerSha256,
    frontendDelivery: 'Playwright routes the checkout app.js, actions.js, en-US.js, human-forms.js, workspace-evidence-form.js, workspace-view.js, workspace-kanban.js, workspace-view.css, workspace-kanban.css and workspace-dialogs.css to Chromium; HTTP API and isolated SQLite state use bin/wos.',
    viewport: page.viewportSize(),
    browser: page.context().browser().version(),
    pngSha256: createHash('sha256').update(bytes).digest('hex'),
    state,
  }, null, 2) + '\n');
}

for (const viewport of [{ width: 1440, height: 1000 }, { width: 768, height: 1024 }, { width: 360, height: 800 }]) {
  test(`a stale Criterion conflict safely recovers its Evidence draft at ${viewport.width}px`, async ({ page, wos }) => {
    test.setTimeout(90000);
    await page.setViewportSize(viewport);
    const { value: outcome, scope } = await wos.outcome('Recover Evidence after a revised criterion');
    const owner = { ...scope, kind: 'outcome', id: outcome.id };
    const criterion = await wos.command('add_criterion', {
      owner, expected_version: outcome.version, title: 'The handoff is reviewable',
      required: true, verification_mode: 'evidence_review',
    });
    await wos.command('activate_outcome', { scope, expected_version: outcome.version + 1 });

    await serveCheckoutAssets(page);
    await wos.login(page);
    await openOutcome(page, outcome.title, { area: null });
    await page.getByRole('combobox', { name: 'Search Outcome Board', exact: true }).fill(outcome.title);
    await page.getByRole('option').filter({ hasText: outcome.title }).first().click();
    const canvas = page.locator('#canvas-details');
    await expandWorkspaceDetails(canvas, { history: false });
    await canvas.getByRole('button', { name: criterion.title, exact: true }).click();
    await expect(canvas.locator('#detail-title')).toHaveText(criterion.title);
    await canvas.locator('[data-action-id="register_evidence_with_links"]').filter({ visible: true }).first().click();
    await expect(page.locator('#command-form')).toBeVisible();
    await page.getByLabel('Evidence source', { exact: true }).selectOption('observation');
    const observation = 'The reviewer reproduced the documented handoff on a clean workspace.';
    const rationale = 'This observation supports the criterion revision the reviewer read.';
    await page.getByLabel('Description', { exact: true }).fill(observation);
    await page.getByLabel('Why is this relevant?', { exact: true }).fill(rationale);

    const current = (await wos.api(`/namespaces/${wos.namespace}/outcomes/${outcome.id}`)).value;
    const revisedTitle = 'The handoff is reviewable by a new participant';
    const revised = await wos.command('revise_criterion', {
      owner, expected_version: current.version, criterion_id: criterion.id,
      title: revisedTitle,
      description: 'A new participant can reproduce the documented steps.',
      required: true, verification_mode: 'evidence_review',
    });

    const staleResponse = page.waitForResponse((response) => response.request().method() === 'POST'
      && response.url().endsWith('/commands/register_evidence_with_links'));
    await page.locator('#submit-command').click();
    const rejected = await staleResponse;
    expect(rejected.ok()).toBe(false);
    await expect(page.getByLabel('Description', { exact: true })).toHaveValue(observation);
    await expect(page.getByLabel('Why is this relevant?', { exact: true })).toHaveValue(rationale);
    await expectNoSeriousA11yViolations(page, '#canvas-details');
    await expect(page.locator('#command-error')).toContainText('A newer version exists');
    const unchanged = await wos.api(`/namespaces/${wos.namespace}/outcomes/${outcome.id}/state`);
    expect(unchanged.evidence).toHaveLength(0);
    expect(unchanged.evidence_links).toHaveLength(0);
    await capture(page, `evidence-conflict-stale-${viewport.width}`, 'The backend rejected the stale Criterion binding without writing Evidence; the local form values remain available.');

    // The recovery action refreshes the owner and Criterion without writing or
    // changing the Evidence draft, then lets the human inspect both definitions.
    await expect(page.locator('#conflict-review').getByRole('button', {
      name: 'Review latest Criterion and keep this draft', exact: true,
    })).toBeVisible();

    const review = page.waitForResponse((response) => response.request().method() === 'POST'
      && response.url().endsWith('/workspace-view'));
    await page.locator('#conflict-review').getByRole('button', {
      name: 'Review latest Criterion and keep this draft', exact: true,
    }).click();
    await review;
    const conflict = page.locator('#conflict-review');
    await expect(conflict).toContainText(criterion.title);
    await expect(conflict).toContainText(revisedTitle);
    const rebind = conflict.getByRole('button', {
      name: 'Use latest criterion and keep this Evidence draft', exact: true,
    });
    await expect(rebind).toBeVisible();
    await expect(page.getByLabel('Description', { exact: true })).toHaveValue(observation);
    await expect(page.getByLabel('Why is this relevant?', { exact: true })).toHaveValue(rationale);
    await expectNoSeriousA11yViolations(page, '#canvas-details');
    expect(await conflict.evaluate((node) => node.scrollWidth - node.clientWidth)).toBeLessThanOrEqual(1);
    await rebind.scrollIntoViewIfNeeded();
    const rebindBounds = await rebind.boundingBox();
    expect(rebindBounds.y).toBeGreaterThanOrEqual(0);
    expect(rebindBounds.y + rebindBounds.height).toBeLessThanOrEqual(viewport.height);
    await capture(page, `evidence-conflict-review-${viewport.width}`, 'The author compares the previous and current Criterion definitions while the unsaved Evidence form values remain in place.');

    await conflict.getByRole('button', {
      name: 'Use latest criterion and keep this Evidence draft', exact: true,
    }).click();
    const latestOwner = (await wos.api(`/namespaces/${wos.namespace}/outcomes/${outcome.id}`)).value;
    await expect(conflict).toContainText('Evidence draft retained');
    await expect(conflict).toContainText(`Criterion revision ${revised.criterion_revision}`);
    await expect(page.getByLabel('Description', { exact: true })).toHaveValue(observation);
    await expect(page.getByLabel('Why is this relevant?', { exact: true })).toHaveValue(rationale);
    await expect(page.locator('#conflict-review')).toBeVisible();
    await expectNoSeriousA11yViolations(page, '#canvas-details');
    const formBox = await page.locator('#command-form').boundingBox();
    expect(formBox.x).toBeGreaterThanOrEqual(0);
    expect(formBox.x + formBox.width).toBeLessThanOrEqual(viewport.width);
    await capture(page, `evidence-conflict-rebound-${viewport.width}`, 'The human explicitly selected the current Criterion revision; the form still awaits a separate save and server validation.');

    const savedRequest = page.waitForRequest((request) => request.method() === 'POST'
      && request.url().endsWith('/commands/register_evidence_with_links'));
    const savedResponse = page.waitForResponse((response) => response.request().method() === 'POST'
      && response.url().endsWith('/commands/register_evidence_with_links'));
    await page.locator('#submit-command').click();
    const [request, response] = await Promise.all([savedRequest, savedResponse]);
    const saved = await response.json();
    expect(response.ok(), JSON.stringify(saved)).toBe(true);
    const command = request.postDataJSON().command;
    expect(command.evidence.description).toBe(observation);
    expect(command.observation.body).toBe(observation);
    expect(command.links).toEqual([expect.objectContaining({
      target_ref: owner,
      criterion_id: criterion.id,
      criterion_revision: String(revised.criterion_revision),
      target_expected_version: String(latestOwner.version),
      stance: 'supports',
      rationale,
    })]);
    expect(saved.value.links[0].criterion_id).toBe(criterion.id);
    const state = await wos.api(`/namespaces/${wos.namespace}/outcomes/${outcome.id}/state`);
    expect(state.evidence).toHaveLength(1);
    expect(state.evidence_links).toHaveLength(1);
    expect(state.evidence_links[0].criterion_id).toBe(criterion.id);
  });
}
