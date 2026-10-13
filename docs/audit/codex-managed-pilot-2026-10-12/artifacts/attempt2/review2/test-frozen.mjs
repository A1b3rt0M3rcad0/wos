import assert from 'node:assert/strict';
import { readFileSync, statSync, writeFileSync } from 'node:fs';
import { createHash } from 'node:crypto';
import { normalizeLabel } from '/tmp/wos-t89-managed-pilot/artifacts/attempt2/round2/normalize-label.mjs';
const path = '/tmp/wos-t89-managed-pilot/artifacts/attempt2/round2/normalize-label.mjs';
const expected = '4443b5354ece623cc50a14586685603780d92fddd2cc903813c9177725ab704d';
const checksum = () => createHash('sha256').update(readFileSync(path)).digest('hex');
assert.equal(checksum(), expected);
assert.equal(statSync(path).mode & 0o777, 0o444);
const cases = [
 ['trim', ' \t label \r\n', 'label'],
 ['internal spaces', 'a   b', 'a b'],
 ['mixed whitespace', '\t a\t \n\r b\v\f c \n', 'a b c'],
 ['whitespace only', ' \t\r\n\v\f ', ''],
 ['empty string', '', ''],
 ['unchanged text', 'plain label', 'plain label'],
 ['unicode whitespace', '\u00a0a\u2003\u2028b\ufeff', 'a b'],
];
const results = [];
for (const [name, input, output] of cases) {
 assert.equal(normalizeLabel(input), output, name);
 results.push({name, passed:true});
}
for (const [name, value] of [['null',null], ['undefined',undefined], ['number zero',0], ['number finite',42], ['number NaN',NaN], ['number Infinity',Infinity], ['boolean true',true], ['boolean false',false], ['object',{}], ['array',[]], ['boxed string',new String('label')], ['bigint',1n], ['symbol',Symbol('x')], ['function',()=>{}]]) {
 assert.throws(()=>normalizeLabel(value), TypeError, name);
 results.push({name:`TypeError ${name}`,passed:true});
}
assert.equal(checksum(), expected);
const result = {artifact:path, sha256:expected, mode:'0444', test_count:results.length, passed:results.length, failed:0, checksum_before_and_after_matches:true, results};
writeFileSync('/tmp/wos-t89-managed-pilot/artifacts/attempt2/review2/test-results.json',JSON.stringify(result,null,2)+'\n');
console.log(JSON.stringify(result));
