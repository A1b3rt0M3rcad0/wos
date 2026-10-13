import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { createHash, createPublicKey, verify } from 'node:crypto';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const root = dirname(fileURLToPath(import.meta.url));
const originalRoot = '/tmp/wos-t89-managed-pilot/';
const read = name => JSON.parse(readFileSync(join(root, name), 'utf8'));
const sha256 = bytes => createHash('sha256').update(bytes).digest('hex');
const local = path => {
  assert.ok(path.startsWith(originalRoot), 'unrecognized archived source path');
  const relative = path.slice(originalRoot.length);
  assert.ok(!relative.split('/').includes('..'), 'invalid archived path');
  return join(root, relative);
};
const canonical = value => value === null || typeof value !== 'object'
  ? JSON.stringify(value)
  : Array.isArray(value)
    ? '[' + value.map(canonical).join(',') + ']'
    : '{' + Object.keys(value).sort().map(key => JSON.stringify(key) + ':' + canonical(value[key])).join(',') + '}';
const report = read('report.json');
assert.equal(report.source.npm_package, 'wosctl');
assert.equal(report.source.accepted_commit, report.source.release_manifest.commit);
assert.equal(report.source.native_sha256, report.source.release_manifest.binaries['linux-amd64'].wos.sha256);
for (const item of report.sanitized_artifact_manifest) {
  assert.equal(sha256(readFileSync(local(item.path))), item.sha256, item.path);
}
for (const attempt of [1, 2]) {
  const dir = `artifacts/attempt${attempt}`;
  const trust = read(`${dir}/public-trust.json`);
  const key = createPublicKey({
    key: Buffer.concat([Buffer.from('302a300506032b6570032100', 'hex'), Buffer.from(trust.server.public_key, 'base64')]),
    format: 'der', type: 'spki',
  });
  const results = read(`${dir}/signed-operation-results.json`).filter(row => /^ReturnSigned/.test(row.command_name));
  assert.equal(results.length, attempt === 1 ? 1 : 4);
  for (const row of results) {
    const doc = row.response_json.value.acceptance_receipt;
    const payload = Buffer.from(canonical(doc.payload));
    const type = doc.proof.payload_type;
    const pae = Buffer.concat([Buffer.from(`DSSEv1 ${Buffer.byteLength(type)} ${type} ${payload.length} `), payload]);
    assert.ok(verify(null, pae, key, Buffer.from(doc.proof.signature, 'base64')), 'receipt signature');
    assert.equal(doc.payload.accepted, true);
    assert.equal(doc.payload.local_obligation_closed, true);
    const recorded = (attempt === 1 ? report.attempt1.accepted_execution_receipt : report.attempt2.verified_signed_receipts)
      .find(item => item.contract_id === doc.payload.contract_id);
    assert.ok(recorded);
    assert.equal(recorded.acceptance_digest, 'sha256:' + sha256(payload));
    assert.equal(recorded.submission_digest, doc.payload.submission_digest);
  }
  for (const row of read(`${dir}/submissions.json`)) {
    const material = row.payload_json.material;
    assert.equal('sha256:' + sha256(canonical(material)), row.submission_digest, 'submitted material');
    const source = material.summary.match(/\/tmp\/[^;\s]+normalize-label\.mjs/)[0];
    assert.ok(material.summary.includes(sha256(readFileSync(local(source)))), 'frozen artifact checksum');
  }
}
assert.equal(report.attempt1.result, 'failed_preserved');
assert.equal(report.attempt2.final_lifecycle, 'done');
assert.equal(report.attempt2.final_review_test_results.passed, 21);
assert.equal(report.attempt2.final_review_test_results.failed, 0);
assert.deepEqual(report.attempt2.contract_directories, {executor: [], reviewer: []});
assert.equal(report.provider_token_counts, null);
assert.equal(report.provider_cost, null);
console.log('Archived pilot: checksums, five signed receipts, submitted material and frozen artifact bindings verified.');
