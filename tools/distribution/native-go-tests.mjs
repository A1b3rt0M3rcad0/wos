import * as fs from 'node:fs/promises';
import path from 'node:path';
import {spawnSync} from 'node:child_process';

const maxAnnotations = 24;
const maxMessage = 4096;

function escapeAnnotation(value) {
  return value.replaceAll('%', '%25').replaceAll('\r', '%0D').replaceAll('\n', '%0A');
}

// Preserve the failing test's own output instead of the last package summary.
// Bound annotations so a large failure still leaves a readable Actions check.
export function goFailureAnnotations(events, fallback = '') {
  const output = new Map();
  const failures = [];
  for (const event of events) {
    const key = `${event.Package || ''}\0${event.Test || ''}`;
    if (typeof event.Output === 'string') {
      output.set(key, ((output.get(key) || '') + event.Output).slice(-maxMessage));
    }
    if (event.Action === 'fail' && failures.length < maxAnnotations) {
      const label = [event.Package, event.Test].filter(Boolean).join(': ').slice(0, 1024);
      const detail = output.get(key) || fallback || 'See the native Go test artifact for details.';
      failures.push(`::error title=Native Go test failure::${escapeAnnotation(`${label}\n${detail.slice(-(maxMessage - label.length - 1))}`)}`);
    }
  }
  if (failures.length === 0) {
    failures.push(`::error title=Native Go test failure::${escapeAnnotation((fallback || 'Native Go tests failed; see the captured JSON and stderr artifacts.').slice(-maxMessage))}`);
  }
  return failures;
}

export async function nativeGoTestEvents({root, platform, execute = spawnSync, annotate = line => console.error(line)}) {
  const directory = path.join(root, 'dist');
  await fs.mkdir(directory, {recursive: true});
  const stem = path.join(directory, `native-go-tests-${platform}-amd64`);
  const result = execute('go', ['test', '-race', '-count=1', '-json', './packages/wos-api/cmd/wos', './packages/wos-skill', './packages/wos-cli/...', './packages/wos-sdk-go/...', './tests/acceptance'], {cwd: root, encoding: 'utf8', maxBuffer: 8 << 20});
  const stdout = result.stdout || '';
  const stderr = [result.stderr, result.error?.message].filter(Boolean).join('\n');
  // Persist before interpreting status: an unsuccessful native run must remain
  // diagnosable even when Actions' separate full-log host is unavailable.
  await fs.writeFile(stem + '.jsonl', stdout);
  await fs.writeFile(stem + '.stderr.log', stderr);
  const events = [];
  let malformed = false;
  for (const line of stdout.split('\n').filter(Boolean)) {
    try { events.push(JSON.parse(line)); }
    catch { malformed = true; }
  }
  if (result.error || result.status !== 0 || malformed) {
    const fallback = stderr || (malformed ? 'Native Go tests emitted malformed JSON.' : `go test failed (${result.status ?? result.signal ?? 'unknown'}).`);
    for (const annotation of goFailureAnnotations(events, fallback)) annotate(annotation);
    throw new Error(`Native Go tests failed; captured output: ${stem}.jsonl and ${stem}.stderr.log`);
  }
  return events;
}
