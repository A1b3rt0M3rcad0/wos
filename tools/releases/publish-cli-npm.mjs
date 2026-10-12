import { loadRelease, realAdapters } from './publish.mjs';
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
  if (!entry || entry.name !== '@woobeai/wos-cli') throw new Error('Native CLI npm artifact identity differs from approved registry scope');
  const npm = realAdapters().npm;
  const existing = await npm.integrity(entry.name, entry.version);
  if (existing && existing !== entry.integrity) throw new Error('This npm version has different bytes; immutable version cannot be overwritten');
  if (!existing) await npm.publish(entry, release.output);
  if (await npm.integrity(entry.name, entry.version) !== entry.integrity) throw new Error('Published npm integrity could not be verified');
  console.log(JSON.stringify({published:true,name:entry.name,version:entry.version,source_sha:release.manifest.commit,integrity_verified:true}));
} catch(error) { console.error('WOS CLI npm publication: ' + error.message); process.exitCode = 1; }
