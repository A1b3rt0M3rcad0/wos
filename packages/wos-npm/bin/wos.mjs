#!/usr/bin/env node
import { spawn } from 'node:child_process';
import { readFile, lstat } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { fileURLToPath } from 'node:url';
import os from 'node:os';

try {
  if (process.platform !== 'linux' || process.arch !== 'x64') throw new Error('This WOS service package supports Linux amd64 only. Portable skills are a separate package.');
  const binary = fileURLToPath(new URL('../native/wos', import.meta.url));
  const info = await lstat(binary);
  if (!info.isFile() || info.isSymbolicLink()) throw new Error('Invalid packaged WOS binary');
  const pkg = JSON.parse(await readFile(new URL('../package.json', import.meta.url), 'utf8'));
  const release = JSON.parse(await readFile(new URL('../release.json', import.meta.url), 'utf8'));
  const checksum = createHash('sha256').update(await readFile(binary)).digest('hex');
  if (release.schema !== 1 || release.version !== pkg.version || release.platform !== 'linux' || release.arch !== 'amd64' || checksum !== release.sha256) throw new Error('Packaged WOS integrity/version check failed; reinstall a verified release');
  const child = spawn(binary, process.argv.slice(2), { stdio: 'inherit', env: process.env });
  const handlers = new Map();
  for (const signal of ['SIGINT', 'SIGTERM', 'SIGHUP']) {
    const handler = () => child.kill(signal);
    handlers.set(signal, handler); process.on(signal, handler);
  }
  child.on('error', e => { console.error(`wos: ${e.message}`); process.exitCode = 1; });
  child.on('exit', (code, signal) => {
    for (const [sig, handler] of handlers) process.off(sig, handler);
    process.exitCode = code ?? (128 + (os.constants.signals[signal] ?? 1));
  });
} catch (e) { console.error(`wos: ${e.message}`); process.exitCode = 1; }
