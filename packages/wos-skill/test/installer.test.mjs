import test from 'node:test';
import assert from 'node:assert/strict';
import * as fs from 'node:fs/promises';
import path from 'node:path';
import os from 'node:os';
import { destinations, names, operate, packageRoot } from '../lib/installer.mjs';

async function fixture(t) {
  const root = await fs.mkdtemp(path.join(os.tmpdir(), 'wos-skill-'));
  t.after(() => fs.rm(root, { recursive: true, force: true }));
  return root;
}

test('project/global destinations match four documented discovery roots and generic explicit path', () => {
  assert.deepEqual(destinations({agent:'all',project:'/project'}), ['/project/.claude/skills','/project/.agents/skills','/project/.hermes/skills','/project/skills'].map(p=>path.resolve(p)));
  assert.deepEqual(destinations({agent:'all',global:true,home:'/user'}), ['/user/.claude/skills','/user/.agents/skills','/user/.hermes/skills','/user/.openclaw/skills'].map(p=>path.resolve(p)));
  assert.deepEqual(destinations({agent:'generic',dir:'/custom'}), [path.resolve('/custom')]);
  assert.throws(() => destinations({agent:'generic'}), /require --dir/);
  assert.throws(() => destinations({agent:'unknown'}), /Unknown agent/);
  assert.throws(() => destinations({agent:'all',dir:'/custom'}), /cannot be combined/);
});

test('all-agent installation is repeatable, dry-run makes no files, updates managed content and removes it', async t => {
  const root = await fixture(t); const opts={agent:'all',project:root};
  assert.equal((await operate('install',{...opts,dryRun:true})).length,names.length*4);
  assert.deepEqual(await fs.readdir(root),[]);
  assert.ok((await operate('install',opts)).every(x=>x.action==='install'));
  assert.ok((await operate('doctor',opts)).every(x=>x.action==='installed'));
  assert.ok((await operate('install',opts)).every(x=>x.action==='unchanged'));
  const marker=path.join(root,'.agents/skills/wos-coordination/.wos-skill.json');
  const data=JSON.parse(await fs.readFile(marker,'utf8')); data.version='0.0.1'; await fs.writeFile(marker,JSON.stringify(data));
  assert.equal((await operate('update',opts)).filter(x=>x.action==='update').length,1);
  assert.ok((await operate('uninstall',opts)).every(x=>x.action==='remove'));
  assert.ok((await operate('uninstall',opts)).every(x=>x.action==='absent'));
});

test('user edits, unmanaged bundles, extra files and symlinks are preserved before any write', async t => {
  const root=await fixture(t); const opts={agent:'codex',project:root};
  await operate('install',opts);
  const target=path.join(root,'.agents/skills/wos-review');
  await fs.appendFile(path.join(target,'SKILL.md'),'\nUser instruction\n');
  await assert.rejects(operate('uninstall',opts),/Locally modified/);
  await assert.rejects(operate('update',opts),/Locally modified/);
  assert.match(await fs.readFile(path.join(target,'SKILL.md'),'utf8'),/User instruction/);
  const other=await fixture(t); await fs.mkdir(path.join(other,'skills/wos-review'),{recursive:true});
  await assert.rejects(operate('install',{agent:'all',project:other}),/Unmanaged/);
  assert.deepEqual(await fs.readdir(other),['skills']);
  const linked=await fixture(t); await fs.symlink(root,path.join(linked,'.agents'),'dir');
  await assert.rejects(operate('install',{project:linked}),/Unsafe directory/);
  const extras=await fixture(t); await operate('install',{project:extras});
  await fs.writeFile(path.join(extras,'.agents/skills/wos-review/personal.txt'),'keep');
  await assert.rejects(operate('uninstall',{project:extras}),/Locally modified/);
});

test('portable skill frontmatter, bounded metadata and progressive references are bundled', async () => {
  for(const name of names) {
    const text=await fs.readFile(path.join(packageRoot,'skills',name,'SKILL.md'),'utf8');
    assert.match(text,new RegExp(`^---\\nname: ${name}\\ndescription: .+\\n---`));
    assert.ok(text.length<6500,'main instructions remain bounded');
    for(const match of text.matchAll(/\]\((references\/[^)]+)\)/g)) await fs.access(path.join(packageRoot,'skills',name,match[1]));
  }
});

test('skill tool names exist in the real MCP catalog', async () => {
  const repository=path.resolve(packageRoot,'../..');
  const catalog=(await fs.readFile(path.join(repository,'packages/wos-api/mcp/commands_generated.go'),'utf8'))+(await fs.readFile(path.join(repository,'packages/wos-api/mcp/queries.go'),'utf8'));
  const known=new Set([...catalog.matchAll(/"(wos_[a-z_]+)"/g)].map(m=>m[1]));
  for(const name of names) {
    const text=await fs.readFile(path.join(packageRoot,'skills',name,'SKILL.md'),'utf8');
    for(const m of text.matchAll(/\b(wos_[a-z_]+)\b/g))assert.ok(known.has(m[1]),`Unknown skill tool ${m[1]}`);
  }
});
