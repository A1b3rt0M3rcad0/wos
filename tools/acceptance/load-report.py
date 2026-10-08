"""Turn successful Go load fixture logs into scoped, reproducible evidence."""
import datetime,json,re,subprocess,sys
from pathlib import Path
if len(sys.argv)!=4:raise SystemExit('usage: load-report.py LOG SOURCE_COMMIT OUTPUT')
log,commit,output=sys.argv[1:]
if not re.fullmatch('[a-f0-9]{40}',commit):raise SystemExit('full source commit required')
rows=[];pending=[];text=Path(log).read_text()
if re.search(r'^FAIL|--- FAIL:',text,re.M):raise SystemExit('failed workload cannot be accepted as evidence')
for line in text.splitlines():
 if 'LOAD {' in line:pending.append(json.loads(line.split('LOAD ',1)[1]))
 if line.startswith('ok ') or line.startswith('ok\t'):
  if '/storage/sqlite' in line or '/storage/postgres' in line:
   backend='sqlite' if '/storage/sqlite' in line else 'postgres'
   rows.extend(dict(backend=backend,**item) for item in pending);pending=[]
expected={(b,n,h,c) for b in ('sqlite','postgres') for n in (100,1000,10000) for h in (0,10,100) for c in (2,10,50)}
if {(r['backend'],r['work_items'],r['checkpoint_sample'],r['consumers']) for r in rows}!=expected or len(rows)!=54:raise SystemExit('incomplete matrix')
report={'schema':1,'source_commit':commit,'measured_at':datetime.datetime.now(datetime.timezone.utc).isoformat(),'go':subprocess.check_output(['go','version'],text=True).strip(),'scope':'Local storage/application fixture; one shared Outcome; two renewal/query rounds per consumer; checkpoint density on one active sample task. Not end-to-end network throughput, production SLA, native Windows or Woobe certification.','retry_policy':'Only transaction_conflict: same mutation payload and idempotency key, at most 100 retries, increasing bounded backoff plus jitter. Read retries repeat the same query.','limitations':['No complete 1,000,000 checkpoint dataset','PostgreSQL statement count unavailable; SQLite native trace records actual statements','One sandbox measurement; no distributed infrastructure or Windows run'],'rows':rows}
Path(output).write_text(json.dumps(report,indent=2)+'\n')
print(f'{len(rows)} scoped measurements saved')
