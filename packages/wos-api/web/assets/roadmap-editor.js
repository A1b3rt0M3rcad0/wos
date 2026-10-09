import {el,button,inputField} from './ui.js';
import {referencePicker} from './reference-picker.js';
export function roadmapEditor({entity,scope,request}) {
 const root=el('section',undefined,'roadmap-editor'),rows=el('div'),models=[],disposers=[];
 root.append(el('p','Roadmaps reference work. Removing a reference never deletes or cancels its Task or Objective. Saving order does not change operational dependencies.','field-help'));
 function add(node={}){
  const item={...structuredClone(node),node_key:node.node_key||crypto.randomUUID(),node_type:node.node_type||'phase'};
  const title=inputField('Item title',{value:item.title,required:true});
  const parent=inputField('Within phase',{options:[['','No parent']]});
  const start=inputField('Planned start (local time)',{type:'datetime-local',value:local(item.planned_start)}),end=inputField('Planned end (local time)',{type:'datetime-local',value:local(item.planned_end)});
  const picker=item.node_type==='reference'?referencePicker({label:'Reference objective or task',kinds:['objective','work_item'],scope,request,required:true,initial:item.target_ref?[item.target_ref]:[]}):null;
  if(picker)disposers.push(picker.dispose);
  const row=el('article',undefined,'roadmap-edit-row');row.append(el('h3',item.node_type==='reference'?'Work reference':item.node_type==='phase'?'Phase':'Milestone'),title.node,parent.node);if(picker)row.append(picker.node);
  const schedule=el('details');schedule.append(el('summary','Schedule'),start.node,end.node);row.append(schedule);
  const model={item,title,parent,start,end,picker,row};models.push(model);
  row.append(button('Move up',()=>move(model,-1),'quiet'),button('Move down',()=>move(model,1),'quiet'),button('Remove from draft',()=>{models.splice(models.indexOf(model),1);picker?.dispose();paint()},'quiet'));parent.input.value=item.parent_node_key||'';paint();
 }
 function move(model,delta){const i=models.indexOf(model),j=i+delta;if(j<0||j>=models.length)return;[models[i],models[j]]=[models[j],models[i]];paint();model.row.querySelector('input')?.focus();}
 function paint(){for(const m of models){const value=m.parent.input.value||m.item.parent_node_key||'';m.parent.input.replaceChildren(Object.assign(el('option','No parent'),{value:''}));for(const other of models.filter(x=>x.item.node_type==='phase'&&x!==m)){m.parent.input.append(Object.assign(el('option',other.title.input.value||'Untitled phase'),{value:other.item.node_key}));}m.parent.input.value=value;if(!m.parent.input.value)m.item.parent_node_key=undefined; }rows.replaceChildren(...models.map(m=>m.row));}
 for(const n of entity.draft?.nodes||[])add(n);
 root.append(rows,button('Add phase',()=>add({node_type:'phase'}),'quiet'),button('Add milestone',()=>add({node_type:'milestone'}),'quiet'),button('Reference objective/task',()=>add({node_type:'reference'}),'quiet'));
 return {node:root,get(){const nodes=models.map((m,position)=>{const title=m.title.get();if(!title)throw Error('Every roadmap item needs a title.');return {...m.item,title,position,parent_node_key:m.parent.get(),target_ref:m.picker?m.picker.get():m.item.target_ref,planned_start:iso(m.start.get()),planned_end:iso(m.end.get())}});const keys=new Set(nodes.map(x=>x.node_key));return {nodes,after_links:(entity.draft?.after_links||[]).filter(l=>keys.has(l.node_key)&&keys.has(l.after_node_key))};},dispose(){disposers.forEach(f=>f());}};
}
function local(value){if(!value)return '';const d=new Date(value);return new Date(d.getTime()-d.getTimezoneOffset()*60000).toISOString().slice(0,16);}
function iso(value){return value?new Date(value).toISOString():undefined;}
export function roadmapDiff(entity){const root=el('section',undefined,'roadmap-diff');root.append(el('h3','Review changes before publication'));const previous=entity.revisions?.find(r=>r.revision_number===entity.draft?.base_revision_number)?.nodes||[];const next=entity.draft?.nodes||[];let changes=0;
 for(const n of next){const old=previous.find(x=>x.node_key===n.node_key);if(!old||JSON.stringify(old)!==JSON.stringify(n)){root.append(el('p',`${old?'Changed':'Added'}: ${n.title}`));changes++;}}
 for(const n of previous)if(!next.some(x=>x.node_key===n.node_key)){root.append(el('p',`Removed reference or plan item: ${n.title}`));changes++;}
 if(!changes)root.append(el('p','No content changes from the selected base revision.'));
 root.append(el('p','Publishing freezes this revision. Activation is a separate action. Tasks and Objectives retain independent lifecycles.'));return root;
}
