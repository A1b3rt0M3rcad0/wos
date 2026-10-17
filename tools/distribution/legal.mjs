import assert from 'node:assert/strict';
import { readFile, copyFile, mkdir } from 'node:fs/promises';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';

export const legalEntries = ['LICENSE', 'TRADEMARKS.md', 'NOTICE.md', 'THIRD_PARTY_NOTICES.md', 'LICENSES'];
export const legalFiles = [...legalEntries.filter(filename => filename !== 'LICENSES'), 'LICENSES/Apache-2.0.txt'];
export const npmLicense = 'SEE LICENSE IN LICENSE';
export const licenseReference = 'LicenseRef-WOS-Sustainable-Use-1.0';

export async function copyLegalFiles(source, destination) {
  for (const filename of legalFiles) {
    await mkdir(path.dirname(path.join(destination, filename)), {recursive: true});
    await copyFile(path.join(source, filename), path.join(destination, filename));
  }
}

export async function verifyLegalFiles(source, destination) {
  for (const filename of legalFiles) {
    const expected = await readFile(path.join(source, filename));
    assert.ok(expected.length > 0, `${filename} must not be empty`);
    assert.deepEqual(await readFile(path.join(destination, filename)), expected, `${destination}/${filename} differs from the source legal terms`);
  }
}

export async function verifyArchiveLegalFiles(source, archivePath) {
  // Read exact members without extracting potentially unrelated archive paths.
  // Python is already required for deterministic Windows ZIP construction.
  const program = `import hashlib,json,sys,tarfile,zipfile
filename=sys.argv[1]
names=sys.argv[2:]
if zipfile.is_zipfile(filename):
 with zipfile.ZipFile(filename) as archive:
  hashes={name:hashlib.sha256(archive.read(name)).hexdigest() for name in names}
else:
 with tarfile.open(filename, 'r:*') as archive:
  hashes={name:hashlib.sha256(archive.extractfile('./'+name).read()).hexdigest() for name in names}
print(json.dumps(hashes))`;
  const result = spawnSync('python3', ['-c', program, archivePath, ...legalFiles], {encoding: 'utf8', maxBuffer: 1 << 20});
  if (result.error) throw result.error;
  assert.equal(result.status, 0, `${archivePath} has missing or unreadable legal files: ${result.stderr}`);
  const actual = JSON.parse(result.stdout);
  for (const filename of legalFiles) {
    const expected = createHash('sha256').update(await readFile(path.join(source, filename))).digest('hex');
    assert.equal(actual[filename], expected, `${archivePath}/${filename} differs from the source legal terms`);
  }
}
