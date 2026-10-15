import {spawn} from 'node:child_process';
import {createConnection,createServer} from 'node:net';
import {once} from 'node:events';
import {mkdtemp,rm,readFile} from 'node:fs/promises';
import {join} from 'node:path';
import {tmpdir} from 'node:os';
import {createHash} from 'node:crypto';
const binary=process.argv[2];
const dir=await mkdtemp(join(tmpdir(),'wos-slow-body-'));
const reservation=createServer();reservation.listen(0,'127.0.0.1');await once(reservation,'listening');const port=reservation.address().port;await new Promise(done=>reservation.close(done));
const child=spawn(binary,['server'],{env:{...process.env,WOS_LISTEN:`127.0.0.1:${port}`,WOS_AUTH_MODE:'local',WOS_SERVER_SIGNING_SEED_ENV:'',WOS_SQLITE_PATH:join(dir,'wos.db')},stdio:['ignore','ignore','pipe']});
let log='';child.stderr.on('data',chunk=>{log=(log+chunk.toString()).slice(-1000)});
try{
 let ready=false;for(let i=0;i<100;i++){try{ready=(await fetch(`http://127.0.0.1:${port}/readyz`)).ok}catch{}if(ready)break;await new Promise(done=>setTimeout(done,25))}if(!ready)throw Error('isolated server did not start: '+log);
 const socket=createConnection({host:'127.0.0.1',port});await once(socket,'connect');
 const started=performance.now();let responded=false,closed=false,status='';
 socket.on('data',chunk=>{responded=true;status+=chunk.toString().slice(0,100)});socket.on('close',()=>closed=true);socket.on('error',()=>{});
 socket.write(`POST /api/v1/namespaces/0199d000-0000-7000-8000-000000000001/outcomes HTTP/1.1\r\nHost: 127.0.0.1:${port}\r\nContent-Type: application/json\r\nIdempotency-Key: incomplete-body-does-not-parse\r\nContent-Length: 128\r\n\r\n{"title":`);
 await new Promise(done=>setTimeout(done,17000));
 console.log(JSON.stringify({binary_sha256:createHash('sha256').update(await readFile(binary)).digest('hex'),observation_ms:Math.round(performance.now()-started),configured_request_timeout_ms:15000,partial_json_body:true,responded,closed,status_line:status.split('\r\n')[0]||null}));socket.destroy();
}finally{if(child.exitCode===null){const exit=once(child,'exit');child.kill('SIGTERM');await exit}await rm(dir,{recursive:true,force:true})}
