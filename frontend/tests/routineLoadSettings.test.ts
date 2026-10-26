import assert from 'node:assert/strict';
import { test } from 'node:test';
import { isRoutineLoadSettings } from '../src/api/snapshots';

test('accepts cluster settings and rejects malformed updates', () => {
  assert.equal(isRoutineLoadSettings({ maxBatchSize: '4294967296', taskConsumeSecond: '15' }), true);
  assert.equal(isRoutineLoadSettings({ maxBatchSize: '', taskConsumeSecond: '' }), true);
  for (const invalid of [
    null,
    [],
    {},
    { maxBatchSize: 4294967296, taskConsumeSecond: '15' },
    { maxBatchSize: '4294967296', taskConsumeSecond: null },
  ]) {
    assert.equal(isRoutineLoadSettings(invalid), false);
  }
});
