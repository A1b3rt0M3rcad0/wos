import assert from 'node:assert/strict';
import { normalizeLabel } from './normalize-label.mjs';
const checks = [];
function check(name, fn) { fn(); checks.push({name, passed:true}); }
check('trim', () => assert.equal(normalizeLabel(' \t label \r\n'), 'label'));
check('internal spaces', () => assert.equal(normalizeLabel('a   b'), 'a b'));
check('mixed whitespace', () => assert.equal(normalizeLabel(' \ta\t \r\n b\u00a0\u2003c '), 'a b c'));
check('whitespace-only', () => assert.equal(normalizeLabel(' \t\r\n\u00a0\u2003'), ''));
check('empty string', () => assert.equal(normalizeLabel(''), ''));
check('unchanged text', () => assert.equal(normalizeLabel('alpha beta'), 'alpha beta'));
check('nonstrings throw TypeError', () => {
  for (const value of [undefined, null, true, false, 0, 42, NaN, 1n, Symbol('x'), {}, [], () => {}, new String('x')]) {
    assert.throws(() => normalizeLabel(value), TypeError);
  }
});
console.log(JSON.stringify({passed:checks.length, failed:0, nonstringCases:13, checks}));
