export function el(tag,text,className){const node=document.createElement(tag);if(text!==undefined)node.textContent=text;if(className)node.className=className;return node;}
export function button(label,action,className=''){const b=el('button',label,className);b.type='button';b.onclick=action;return b;}
let fieldSequence=0;
export function inputField(label,{value='',type='text',required=false,options,help}={}){
 const wrap=el('div',undefined,'human-field'),l=el('label',label),input=el(options?'select':type==='textarea'?'textarea':'input');if(options)for(const [value,text] of options)input.append(Object.assign(el('option',text),{value}));else if(type!=='textarea')input.type=type;
 if(type==='checkbox')input.checked=Boolean(value);else input.value=options?(value||options[0]?.[0]||''):(value??'');input.required=required&&type!=='checkbox';input.id=`human-field-${++fieldSequence}`;l.htmlFor=input.id;wrap.append(l,input);if(help)wrap.append(el('small',help,'field-help'));
 return {node:wrap,input,get:()=>type==='checkbox'?input.checked:input.value.trim()||undefined};
}
export function listField(label,initial=[]){const group=el('fieldset'),legend=el('legend',label),rows=el('div');group.append(legend,rows);const entries=[];function add(value=''){const f=inputField(`${label} item`,{value}),row=el('div',undefined,'list-editor-row'),entry={f,active:true};entries.push(entry);row.append(f.node,button('Remove item',()=>{entry.active=false;row.remove()},'quiet'));rows.append(row);}initial.forEach(add);group.append(button(`Add ${label.toLowerCase()} item`,()=>add(),'quiet'));return {node:group,get:()=>entries.filter(x=>x.active).map(x=>x.f.get()).filter(Boolean)};}
export function details(label){const node=el('details',undefined,'human-advanced');node.append(el('summary',label));return node;}
