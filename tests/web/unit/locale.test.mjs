import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {copy} from '../../../packages/wos-api/web/assets/en-US.js';
import {date,display,kindName,formatValue} from '../../../packages/wos-api/web/assets/presentation.js';

const asset = name => new URL(`../../../packages/wos-api/web/assets/${name}`,import.meta.url);
const decode = value => value.replaceAll('&amp;','&').replaceAll('&lt;','<').replaceAll('&gt;','>').replaceAll('&quot;','"').replaceAll('&#x27;',"'");

test('official human terms distinguish tasks, outcomes and waived verification',()=>{
  assert.equal(copy.locale,'en-US');
  assert.equal(kindName('work_item'),'Task');
  assert.equal(kindName('outcome'),'Outcome');
  assert.equal(kindName('objective'),'Objective');
  assert.equal(display('done'),'Done');
  assert.equal(display('achieved'),'Achieved');
  assert.equal(display('waived'),'Waived (not verified)');
  assert.match(date('2026-10-09T12:00:00Z'),/Oct 9, 2026.*12:00 PM UTC/);
});

test('typed labels never translate authored content or identifiers',()=>{
  assert.equal(formatValue('active','lifecycle'),'Active');
  assert.equal(formatValue('active','description'),'active');
  assert.equal(formatValue('not_met','title'),'not_met');
  assert.equal(formatValue('Não alterar meu texto','rationale'),'Não alterar meu texto');
  assert.equal(formatValue('run_with_context','principal_id'),'run_with_context');
});

test('static shell fallbacks agree with the single official copy catalog',async()=>{
  const html=await readFile(asset('index.html'),'utf8');
  assert.match(html,/<html lang="en">/);
  let count=0;
  for(const [,key,text] of html.matchAll(/<[^>]+\bdata-copy="([^"]+)"[^>]*>([^<]+)</g)) {
    assert.equal(decode(text.trim()),copy.text[key],`stale shell copy: ${key}`);count++;
  }
  assert.ok(count>50,'shell copy must actually be checked');
  for(const [,attribute,text,key] of html.matchAll(/(aria-label|title|placeholder)="([^"]+)" data-copy-\1="([^"]+)"/g)) assert.equal(decode(text),copy.text[key],`${attribute}: ${key}`);
});

test('official web source contains no Portuguese locale or accented copy',async()=>{
  // Historical documents and authored fixtures are outside this official-copy
  // gate. This does not prohibit Unicode supplied by users.
  for(const name of ['index.html','app.js','presentation.js','en-US.js','command-exposure.json']) {
    const text=await readFile(asset(name),'utf8');
    assert.doesNotMatch(text,/pt-BR|[À-ÖØ-öø-ÿ]/,name);
  }
});
