import { test, expect, openOutcome } from './ux-fixture.mjs';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { execFileSync } from 'node:child_process';
import { resolve } from 'node:path';
import AxeBuilder from '@axe-core/playwright';

const board = (page) => page.locator('#workspace-kanban');

async function serveCheckoutAssets(page) {
  const root = resolve('../..');
  const sources = new Map([
    ['app.js', 'text/javascript; charset=utf-8'],
    ['en-US.js', 'text/javascript; charset=utf-8'],
    ['workspace-view.js', 'text/javascript; charset=utf-8'],
    ['workspace-view.css', 'text/css; charset=utf-8'],
  ].map(([name, mimeType]) => [name, { path: resolve(root, 'packages/wos-api/web/assets', name), mimeType }]));
  await page.route('**/app/*', async (route) => {
    const name = new URL(route.request().url()).pathname.split('/').at(-1);
    const source = sources.get(name);
    if (!source) return route.continue();
    await route.fulfill({ body: await readFile(source.path), contentType: source.mimeType });
  });
}

async function capture(page, name) {
  const directory = process.env.WOS_CRITERION_HISTORY_CAPTURE;
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
    frontendDelivery: 'Playwright routes the checkout app.js, en-US.js, workspace-view.js and workspace-view.css to Chromium; HTTP API and isolated SQLite state use bin/wos.',
    viewport: page.viewportSize(),
    browser: page.context().browser().version(),
    pngSha256: createHash('sha256').update(bytes).digest('hex'),
    state: 'A real Task delivery has been assessed against its current criterion; history shows its immutable delivery identity.',
  }, null, 2) + '\n');
}

for (const viewport of [{ width: 1440, height: 1000 }, { width: 360, height: 800 }]) {
test(`criterion history preserves the exact delivery and Evidence trace at ${viewport.width}px`, async ({ page, wos }) => {
  await page.setViewportSize(viewport);
  const { value: outcome, scope } = await wos.outcome('Traceable criterion history');
  await wos.command('add_criterion', {
    owner: { ...scope, kind: 'outcome', id: outcome.id }, expected_version: outcome.version,
    title: 'The requested result is verifiable', required: true, verification_mode: 'attestation',
  });
  await wos.command('activate_outcome', { scope, expected_version: outcome.version + 1 });
  const objective = await wos.command('create_objective', {
    scope, title: 'A reviewed implementation', priority: 'normal',
  });
  const task = await wos.command('create_work_item', {
    scope, objective_id: objective.id, title: 'Deliver a reviewed change', priority: 'normal', lifecycle: 'backlog',
  });
  const owner = { ...scope, kind: 'work_item', id: task.id };
  const criterion = await wos.command('add_criterion', {
    owner, expected_version: task.version, title: 'The change passes its focused test',
    required: true, verification_mode: 'attestation',
  });
  const currentTask = (await wos.api(`/namespaces/${wos.namespace}/outcomes/${outcome.id}/work-items/${task.id}`)).value;
  const evidence = await wos.command('register_evidence_with_links', {
    scope,
    evidence: { evidence_type: 'attestation', description: 'Focused test report for the reviewed change' },
    observation: {
      subject_ref: owner,
      criterion_id: criterion.id,
      body: 'The focused test completed and its result was recorded for independent inspection.',
    },
    links: [{
      target_ref: owner,
      target_expected_version: String(currentTask.version),
      criterion_id: criterion.id,
      criterion_revision: String(criterion.criterion_revision),
      stance: 'supports',
      rationale: 'This report is the evidence inspected for the exact criterion.',
    }],
  });
  const activated = await wos.command('activate_work_item', {
    scope, work_item_id: task.id, expected_version: task.version + 1,
  });
  const claimed = await wos.command('claim_work_item', {
    scope, work_item_id: task.id, expected_version: activated.version, ttl_seconds: 900,
  });
  const submitted = await wos.command('submit_task_for_review', {
    scope,
    work_item_id: task.id,
    expected_work_item_version: String(claimed.version),
    claim_id: claimed.current_lease.claim_id,
    fencing_token: String(claimed.current_lease.fencing_token),
    legacy_material: {
      summary: 'Focused test run completed; the result is ready for review.',
      artifacts: [], evidence_ids: [], criterion_evidence: [],
    },
  });
  const delivery = submitted.delivery;
  const assessment = await wos.command('record_criterion_assessment', {
    owner,
    criterion_id: criterion.id,
    criterion_revision: criterion.criterion_revision,
    expected_version: submitted.work_item.version,
    result: 'met',
    rationale: 'The focused test was inspected against the agreed condition.',
    evidence_ids: [evidence.evidence.id],
    task_delivery_ref: { id: delivery.id, digest: delivery.delivery_digest, round: String(delivery.round) },
  });
  expect(assessment.task_delivery_ref).toEqual({ id: delivery.id, digest: delivery.delivery_digest, round: String(delivery.round) });
  const taskHistory = (await wos.api(`/namespaces/${wos.namespace}/outcomes/${outcome.id}/work-items/${task.id}`)).value.criteria;
  expect(taskHistory.assessments.map((item) => item.id)).toContain(assessment.id);

  await serveCheckoutAssets(page);
  await wos.login(page);
  await openOutcome(page, outcome.title, { area: null });
  await page.getByRole('combobox', { name: 'Search Outcome Board', exact: true }).fill(objective.title);
  await page.getByRole('option').filter({ hasText: objective.title }).first().click();
  await page.locator('#canvas-details').getByRole('button', { name: 'Open Tasks Kanban', exact: true }).click();
  await expect(board(page).locator('.workspace-kanban-status')).not.toHaveText('Loading work…');
  await board(page).locator('.workspace-kanban-card').filter({ hasText: task.title }).click();
  const inspector = board(page).locator('.workspace-kanban-inspection');
  const criteria = inspector.locator('[data-operational-group=criteria]');
  if (!(await criteria.evaluate((node) => node.open))) await inspector.locator('summary').filter({ hasText: 'Criteria & proof' }).click();
  await criteria.getByRole('button', { name: criterion.title, exact: true }).click();
  await expect(inspector.locator('.workspace-kanban-inspection-header h3')).toHaveText(criterion.title);
  const focusedCriteria = inspector.locator('[data-operational-group=criteria]');
  if (!(await focusedCriteria.evaluate((node) => node.open))) await focusedCriteria.locator(':scope > summary').click();
  await expect(inspector.getByRole('button', { name: 'Definition and assessment history', exact: true })).toBeVisible();
  await inspector.getByRole('button', { name: 'Definition and assessment history', exact: true }).click();
  const history = page.locator('.workspace-criterion-history').filter({ hasText: 'Definition and assessment history' }).last();
  const row = history.locator('article').filter({ hasText: 'The focused test was inspected against the agreed condition.' });
  await expect(row).toContainText('Met');
  await expect(row).toContainText('Definition revision 1');
  await expect(row).toContainText(`Assessed delivery · round ${delivery.round}`);
  await expect(row).toContainText(`Delivery ID: ${delivery.id}`);
  await expect(row).toContainText('owner');
  await expect(row.getByRole('button', { name: 'Inspect linked Evidence', exact: true })).toBeVisible();
  await row.getByText('Show the full delivery digest', { exact: true }).click();
  await expect(row).toContainText(delivery.delivery_digest);
  await expect(row).toContainText('The focused test was inspected against the agreed condition.');
  await expect(page.getByText('Load full record', { exact: true })).toHaveCount(0);
  await row.scrollIntoViewIfNeeded();
  const accessibility = await new AxeBuilder({ page }).include('#workspace-kanban-operation')
    .withTags(['wcag2a', 'wcag2aa', 'wcag21aa', 'wcag22aa']).analyze();
  expect(accessibility.violations.filter((violation) => ['critical', 'serious'].includes(violation.impact)),
    JSON.stringify(accessibility.violations)).toEqual([]);
  const operation = page.locator('#workspace-kanban-operation');
  const rect = await operation.boundingBox();
  expect(rect.x).toBeGreaterThanOrEqual(0);
  expect(rect.y).toBeGreaterThanOrEqual(0);
  expect(rect.x + rect.width).toBeLessThanOrEqual(viewport.width);
  expect(rect.y + rect.height).toBeLessThanOrEqual(viewport.height);
  expect(await operation.evaluate((node) => node.scrollWidth - node.clientWidth)).toBeLessThanOrEqual(1);
  await capture(page, `criterion-history-delivery-${viewport.width}`);
  await row.getByRole('button', { name: 'Inspect linked Evidence', exact: true }).click();
  await expect(inspector).toContainText('Focused test report for the reviewed change');
});
}
