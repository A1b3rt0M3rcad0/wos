import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { root, validVersion, run } from './build.mjs';
import path from 'node:path';

test('distribution rejects prerelease/untrusted metadata and failed commands', () => {
  for(const value of ['1.2.3','0.1.0','10.20.30']) assert.equal(validVersion(value),value);
  for(const value of ['v1.2.3','1.2.3; echo unsafe','01.2.3','1.2','1.2.3-rc.1']) assert.throws(()=>validVersion(value),/Invalid stable SemVer/);
  assert.throws(()=>run(process.execPath,['-e','process.exit(7)']),/failed \(7\)/);
});

test('both package identities/versions are synchronized without install hooks or dependencies', async () => {
  const version=(await readFile(path.join(root,'VERSION'),'utf8')).trim();
  for(const folder of ['wos-npm','wos-skill']) {
    const pkg=JSON.parse(await readFile(path.join(root,'packages',folder,'package.json'),'utf8'));
    assert.equal(pkg.version,version); assert.equal(pkg.license,'Apache-2.0');
    assert.ok(pkg.name.startsWith('@a1b3rt0m3rcad0/'));
    assert.equal(pkg.dependencies,undefined);
    for(const hook of ['preinstall','install','postinstall','prepare']) assert.equal(pkg.scripts?.[hook],undefined);
  }
});
