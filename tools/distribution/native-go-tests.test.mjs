import assert from 'node:assert/strict';
import * as fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import {goFailureAnnotations, nativeGoTestEvents} from './native-go-tests.mjs';

test('failure annotations preserve test names, escape control lines, and bound output', () => {
  const events = [];
  for (let n = 0; n < 30; n++) {
    events.push({Action: 'output', Package: 'test/package', Test: `TestFailure${n}`, Output: 'x'.repeat(8000) + '\n::error::100%\r\n'});
    events.push({Action: 'fail', Package: 'test/package', Test: `TestFailure${n}`});
  }
  const annotations = goFailureAnnotations(events);
  assert.equal(annotations.length, 24);
  assert.match(annotations[0], /test\/package: TestFailure0/);
  assert.match(annotations[0], /%0A::error::100%25%0D%0A/);
  assert.ok(annotations.every(line => !/[\r\n]/.test(line) && line.length < 13000));
});

test('unsuccessful native tests retain diagnostics and cannot return acceptance events', async () => {
  const root = await fs.mkdtemp(path.join(os.tmpdir(), 'wos-native-diagnostics-'));
  try {
    const events = [
      {Action: 'output', Package: 'test/package', Test: 'TestWindowsFixture', Output: 'sharing violation\n'},
      {Action: 'fail', Package: 'test/package', Test: 'TestWindowsFixture'},
    ];
    const stdout = events.map(event => JSON.stringify(event)).join('\n') + '\n';
    const annotations = [];
    await assert.rejects(nativeGoTestEvents({root, platform: 'windows', execute: (_command, args) => {
      assert.ok(args.includes('-race'));
      assert.ok(args.includes('./tests/acceptance'));
      return {status: 1, stdout, stderr: 'native failure'};
    }, annotate: line => annotations.push(line)}), /Native Go tests failed/);
    assert.equal(await fs.readFile(path.join(root, 'dist', 'native-go-tests-windows-amd64.jsonl'), 'utf8'), stdout);
    assert.equal(await fs.readFile(path.join(root, 'dist', 'native-go-tests-windows-amd64.stderr.log'), 'utf8'), 'native failure');
    assert.match(annotations[0], /TestWindowsFixture.*sharing violation/);
  } finally {
    await fs.rm(root, {recursive: true, force: true});
  }
});

test('successful native tests retain every race gate and return their actual events', async () => {
  const root = await fs.mkdtemp(path.join(os.tmpdir(), 'wos-native-success-'));
  try {
    const events = [{Action: 'pass', Package: 'test/package', Test: 'TestNative', Elapsed: 0.01}];
    const annotations = [];
    const result = await nativeGoTestEvents({root, platform: 'linux', execute: (command, args, options) => {
      assert.equal(command, 'go');
      assert.deepEqual(args, ['test', '-race', '-count=1', '-json', './packages/wos-api/cmd/wos', './packages/wos-skill', './packages/wos-cli/...', './packages/wos-sdk-go/...', './tests/acceptance']);
      assert.equal(options.cwd, root);
      return {status: 0, stdout: JSON.stringify(events[0]) + '\n', stderr: ''};
    }, annotate: line => annotations.push(line)});
    assert.deepEqual(result, events);
    assert.deepEqual(annotations, []);
  } finally {
    await fs.rm(root, {recursive: true, force: true});
  }
});
