import * as fs from 'node:fs/promises';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { fileURLToPath } from 'node:url';

export const root = fileURLToPath(new URL('../../', import.meta.url));
export const sha256 = data => createHash('sha256').update(data).digest('hex');
export function run(command, args, options = {}) {
  const result = spawnSync(command, args, { cwd: root, encoding: 'utf8', maxBuffer: 8 << 20, ...options });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error(`${command} failed (${result.status}): ${result.stderr || result.stdout}`);
  return result.stdout.trim();
}
export function validVersion(value) {
  if (!/^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/.test(value)) throw new Error(`Invalid stable SemVer: ${value}`);
  return value;
}

async function normalize(dir) {
  await fs.chmod(dir, 0o755);
  for (const e of await fs.readdir(dir, { withFileTypes: true })) {
    const p = path.join(dir, e.name);
    if (e.isSymbolicLink()) throw new Error(`Symlink in distribution: ${p}`);
    if (e.isDirectory()) await normalize(p);
    else await fs.chmod(p, e.name === 'wos' || e.name.endsWith('.mjs') && path.basename(dir) === 'bin' ? 0o755 : 0o644);
  }
}

export async function buildDistribution({ output = path.join(root, 'dist'), source = root } = {}) {
  const version = validVersion((await fs.readFile(path.join(source, 'VERSION'), 'utf8')).trim());
  const commit = run('git', ['rev-parse', '--verify', 'HEAD^{commit}'], { cwd: source });
  const epoch = run('git', ['show', '-s', '--format=%ct', commit], { cwd: source });
  if (!/^\d+$/.test(epoch) || !/^[a-f0-9]{40}$/.test(commit)) throw new Error('Invalid source metadata');
  const builtAt = new Date(Number(epoch) * 1000).toISOString().replace('.000Z','Z');
  const stages = {};
  await fs.mkdir(output, { recursive: true });
  for (const [kind, src] of [['wos','wos-npm'], ['wos-skill','wos-skill']]) {
    const stage = path.join(output,'npm',kind);
    await fs.rm(stage,{recursive:true,force:true}); await fs.mkdir(stage,{recursive:true});
    const pkg = JSON.parse(await fs.readFile(path.join(source,'packages',src,'package.json'),'utf8'));
    if (pkg.version !== version || pkg.scripts?.preinstall || pkg.scripts?.install || pkg.scripts?.postinstall || pkg.scripts?.prepare) throw new Error('Version mismatch or unexpected lifecycle script');
    for (const entry of pkg.files.filter(f=>!['native','release.json','LICENSE'].includes(f))) {
      await fs.cp(path.join(source,'packages',src,entry),path.join(stage,entry),{recursive:true});
    }
    // Do not advertise test scripts when test sources are intentionally excluded.
    delete pkg.scripts;
    await fs.writeFile(path.join(stage,'package.json'),JSON.stringify(pkg,null,2)+'\n');
    await fs.copyFile(path.join(source,'LICENSE'),path.join(stage,'LICENSE'));
    stages[kind] = {stage,pkg};
  }
  const native=path.join(stages.wos.stage,'native'); await fs.mkdir(native);
  const metadata='github.com/A1b3rt0M3rcad0/wos/packages/wos-api/internal/server';
  const ldflags=`-s -w -X ${metadata}.Version=${version} -X ${metadata}.Commit=${commit} -X ${metadata}.BuiltAt=${builtAt}`;
  run('go',['build','-trimpath','-buildvcs=false',`-ldflags=${ldflags}`,'-o',path.join(native,'wos'),'./packages/wos-api/cmd/wos'],{cwd:source,env:{...process.env,CGO_ENABLED:'0',GOOS:'linux',GOARCH:'amd64'}});
  const release={schema:1,version,commit,built_at:builtAt,platform:'linux',arch:'amd64',sha256:sha256(await fs.readFile(path.join(native,'wos')))};
  await fs.writeFile(path.join(stages.wos.stage,'release.json'),JSON.stringify(release,null,2)+'\n');
  const artifacts=[];
  for(const [kind,{stage,pkg}] of Object.entries(stages)) {
    await normalize(stage);
    const pack=JSON.parse(run('npm',['pack',stage,'--ignore-scripts','--json','--pack-destination',output]))[0];
    const filename=`${kind}-${version}.tgz`;
    await fs.rename(path.join(output,pack.filename),path.join(output,filename));
    artifacts.push({kind,name:pkg.name,version,filename,integrity:pack.integrity,sha256:sha256(await fs.readFile(path.join(output,filename)))});
  }
  const archiveStage=path.join(output,'archive'); await fs.rm(archiveStage,{recursive:true,force:true}); await fs.mkdir(archiveStage);
  for(const [src,name] of [[path.join(native,'wos'),'wos'],[path.join(source,'LICENSE'),'LICENSE'],[path.join(stages.wos.stage,'release.json'),'release.json'],[path.join(source,'packages/wos-npm/README.md'),'README.md']]) await fs.copyFile(src,path.join(archiveStage,name));
  await normalize(archiveStage);
  const archive=`wos_${version}_linux_amd64.tar.gz`;
  run('tar',['--sort=name',`--mtime=@${epoch}`,'--owner=0','--group=0','--numeric-owner','-czf',path.join(output,archive),'-C',archiveStage,'.']);
  artifacts.push({kind:'archive',filename:archive,sha256:sha256(await fs.readFile(path.join(output,archive)))});
  const manifest={schema:1,version,commit,built_at:builtAt,artifacts};
  await fs.writeFile(path.join(output,'distribution.json'),JSON.stringify(manifest,null,2)+'\n');
  await fs.writeFile(path.join(output,'SHA256SUMS'),[...artifacts.map(a=>`${a.sha256}  ${a.filename}`),`${sha256(await fs.readFile(path.join(output,'distribution.json')))}  distribution.json`].join('\n')+'\n');
  return manifest;
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    if(process.argv.length>2) throw new Error('Usage: node tools/distribution/build.mjs');
    console.log(JSON.stringify(await buildDistribution(),null,2));
  } catch(e) {console.error(e.message);process.exitCode=1;}
}
