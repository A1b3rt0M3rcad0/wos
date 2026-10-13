import { createHash } from 'node:crypto';
import { readFileSync, writeFileSync } from 'node:fs';
import assert from 'node:assert/strict';
const artifactPath = '/tmp/wos-t89-managed-pilot/artifacts/round1/normalize-label.mjs';
const expectedSha256 = 'f1ccc45307af2b35bcc3e959c1391b6fcbca31fe955ce4733ad883f3f5e1a2f1';
const actualSha256 = createHash('sha256').update(readFileSync(artifactPath)).digest('hex');
assert.equal(actualSha256, expectedSha256, 'Frozen artifact must match issued submission');
const { normalizeLabel } = await import(artifactPath);
const results = [];
function test(name, fn) {
  try { fn(); results.push({ name, passed: true }); }
  catch (error) { results.push({ name, passed: false, error: error.message }); }
}
test('trim leading/trailing whitespace', () => assert.equal(normalizeLabel(' \t label \n '), 'label'));
test('collapse internal spaces', () => assert.equal(normalizeLabel(' alpha   beta '), 'alpha beta'));
test('collapse mixed internal whitespace', () => assert.equal(normalizeLabel('alpha\t\n  beta'), 'alpha beta'));
test('whitespace-only becomes empty', () => assert.equal(normalizeLabel(' \t\n\r '), ''));
for (const [name, value] of [['null', null], ['undefined', undefined], ['number', 42], ['boolean', true], ['object', {}], ['array', []]]) {
  test(`nonstring ${name} throws TypeError`, () => assert.throws(() => normalizeLabel(value), TypeError));
}
const report = { artifactPath, expectedSha256, actualSha256, checksumMatches: actualSha256 === expectedSha256, results, passed: results.filter(r => r.passed).length, failed: results.filter(r => !r.passed).length };
writeFileSync('/tmp/wos-t89-managed-pilot/artifacts/review1/test-results.json', JSON.stringify(report, null, 2) + '\n');
console.log(JSON.stringify(report, null, 2));
process.exitCode = report.failed ? 1 : 0;
