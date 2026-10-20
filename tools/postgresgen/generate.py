"""Generate the PostgreSQL SQL variant of the shared repository implementation.

Domain and Application rules remain shared. Differences are explicit SQL placeholders,
insert-if-absent syntax, search functions and the Outcome row guard. Generated output
is reviewed and committed, so PostgreSQL needs no generator at runtime.
"""
from pathlib import Path
import re
import subprocess
source=Path('packages/wos-core/storage/sqlite')
target=Path('packages/wos-core/storage/postgres')
for name in ['workspace_placement.go','entity_notes.go','workspace_timeline.go','workspace_read.go','work_activity.go','graph_read.go','focal_current.go','focal_records.go','continuity_projection.go','continuity_read.go','delivery_health.go','signed_return_resolutions.go','signed_operations.go','signed_participants.go','server_identity.go','signed_contracts.go','signed_work_contracts.go','signing_identity.go','work_protocol.go','work_contracts.go','repositories.go','owned_scope.go','nested.go','relations.go','issues_blockers.go','documentary.go','roadmaps.go','events.go','security.go','security_admin.go','context_index.go','access.go','queries.go','references.go','external_context.go','integration.go','codec.go','migrations.go']:
 s=(source/name).read_text().replace('package sqlite','package postgres',1).replace('sqlite migration','postgres migration')
 def transform_sql(m):
  q=m[1]
  if 'SELECT ' not in q and 'INSERT ' not in q and 'UPDATE ' not in q and 'DELETE ' not in q and 'CREATE ' not in q and 'FROM ' not in q:return m[0]
  if 'INSERT OR IGNORE' in q:q=q.replace('INSERT OR IGNORE','INSERT')+' ON CONFLICT DO NOTHING'
  q=q.replace('instr(lower(title),lower(?))>0','strpos(lower(title),lower(?))>0').replace('instr(lower(description),lower(?))>0','strpos(lower(description),lower(?))>0')
  q=q.replace("json_extract(state_json,'$.signed_binding.separation_group')","state_json::jsonb #>> '{signed_binding,separation_group}'")
  if 'SELECT version FROM namespaces WHERE' in q:q+=' FOR UPDATE'
  if 'SELECT lifecycle FROM namespaces WHERE' in q:q+=' FOR SHARE'
  if name=='server_identity.go' and 'SELECT state_json FROM server_protocol_identity WHERE singleton=1' in q:
   q+= ' FOR UPDATE' if '/* issuer recovery */' in q else ' FOR SHARE'
  if name=='access.go'  and ' FROM credentials WHERE digest=' in q:q+=' FOR SHARE'
  if re.search(r'SELECT state_revision\s+FROM outcome_coordination',q):q+=' FOR UPDATE'
  counter=0
  def placeholder(_):
   nonlocal counter
   counter+=1;return '$'+str(counter)
  q=re.sub(r'\?',placeholder,q)
  return '`'+q+'`'
 s=re.sub(r'`([^`]*)`',transform_sql,s,flags=re.S)
 if name=='integration.go':
  # The integration queue has row-local invariants: locked SKIP LOCKED claims
  # and exact lease/fence/deadline ACK predicates. SSI makes independent queue
  # lanes abort after a receiver side effect. Domain UOWs stay Serializable.
  start=s.index('func (s *Store) ClaimDelivery(')
  s=s[:start]+s[start:].replace('sql.LevelSerializable','sql.LevelReadCommitted')
  s=s.replace('ORDER BY next_attempt,id LIMIT 1`','ORDER BY next_attempt,id LIMIT 1 FOR UPDATE SKIP LOCKED`')
 # Query fragments with placeholders need runtime rebinding after composition.
 if name in ['workspace_placement.go','entity_notes.go','workspace_timeline.go','workspace_read.go','work_activity.go','queries.go','references.go','delivery_health.go','continuity_projection.go','focal_current.go','focal_records.go','graph_read.go']:
  original=(source/name).read_text().replace('package sqlite','package postgres',1)
  s=original.replace('instr(lower(t."+title+"),lower(?))>0',"strpos(translate(t.\"+title+\",'ABCDEFGHIJKLMNOPQRSTUVWXYZ','abcdefghijklmnopqrstuvwxyz'),translate(?,'ABCDEFGHIJKLMNOPQRSTUVWXYZ','abcdefghijklmnopqrstuvwxyz'))>0").replace('instr(lower(title),lower(?))>0','strpos(lower(title),lower(?))>0').replace('instr(lower(description),lower(?))>0','strpos(lower(description),lower(?))>0').replace('QueryContext(ctx, q, args...)','QueryContext(ctx, rebind(q), args...)').replace('QueryContext(ctx,q,args...)','QueryContext(ctx,rebind(q),args...)').replace('QueryRowContext(ctx, q, args...)','QueryRowContext(ctx, rebind(q), args...)')
 if name in ['work_activity.go','entity_notes.go']:
  s=s.replace('ExecContext(ctx, query,','ExecContext(ctx, rebind(query),').replace('ExecContext(ctx, q,','ExecContext(ctx, rebind(q),').replace('ExecContext(ctx,q,','ExecContext(ctx,rebind(q),').replace('QueryRowContext(ctx,q,','QueryRowContext(ctx,rebind(q),').replace('QueryRowContext(ctx, q,','QueryRowContext(ctx, rebind(q),')
 if name=='workspace_read.go':
  # PostgreSQL index expressions use immutable calendar arithmetic rather than
  # timezone-dependent text-to-timestamptz casts, preserving nanoseconds.
  for constant,fragment in [('workspaceExecutionSeconds','workspace_execution_seconds.sql'),('workspaceExecutionNanos','workspace_execution_nanos.sql')]:
   expression=Path('tools/postgresgen',fragment).read_text().strip()
   s,count=re.subn(r'(const '+constant+r' = )`[^`]+`',lambda m:m[1]+'`'+expression+'`',s,count=1)
   assert count==1,constant
 if name=='workspace_timeline.go':
  s=s.replace('QueryRowContext(ctx, q,','QueryRowContext(ctx, rebind(q),').replace('QueryRowContext(ctx,q,','QueryRowContext(ctx,rebind(q),')
  s=s.replace("json_extract(e.payload_json,'$.criterion_id')", "(e.payload_json::jsonb ->> 'criterion_id')").replace("json_extract(e.payload_json,'$.CriterionID')", "(e.payload_json::jsonb ->> 'CriterionID')")
  s=s.replace("json_each(e.payload_json,'$.criterion_refs') c WHERE json_extract(c.value,'$.criterion_id')", "jsonb_array_elements(COALESCE(e.payload_json::jsonb -> 'criterion_refs','[]'::jsonb)) c WHERE (c.value ->> 'criterion_id')")
 if name=='continuity_projection.go':
  # SQLite uses indexed point joins; PostgreSQL materializes scoped sets to
  # avoid a correlated history/evidence subplan and pathological JIT cost.
  start=s.index('func (r continuityReadRepository) projectionProofCounts')
  prefix=s[:start]; tail=s[start:]
  tail=re.sub(r'query := `[^`]+`',lambda _: 'query := `'+Path('tools/postgresgen/projection_proof_counts.sql').read_text().strip()+'`',tail,count=1)
  tail=tail.replace('scope.NamespaceID.String(), scope.OutcomeID.String()).Scan(&required, &met, &waived)','scope.NamespaceID.String(), scope.OutcomeID.String(), scope.NamespaceID.String(), scope.OutcomeID.String(), scope.NamespaceID.String(), scope.OutcomeID.String()).Scan(&required, &met, &waived)')
  s=prefix+tail
 if name=='continuity_read.go':
  s=s.replace('QueryContext(ctx, q, args...)','QueryContext(ctx, rebind(q), args...)').replace('QueryContext(ctx,q,args...)','QueryContext(ctx,rebind(q),args...)').replace('QueryRowContext(ctx, q, args...)','QueryRowContext(ctx, rebind(q), args...)').replace('QueryRowContext(ctx,q,args...)','QueryRowContext(ctx,rebind(q),args...)')
 if name=='references.go':
  s=s.replace('instr(lower(t." + title + "),lower(?))>0',"strpos(translate(t.\" + title + \",'ABCDEFGHIJKLMNOPQRSTUVWXYZ','abcdefghijklmnopqrstuvwxyz'),translate(?,'ABCDEFGHIJKLMNOPQRSTUVWXYZ','abcdefghijklmnopqrstuvwxyz'))>0")
 # SQL queries composed by fmt.Sprintf, and single-line quoted migration statements.
 s=s.replace('"SELECT id FROM " + table + " WHERE namespace_id = ? AND outcome_id = ? ORDER BY id"','"SELECT id FROM " + table + " WHERE namespace_id = $1 AND outcome_id = $2 ORDER BY id"')
 s=s.replace('encodeTime(c.ExpiresAt), c.Revoked)','encodeTime(c.ExpiresAt), boolInt(c.Revoked))')
 s=s.replace('"SELECT version FROM %s WHERE namespace_id = ? AND outcome_id = ? AND id = ?"','"SELECT version FROM %s WHERE namespace_id = $1 AND outcome_id = $2 AND id = $3"')
 s=s.replace('"SELECT checksum FROM schema_migrations WHERE version = ?"','"SELECT checksum FROM schema_migrations WHERE version = $1"').replace('"INSERT INTO schema_migrations(version, name, checksum, applied_at) VALUES (?, ?, ?, ?)"','"INSERT INTO schema_migrations(version, name, checksum, applied_at) VALUES ($1, $2, $3, $4)"')
 if name=='migrations.go':
  s=s.replace('defer tx.Rollback()', 'defer tx.Rollback()\n if _,err:=tx.ExecContext(ctx,"SELECT pg_advisory_xact_lock(846239010)");err!=nil{return err}',1)
 (target/name).write_text('// Code generated by tools/postgresgen/generate.py; DO NOT EDIT.\n'+s)
for p in (source/'migrations').glob('*.sql'):
 sql=p.read_text().replace('COLLATE BINARY','COLLATE "C"')
 if p.name=='0032_workspace_execution_time.sql':
  shared=(source/'workspace_read.go').read_text()
  for constant,fragment in [('workspaceExecutionSeconds','workspace_execution_seconds.sql'),('workspaceExecutionNanos','workspace_execution_nanos.sql')]:
   original=re.search(r'const '+constant+r' = `([^`]+)`',shared)[1]
   assert sql.count(original)==2,constant
   sql=sql.replace(original,Path('tools/postgresgen',fragment).read_text().strip())
 if p.name=='0031_entity_notes.sql':
  sql=sql.replace("outcome_revision TEXT COLLATE \"C\" NOT NULL CHECK (length(outcome_revision)=20 AND outcome_revision NOT GLOB '*[^0-9]*' AND outcome_revision>'00000000000000000000' AND outcome_revision<='18446744073709551615')",'outcome_revision NUMERIC(20,0) NOT NULL CHECK (outcome_revision>0 AND outcome_revision<=18446744073709551615)')
 if p.name=='0030_work_activity.sql':
  sql=sql.replace("sequence TEXT COLLATE \"C\" NOT NULL CHECK (length(sequence)=20 AND sequence NOT GLOB '*[^0-9]*' AND sequence>'00000000000000000000' AND sequence<='18446744073709551615')",'sequence NUMERIC(20,0) NOT NULL CHECK (sequence>0 AND sequence<=18446744073709551615)')
 if p.name=='0016_work_contracts.sql':sql=sql.replace('fencing_token TEXT NOT NULL','fencing_token NUMERIC(20,0) NOT NULL CHECK (fencing_token>0 AND fencing_token<=18446744073709551615)')
 if p.name=='0021_signed_contracts_reviews.sql':
  sql=sql.replace("version TEXT NOT NULL CHECK (version!='0')","version NUMERIC(20,0) NOT NULL CHECK (version>0 AND version<=18446744073709551615)").replace("lease_version TEXT NOT NULL CHECK (lease_version!='0')","lease_version NUMERIC(20,0) NOT NULL CHECK (lease_version>0 AND lease_version<=18446744073709551615)").replace(" version TEXT NOT NULL CHECK(version!='0')"," version NUMERIC(20,0) NOT NULL CHECK(version>0 AND version<=18446744073709551615)").replace("lease_version TEXT NOT NULL CHECK(lease_version!='0')","lease_version NUMERIC(20,0) NOT NULL CHECK(lease_version>0 AND lease_version<=18446744073709551615)").replace('fencing_token TEXT NOT NULL','fencing_token NUMERIC(20,0) NOT NULL CHECK (fencing_token>0 AND fencing_token<=18446744073709551615)')
 # PostgreSQL stores booleans as integer in this shared schema, matching existing
 # relational DTO codecs and preserving all checks. Credential code adapts its bool.
 (target/'migrations'/p.name).write_text(sql)

# Execute the same durability and authorization contracts against a real database.
for name in ['completion_policy_test.go','workspace_scope_preview_test.go','workspace_conclusion_test.go','workspace_lease_test.go','workspace_observation_test.go','workspace_execution_time_test.go','workspace_proof_test.go','workspace_editing_test.go','workspace_read_test.go','workspace_compound_test.go','entity_notes_test.go','workspace_timeline_test.go','work_activity_application_test.go','work_activity_test.go','assessment_plan_fixture_test.go','principal_review_quota_test.go','principal_quota_test.go','focal_graph_read_test.go','continuity_projection_test.go','continuity_cursor_test.go','continuity_read_test.go','delivery_health_test.go','signed_return_resolution_test.go','signed_load_test.go','migration_compatibility_test.go','issue_assignment_migration_test.go','issuer_recovery_test.go','signed_next_test.go','signed_no_cache_test.go','signed_historical_source_test.go','signed_cutover_test.go','signed_query_scope_test.go','signed_review_journey_test.go','signed_return_test.go','signed_acquisition_test.go','signed_contracts_test.go','signing_identity_test.go','work_protocol_test.go','work_contract_test.go','restart_test.go','dependencies_test.go','leases_test.go','documentary_test.go','issues_blockers_test.go','wave10_assessment_test.go','wave10_conclusion_test.go','wave10_immutability_test.go','wave11_roadmap_restart_test.go','security_contract_test.go','security_admin_test.go','context_contract_test.go','external_context_test.go','integration_contract_test.go','continuity_contract_test.go','administrative_override_test.go','release_concurrency_test.go','bulk_history_test.go']:
 s=(source/name).read_text().replace('package sqlite','package postgres',1).replace('sqliteObjectiveForWorkFixture','postgresObjectiveForWorkFixture')
 s=re.sub(r'\bOpen\(', 'openTestPostgres(', s)
 s=re.sub(r'(func Test\w+\(t \*testing.T\) \{)',r'\1\n requirePostgres(t)',s)
 s=re.sub(r'`([^`]*)`',transform_sql,s,flags=re.S)
 (target/name).write_text('// Code generated by tools/postgresgen/generate.py; DO NOT EDIT.\n'+s)

benchmark=(source/'continuity_benchmark_test.go').read_text().replace('package sqlite','package postgres',1).replace('Open(', 'openTestPostgres(',1).replace('sqliteObjectiveForWorkFixture','postgresObjectiveForWorkFixture')
benchmark=benchmark.replace('sqlite3 "github.com/ncruces/go-sqlite3"','').replace('"github.com/ncruces/go-sqlite3/driver"','')
start=benchmark.index('var sqlCount atomic.Uint64')
end=benchmark.index('for _, name := range',start)
benchmark=benchmark[:start]+benchmark[end:]
benchmark=benchmark.replace('func BenchmarkContinuity(b *testing.B) {','func BenchmarkContinuity(b *testing.B) {\n if os.Getenv("WOS_TEST_POSTGRES_DSN")=="" {b.Skip("real PostgreSQL required")}')
benchmark=benchmark.replace('before := sqlCount.Load()', 'var before int64; _ = store.db.QueryRowContext(ctx,"SELECT coalesce(sum(calls),0) FROM public.pg_stat_statements").Scan(&before)')
benchmark=benchmark.replace('b.ReportMetric(float64(sqlCount.Load()-before)/float64(b.N), "sql_statements/op")','var after int64; if err:=store.db.QueryRowContext(ctx,"SELECT coalesce(sum(calls),0) FROM public.pg_stat_statements").Scan(&after); err==nil {b.ReportMetric(float64(after-before-1)/float64(b.N), "sql_statements/op")}')
(target/'continuity_benchmark_test.go').write_text('// Code generated by tools/postgresgen/generate.py; DO NOT EDIT.\n'+benchmark)

# Identical domain-validated operational read harness for before/after fixtures.
operational=(source/'operational_read_benchmark_test.go').read_text().replace('package sqlite','package postgres',1).replace('Open(', 'openTestPostgres(',1)
(target/'operational_read_benchmark_test.go').write_text('// Code generated by tools/postgresgen/generate.py; DO NOT EDIT.\n'+operational)

# Opt-in contract load fixture shares the workload; SQLite tracing does not
# masquerade as PostgreSQL statement measurement.
load=(source/'work_contract_load_test.go').read_text().replace('package sqlite','package postgres',1)
load=load.replace('sqlite3 "github.com/ncruces/go-sqlite3"','').replace('"github.com/ncruces/go-sqlite3/driver"','')
load=load.replace('func TestWorkContractLoadMatrix(t *testing.T) {','func TestWorkContractLoadMatrix(t *testing.T) {\n requirePostgres(t)')
start=load.index('// SQL_TRACE_START');end=load.index('// SQL_TRACE_END',start)+len('// SQL_TRACE_END')
load=load[:start]+'var sqlCount atomic.Uint64; sqlMeasured:=false'+load[end:]
(target/'work_contract_load_test.go').write_text('// Code generated by tools/postgresgen/generate.py; DO NOT EDIT.\n'+load)

# The F08 workload and timing boundaries are identical across adapters. Each
# adapter uses its own real statement tracer, never an estimated query count.
workspace_f08=(source/'workspace_f08_benchmark_test.go').read_text().replace('package sqlite','package postgres',1).replace('Open(', 'openTestPostgres(',1).replace('adapter=SQLite','adapter=PostgreSQL')
workspace_f08=workspace_f08.replace('"path/filepath"','"path/filepath"\n"os"')
workspace_f08=workspace_f08.replace('func BenchmarkWorkspaceF08(b *testing.B) {','func BenchmarkWorkspaceF08(b *testing.B) {\n if os.Getenv("WOS_TEST_POSTGRES_DSN")=="" {b.Skip("real PostgreSQL required")}')
(target/'workspace_f08_benchmark_test.go').write_text('// Code generated by tools/postgresgen/generate.py; DO NOT EDIT.\n'+workspace_f08)

# Generated adapters are committed Go source. Keep regeneration deterministic,
# including blank imports removed from the benchmark variants above.
subprocess.run(['gofmt','-w']+[str(path) for path in sorted(target.glob('*.go'))],check=True)
