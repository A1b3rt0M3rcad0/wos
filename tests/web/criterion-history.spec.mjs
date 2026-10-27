import { test, expect, openOutcome } from './ux-fixture.mjs';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { execFileSync } from 'node:child_process';
import { resolve } from 'node:path';
import AxeBuilder from '@axe-core/playwright';

const board = (page) => page.locator('#workspace-kanban');

async function expectNoSeriousA11yViolations(page, scope) {
  const result = await new AxeBuilder({ page }).include(scope)
    .withTags(['wcag2a', 'wcag2aa', 'wcag21aa', 'wcag22aa']).analyze();
  expect(result.violations.filter((violation) => ['critical', 'serious'].includes(violation.impact)),
    JSON.stringify(result.violations)).toEqual([]);
}

async function serveCheckoutAssets(page) {
  const root = resolve('../..');
  const sources = new Map([
    ['app.js', 'text/javascript; charset=utf-8'],
    ['actions.js', 'text/javascript; charset=utf-8'],
    ['en-US.js', 'text/javascript; charset=utf-8'],
    ['human-forms.js', 'text/javascript; charset=utf-8'],
    ['workspace-kanban.js', 'text/javascript; charset=utf-8'],
    ['workspace-view.js', 'text/javascript; charset=utf-8'],
    ['workspace-kanban.css', 'text/css; charset=utf-8'],
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
    frontendDelivery: 'Playwright routes the checkout app.js, actions.js, en-US.js, human-forms.js, workspace-kanban.js, workspace-kanban.css, workspace-view.js and workspace-view.css to Chromium; HTTP API and isolated SQLite state use bin/wos.',
    viewport: page.viewportSize(),
    browser: page.context().browser().version(),
    pngSha256: createHash('sha256').update(bytes).digest('hex'),
    state: 'The latest recorded assessment is bound to delivery round 1; round 2 is pending and the earlier assessment is explicitly marked inapplicable.',
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
  const afterAssessment = (await wos.api(`/namespaces/${wos.namespace}/outcomes/${outcome.id}/work-items/${task.id}`)).value;
  await wos.command('request_task_delivery_changes', {
    scope,
    work_item_id: task.id,
    expected_work_item_version: String(afterAssessment.version),
    delivery_id: delivery.id,
    expected_delivery_digest: delivery.delivery_digest,
    expected_delivery_round: String(delivery.round),
    reason: 'The reviewer requested a focused correction.',
    findings: ['Update the implementation before it is accepted.'],
  });
  const returnedForCorrection = (await wos.api(`/namespaces/${wos.namespace}/outcomes/${outcome.id}/work-items/${task.id}`)).value;
  const correctionLease = await wos.command('claim_work_item', {
    scope, work_item_id: task.id, expected_version: returnedForCorrection.version, ttl_seconds: 900,
  });
  const correction = await wos.command('submit_task_for_review', {
    scope,
    work_item_id: task.id,
    expected_work_item_version: String(correctionLease.version),
    claim_id: correctionLease.current_lease.claim_id,
    fencing_token: String(correctionLease.current_lease.fencing_token),
    legacy_material: {
      summary: 'A corrected result is ready for a fresh review.',
      artifacts: [], evidence_ids: [], criterion_evidence: [],
    },
  });
  expect(Number(correction.delivery.round)).toBe(Number(delivery.round) + 1);
  await expect(wos.command('accept_task_delivery', {
    scope,
    work_item_id: task.id,
    expected_work_item_version: String(correction.work_item.version),
    delivery_id: correction.delivery.id,
    expected_delivery_digest: correction.delivery.delivery_digest,
    expected_delivery_round: String(correction.delivery.round),
    reason: 'Attempt to accept the correction using its superseded assessment.',
    assessment_refs: [{
      assessment_id: assessment.id,
      criterion_id: criterion.id,
      criterion_revision: criterion.criterion_revision,
      result: 'met',
    }],
  })).rejects.toThrow();
  const pendingCorrection = (await wos.api(`/namespaces/${wos.namespace}/outcomes/${outcome.id}/work-items/${task.id}`)).value;
  expect(pendingCorrection.pending_delivery_id).toBe(correction.delivery.id);
  expect(pendingCorrection.lifecycle).toBe('in_progress');
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
  const taskCriterion = criteria.locator(`[data-criterion-id="${criterion.id}"]`);
  await expect(taskCriterion).toContainText('Assessment is not applicable to the current definition or delivery');
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
  await expect(row).not.toContainText(`Assessed delivery · round ${correction.delivery.round}`);
  await expect(row).toContainText(`Delivery ID: ${delivery.id}`);
  await expect(row).toContainText(`This assessment belongs to an earlier delivery. The current delivery is round ${correction.delivery.round} · ID ${correction.delivery.id}. Assess this delivery before accepting the Task.`);
  await expect(row).toContainText('owner');
  await expect(row.getByRole('button', { name: 'Inspect linked Evidence', exact: true })).toBeVisible();
  await row.getByText('Show the full delivery digest', { exact: true }).click();
  await expect(row).toContainText(delivery.delivery_digest);
  await expect(row).toContainText('The focused test was inspected against the agreed condition.');
  await expect(page.getByText('Load full record', { exact: true })).toHaveCount(0);
  await row.scrollIntoViewIfNeeded();
  await expectNoSeriousA11yViolations(page, '#workspace-kanban-operation');
  const operation = page.locator('#workspace-kanban-operation');
  const rect = await operation.boundingBox();
  expect(rect.x).toBeGreaterThanOrEqual(0);
  expect(rect.y).toBeGreaterThanOrEqual(0);
  expect(rect.x + rect.width).toBeLessThanOrEqual(viewport.width);
  expect(rect.y + rect.height).toBeLessThanOrEqual(viewport.height);
  expect(await operation.evaluate((node) => node.scrollWidth - node.clientWidth)).toBeLessThanOrEqual(1);
  const staleNotice = row.locator('.workspace-assessment-stale');
  await staleNotice.scrollIntoViewIfNeeded();
  await expect(staleNotice).toBeInViewport();
  await capture(page, `criterion-history-delivery-${viewport.width}`);
  await expect(row.getByRole('button', { name: 'Inspect linked Evidence', exact: true })).toBeVisible();
  await operation.getByRole('button', { name: 'Back to item', exact: true }).click();
  await expect(board(page).locator('.workspace-kanban-inspection-header h3')).toHaveText(criterion.title);
  await inspector.getByRole('button', { name: 'Back to previous item', exact: true }).click();
  await expect(inspector.locator('.workspace-kanban-inspection-header h3')).toHaveText(task.title);
  await inspector.getByRole('button', { name: 'Review acceptance criteria', exact: true }).click();
  const review = page.locator('#workspace-kanban-operation');
  await expect(review.locator('.workspace-proof-current-source')).toContainText(`Delivery round ${correction.delivery.round} · ID ${correction.delivery.id}`);
  await expect(review.locator('.workspace-proof-criterion').filter({ hasText: criterion.title })).toContainText('Assessment is not applicable to the current definition or delivery');
  await expectNoSeriousA11yViolations(page, '#workspace-kanban-operation');
  await review.locator('.workspace-proof-criterion').filter({ hasText: criterion.title }).getByRole('checkbox').click();
  await expect(page.locator('#command-form')).toBeVisible();
  await expectNoSeriousA11yViolations(page, '#workspace-kanban-operation');
  await expect(page.locator('#fields')).toContainText(`This assessment will be recorded against delivery round ${correction.delivery.round} · ${correction.delivery.id}.`);
  await page.getByLabel('Rationale', { exact: true }).fill('Reviewed the corrected delivery against the current criterion.');
  const assessmentRequest = page.waitForRequest((request) => request.method() === 'POST' && request.url().endsWith('/commands/record_criterion_assessment'));
  const assessmentResponse = page.waitForResponse((response) => response.request().method() === 'POST' && response.url().endsWith('/commands/record_criterion_assessment'));
  await page.locator('#submit-command').click();
  const [request, response] = await Promise.all([assessmentRequest, assessmentResponse]);
  const assessmentCommand = request.postDataJSON().command;
  expect(assessmentCommand.task_delivery_ref).toEqual({ id: correction.delivery.id, digest: correction.delivery.delivery_digest, round: String(correction.delivery.round) });
  expect(response.ok()).toBe(true);
  const currentAssessment = (await response.json()).value;
  expect(currentAssessment.task_delivery_ref).toEqual(assessmentCommand.task_delivery_ref);
  await expect(review.getByRole('heading', { name: 'Review proof', exact: true })).toBeVisible();
  await review.getByRole('button', { name: 'Back to item', exact: true }).click();
  await expect(inspector.locator('.workspace-kanban-inspection-header h3')).toHaveText(task.title);
  await inspector.getByRole('button', { name: 'Accept delivery', exact: true }).click();
  await expect(page.locator('#command-form')).toBeVisible();
  await expectNoSeriousA11yViolations(page, '#workspace-kanban-operation');
  await expect(page.locator('#fields')).toContainText(`Delivery round ${correction.delivery.round} · ID ${correction.delivery.id}`);
  await page.getByLabel('Reason', { exact: true }).fill('Accept the exact corrected delivery after its current criterion was assessed.');
  const acceptResponse = page.waitForResponse((candidate) => candidate.request().method() === 'POST' && candidate.url().endsWith('/commands/accept_task_delivery'));
  await page.locator('#submit-command').click();
  const accepted = await acceptResponse;
  expect(accepted.ok(), await accepted.text()).toBe(true);
  expect((await accepted.json()).value.work_item.lifecycle).toBe('done');
});
}

for (const viewport of [{ width: 1440, height: 1000 }, { width: 360, height: 800 }]) {
test(`a reviewer requests changes against the exact pending delivery at ${viewport.width}px`, async ({ page, wos }) => {
  await page.setViewportSize(viewport);
  const { value: outcome, scope } = await wos.outcome('Return the exact delivery for correction');
  await wos.command('add_criterion', {
    owner: { ...scope, kind: 'outcome', id: outcome.id }, expected_version: outcome.version,
    title: 'The returned result is explicitly reviewed', required: true, verification_mode: 'attestation',
  });
  await wos.command('activate_outcome', { scope, expected_version: outcome.version + 1 });
  const objective = await wos.command('create_objective', {
    scope, title: 'A reviewable result', priority: 'normal',
  });
  const task = await wos.command('create_work_item', {
    scope, objective_id: objective.id, title: 'Revise the submitted change', priority: 'normal', lifecycle: 'backlog',
  });
  const activated = await wos.command('activate_work_item', {
    scope, work_item_id: task.id, expected_version: task.version,
  });
  const lease = await wos.command('claim_work_item', {
    scope, work_item_id: task.id, expected_version: activated.version, ttl_seconds: 900,
  });
  const submitted = await wos.command('submit_task_for_review', {
    scope,
    work_item_id: task.id,
    expected_work_item_version: String(lease.version),
    claim_id: lease.current_lease.claim_id,
    fencing_token: String(lease.current_lease.fencing_token),
    legacy_material: { summary: 'The first result is ready for review.', artifacts: [], evidence_ids: [], criterion_evidence: [] },
  });

  await serveCheckoutAssets(page);
  await wos.login(page);
  await openOutcome(page, outcome.title, { area: null });
  await page.getByRole('combobox', { name: 'Search Outcome Board', exact: true }).fill(objective.title);
  await page.getByRole('option').filter({ hasText: objective.title }).first().click();
  await page.locator('#canvas-details').getByRole('button', { name: 'Open Tasks Kanban', exact: true }).click();
  await expect(board(page).locator('.workspace-kanban-status')).not.toHaveText('Loading work…');
  await board(page).locator('.workspace-kanban-card').filter({ hasText: task.title }).click();
  const inspector = board(page).locator('.workspace-kanban-inspection');
  const requestChanges = inspector.getByRole('button', { name: 'Request changes', exact: true });
  await expect(requestChanges).toBeVisible();
  await requestChanges.click();
  await expect(page.locator('#command-form')).toBeVisible();
  await expectNoSeriousA11yViolations(page, '#workspace-kanban-operation');
  await expect(page.locator('#fields')).toContainText(`Delivery round ${submitted.delivery.round} · ID ${submitted.delivery.id}`);
  await page.getByLabel('Reason', { exact: true }).fill('Return the exact submitted result with a concrete correction request.');
  await page.getByLabel('Requested correction item', { exact: true }).fill('Update the implementation before the reviewer checks it again.');
  const request = page.waitForRequest((candidate) => candidate.method() === 'POST' && candidate.url().endsWith('/commands/request_task_delivery_changes'));
  const response = page.waitForResponse((candidate) => candidate.request().method() === 'POST' && candidate.url().endsWith('/commands/request_task_delivery_changes'));
  await page.locator('#submit-command').click();
  const [sent, returned] = await Promise.all([request, response]);
  expect(returned.ok(), await returned.text()).toBe(true);
  const command = sent.postDataJSON().command;
  expect(command.delivery_id).toBe(submitted.delivery.id);
  expect(command.expected_delivery_digest).toBe(submitted.delivery.delivery_digest);
  expect(String(command.expected_delivery_round)).toBe(String(submitted.delivery.round));
  expect(command.findings).toEqual(['Update the implementation before the reviewer checks it again.']);
  expect((await wos.api(`/namespaces/${wos.namespace}/outcomes/${outcome.id}/work-items/${task.id}`)).value.lifecycle).toBe('todo');
});
}
