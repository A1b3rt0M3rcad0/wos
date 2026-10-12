import { spawn } from 'node:child_process';
import { lstat, readFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import os from 'node:os';

const root = fileURLToPath(new URL('../', import.meta.url));
export async function resolveNative(command, { platform = process.platform, arch = process.arch } = {}) {
  if (command !== 'wos' && command !== 'wosctl') throw new Error('Unsupported WOS command');
  if (arch !== 'x64' || !['linux','win32'].includes(platform)) throw new Error('WOS native CLI supports Linux x64 and Windows x64');
  const target = platform === 'win32' ? 'windows-amd64' : 'linux-amd64';
  const executable = command + (platform === 'win32' ? '.exe' : '');
  const native = path.join(root, 'native', target, executable);
  const info = await lstat(native);
  if (!info.isFile() || info.isSymbolicLink()) throw new Error('Invalid native WOS executable');
  const pkg = JSON.parse(await readFile(path.join(root, 'package.json'),'utf8'));
  const release = JSON.parse(await readFile(path.join(root, 'release.json'),'utf8'));
  const expected = release.binaries?.[target]?.[command];
  if (release.schema !== 1 || release.version !== pkg.version || !/^[a-f0-9]{40}$/.test(release.commit) || !/^[a-f0-9]{64}$/.test(expected?.sha256 ?? '')) throw new Error('WOS CLI release metadata mismatch');
  const hash = createHash('sha256').update(await readFile(native)).digest('hex');
  if (hash !== expected.sha256) throw new Error('WOS CLI binary checksum mismatch; reinstall from a verified release');
  return native;
}

export async function launch(command, args = process.argv.slice(2)) {
  try {
    const native = await resolveNative(command);
    const child = spawn(native, args, { stdio:'inherit', env:process.env, windowsHide:true });
    const handlers = new Map();
    for (const signal of ['SIGINT','SIGTERM','SIGHUP']) {
      const handler = () => { if (child.exitCode === null && child.signalCode === null) child.kill(signal); };
      handlers.set(signal, handler);
      process.on(signal, handler);
    }
    const cleanup = () => { for (const [signal, fn] of handlers) process.off(signal, fn); };
    child.on('error', error => { cleanup(); console.error('WOS CLI: ' + error.message); process.exitCode = 1; });
    child.on('exit', (code, signal) => { cleanup(); process.exitCode = code ?? (128 + (os.constants.signals[signal] ?? 1)); });
  } catch (error) { console.error('WOS CLI: ' + error.message); process.exitCode = 1; }
}
