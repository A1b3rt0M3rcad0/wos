import {randomUUID} from 'node:crypto';

// Public commands only. A fresh server/database owns each seed; no existing
// user's data is reset. Secrets are passed by the caller and never logged.
export async function seedWorkspace({url, token, namespace, size = 4, label = 'UX acceptance'}) {
  if (![0, 4, 32, 1001].includes(size)) throw Error('Use an explicit acceptance dataset size');
  async function command(name, command) {
    const response = await fetch(`${url}/api/v1/commands/${name}`, {
      method: 'POST', headers: {'Content-Type':'application/json', Authorization:`Bearer ${token}`, 'Idempotency-Key':randomUUID()},
      body: JSON.stringify({command}),
    });
    const result = await response.json();
    if (!response.ok) throw Error(`Seed command ${name}: ${response.status} ${JSON.stringify(result)}`);
    return result.value;
  }
  if (!size) return {outcome:null, objectives:[], tasks:[], evidence:[]};
  const outcome = await command('create_outcome', {namespace_id:namespace, title:label, desired_state:'Humans coordinate verifiable work without internal command knowledge', description:'Repeatable public-command UX dataset', priority:'normal'});
  const scope = {namespace_id:namespace, outcome_id:outcome.id};
  const objectives=[];
  for (let i=0;i<2;i++) objectives.push(await command('create_objective', {scope,title:'Validation objective',description:`Duplicate title, distinct objective ${i+1}`,priority:'normal',required_for_outcome:false}));
  const tasks=[], evidence=[];
  for (let i=0;i<size;i++) {
    tasks.push(await command('create_work_item', {scope,title:i===size-1?'Off-page acceptance task':'Validation task',description:`Distinct task ${i+1}`,priority:i%4===0?'high':'normal',lifecycle:i%3===0?'backlog':'todo',objective_id:objectives[i%2].id,execution_spec:{instructions:['Run only the scoped validation'],deliverables:['A bounded result'],constraints:[],scope_hints:[],context_refs:[]}}));
    evidence.push(await command('register_evidence', {scope,evidence_type:'test_result',description:i===size-1?'Off-page acceptance evidence':'Validation observation',source_ref:{provider:'ux-fixture',id:`observation-${i}`},captured_at:'2026-10-09T12:00:00Z'}));
  }
  return {scope,outcome,objectives,tasks,evidence};
}
