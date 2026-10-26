import assert from 'node:assert/strict';
import { test } from 'node:test';
import { isRoutineLoadTasksResponse } from '../src/api/snapshots';

test('validates routine task snapshots and preserves large IDs and offsets as strings', () => {
  const task = { id: 'task-1', jobId: '9007199254740993', transactionId: '9007199254740994', transactionStatus: 'UNKNOWN', created: '2026-10-24 14:51:38', lastScheduled: '', executeStarted: '', timeout: '60', nodeId: '-1', progress: 'Progress:{"0":9007199254740993},LatestOffset:null', message: '', warehouse: '' };
  assert.equal(isRoutineLoadTasksResponse({ tasks: [] }), true);
  assert.equal(isRoutineLoadTasksResponse({ tasks: [task] }), true);
  for (const invalid of [null, {}, { tasks: null }, { tasks: [null] }, { tasks: [{}] }, ...[
    { jobId: 123 }, { progress: {} }, { created: null }, { message: undefined },
  ].map(fields => ({ tasks: [{ ...task, ...fields }] }))]) {
    assert.equal(isRoutineLoadTasksResponse(invalid), false);
  }
});
