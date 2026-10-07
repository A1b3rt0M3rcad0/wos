import * as fs from 'node:fs/promises';
import path from 'node:path';
import os from 'node:os';
import { createHash, randomUUID } from 'node:crypto';
import { fileURLToPath } from 'node:url';

export const packageRoot = fileURLToPath(new URL('../', import.meta.url));
export const names = ['wos-coordination', 'wos-delegation', 'wos-continuity', 'wos-review'];
export const agents = ['claude', 'codex', 'hermes', 'openclaw', 'generic', 'all'];
const marker = '.wos-skill.json';
const sha = value => createHash('sha256').update(value).digest('hex');

export function destinations({ agent = 'codex', global = false, dir, project = process.cwd(), home = os.homedir() } = {}) {
  if (!agents.includes(agent)) throw new Error(`Unknown agent: ${agent}`);
  if (dir && (global || agent === 'all')) throw new Error('--dir cannot be combined with --global or --agent all');
  if (dir) return [path.resolve(dir)];
  if (agent === 'generic') throw new Error('Generic agents require --dir for their skill discovery directory');
  const base = path.resolve(global ? home : project);
  const paths = {
    claude: path.join(base, '.claude', 'skills'),
    codex: path.join(base, '.agents', 'skills'),
    hermes: path.join(base, '.hermes', 'skills'),
    openclaw: global ? path.join(base, '.openclaw', 'skills') : path.join(base, 'skills'),
  };
  return (agent === 'all' ? ['claude', 'codex', 'hermes', 'openclaw'] : [agent]).map(a => paths[a]);
}

async function stat(p) {
  try { return await fs.lstat(p); } catch (e) { if (e.code === 'ENOENT') return null; throw e; }
}

async function safePath(p) {
  const absolute = path.resolve(p);
  for (let current = absolute; ; current = path.dirname(current)) {
    const info = await stat(current);
    if (info && (info.isSymbolicLink() || !info.isDirectory())) throw new Error(`Unsafe directory (symlink or non-directory): ${current}`);
    if (current === path.dirname(current)) break;
  }
}

async function contents(root) {
  const files = {};
  async function visit(dir) {
    for (const e of await fs.readdir(dir, { withFileTypes: true })) {
      const p = path.join(dir, e.name);
      if (e.isSymbolicLink()) throw new Error(`Symlink in skill: ${p}`);
      if (e.isDirectory()) await visit(p);
      else if (e.isFile() && path.relative(root, p) !== marker) files[path.relative(root, p).split(path.sep).join('/')] = sha(await fs.readFile(p));
      else if (!e.isFile()) throw new Error(`Unsupported skill entry: ${p}`);
    }
  }
  await visit(root);
  return Object.fromEntries(Object.entries(files).sort(([a], [b]) => a.localeCompare(b)));
}

async function inspected(root) {
  const info = await stat(root);
  if (!info) return { exists: false };
  await safePath(root);
  const m = await stat(path.join(root, marker));
  if (!m || !m.isFile() || m.isSymbolicLink()) throw new Error(`Unmanaged skill, preserving it: ${root}`);
  const data = JSON.parse(await fs.readFile(path.join(root, marker), 'utf8'));
  if (data.package !== '@a1b3rt0m3rcad0/wos-skill' || data.schema !== 1 || !names.includes(data.name)) throw new Error(`Invalid ownership manifest: ${root}`);
  const actual = await contents(root);
  if (JSON.stringify(actual) !== JSON.stringify(data.files)) throw new Error(`Locally modified skill, preserving it: ${root}`);
  return { exists: true, ...data };
}

export async function operate(command, options = {}) {
  if (!['install', 'update', 'uninstall', 'list', 'doctor'].includes(command)) throw new Error(`Unknown command: ${command}`);
  const version = JSON.parse(await fs.readFile(path.join(packageRoot, 'package.json'), 'utf8')).version;
  const bases = destinations(options);
  const plan = [];
  // Preflight every destination before the first write, including all-agent installs.
  for (const base of bases) {
    await safePath(base);
    for (const name of names) {
      const target = path.join(base, name);
      const prior = await inspected(target);
      const files = await contents(path.join(packageRoot, 'skills', name));
      const same = prior.exists && prior.version === version && JSON.stringify(prior.files) === JSON.stringify(files);
      const action = command === 'uninstall' ? (prior.exists ? 'remove' : 'absent') : ['doctor', 'list'].includes(command) ? (prior.exists ? 'installed' : 'absent') : same ? 'unchanged' : prior.exists ? 'update' : 'install';
      plan.push({ target, name, version, action, files, prior });
    }
  }
  if (!options.dryRun && !['doctor', 'list'].includes(command)) {
    for (const p of plan) {
      if (['absent', 'unchanged'].includes(p.action)) continue;
      // Recheck before each mutation; shared directories are not a locking service.
      const prior = await inspected(p.target);
      if (JSON.stringify(prior) !== JSON.stringify(p.prior)) throw new Error(`Destination changed during installation: ${p.target}`);
      await safePath(path.dirname(p.target));
      if (p.action === 'remove') { await fs.rm(p.target, { recursive: true }); continue; }
      await fs.mkdir(path.dirname(p.target), { recursive: true });
      const stage = `${p.target}.stage-${randomUUID()}`;
      const backup = `${p.target}.backup-${randomUUID()}`;
      try {
        await fs.cp(path.join(packageRoot, 'skills', p.name), stage, { recursive: true, dereference: false });
        await fs.writeFile(path.join(stage, marker), JSON.stringify({ schema: 1, package: '@a1b3rt0m3rcad0/wos-skill', name: p.name, version, files: p.files }, null, 2) + '\n');
        if (prior.exists) await fs.rename(p.target, backup);
        try { await fs.rename(stage, p.target); } catch (e) { if (prior.exists) await fs.rename(backup, p.target); throw e; }
        if (prior.exists) await fs.rm(backup, { recursive: true });
      } finally { await fs.rm(stage, { recursive: true, force: true }); }
    }
  }
  return plan.map(({ target, name, version, action }) => ({ target, name, version, action, dryRun: !!options.dryRun }));
}
