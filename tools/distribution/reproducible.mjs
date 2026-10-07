import * as fs from 'node:fs/promises';
import path from 'node:path';
import os from 'node:os';
import assert from 'node:assert/strict';
import {root,run,buildDistribution} from './build.mjs';

const temp=await fs.mkdtemp(path.join(os.tmpdir(),'wos-repro-'));
try {
  if(run('git',['status','--porcelain','--untracked-files=normal']))throw new Error('Commit source before checking cross-path reproducibility');
  const source=path.join(temp,'source');
  run('git',['clone','--quiet','--no-hardlinks',root,source]);
  const second=await buildDistribution({source,output:path.join(temp,'dist')});
  const first=JSON.parse(await fs.readFile(path.join(root,'dist/distribution.json'),'utf8'));
  assert.deepEqual(second,first);
  assert.equal(await fs.readFile(path.join(temp,'dist/SHA256SUMS'),'utf8'),await fs.readFile(path.join(root,'dist/SHA256SUMS'),'utf8'));
  console.log('PASS: native/npm assets are byte-identical across separate source paths');
}finally{await fs.rm(temp,{recursive:true,force:true});}
