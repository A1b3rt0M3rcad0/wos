import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { loadRelease, realAdapters, awaitNpmIntegrity } from './publish.mjs';
import { verifyNativePlatforms } from '../distribution/platform.mjs';

// One explicitly authorized initial npm CLI publication. This is not a
// server/GHCR release, and never finalizes a GitHub release or tag.
try {
  if (process.env.GITHUB_REPOSITORY?.toLowerCase() !== 'a1b3rt0m3rcad0/wos' || process.env.WOS_CLI_PUBLISH !== '1') throw new Error('Publication requires the authorized WOS repository workflow');
  if (!process.env.NODE_AUTH_TOKEN) throw new Error('NPM_TOKEN is unavailable to the publishing job');
  const release = await loadRelease();
  if (release.manifest.version !== '0.4.0') throw new Error('Initial npm CLI publication is limited to v0.4.0');
  await verifyNativePlatforms(release.manifest);
  const entry = release.manifest.artifacts.find(a => a.kind === 'wos-cli');
  if (!entry || entry.name !== 'wosctl') throw new Error('Native CLI npm artifact identity differs from approved package name');
  const npm = realAdapters().npm;
  const existing = await npm.integrity(entry.name, entry.version);
  if (existing && existing !== entry.integrity) throw new Error('This npm version has different bytes; immutable version cannot be overwritten');
  if (!existing) await npm.publish(entry, release.output);
  try { await awaitNpmIntegrity(npm,entry,{attempts:31,delayMs:30000}); }
  catch (error) {
    // A successful upload may be staged, rather than publicly propagated.
    // Preserve exact bytes and source; never call npm publish a second time here.
    await mkdir(release.output,{recursive:true});
    await writeFile(path.join(release.output,'npm-publication-pending.json'),JSON.stringify({published:false,name:entry.name,version:entry.version,source_sha:release.manifest.commit,integrity:entry.integrity,sha256:entry.sha256,registry_integrity_verified:false,diagnosis:'Check npm Staged Packages and approve only the original reviewed artifact. Use read-only reconciliation after approval; do not rebuild/reupload this immutable version.'},null,2)+'\n');
    throw error;
  }
  const receipt={published:true,name:entry.name,version:entry.version,source_sha:release.manifest.commit,integrity:entry.integrity,sha256:entry.sha256,integrity_verified:true};
  await mkdir(release.output,{recursive:true});
  await writeFile(path.join(release.output,'npm-publication.json'),JSON.stringify(receipt,null,2)+'\n');
  console.log(JSON.stringify(receipt));
} catch(error) { console.error('WOS CLI npm publication: ' + error.message); process.exitCode = 1; }
