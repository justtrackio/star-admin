import assert from 'node:assert/strict';
import { test } from 'node:test';
import { isDictionariesResponse, isRoutineLoadsResponse, isTablesResponse } from '../src/api/snapshots';

test('accepts complete snapshots and rejects malformed live updates', () => {
  const table = { database: 'analytics', name: 'events', engine: 'OLAP', rows: null, dataBytes: 123 };
  const job = { id: '9007199254740993', database: 'analytics', table: 'events', name: 'kafka', state: 'RUNNING', topic: 'events', loadedRows: '100', errorRows: '0', rowsPerSecond: '10', progress: '{}', reason: '' };
  const dictionary = { id: '1', name: 'lookup', catalog: 'default_catalog', database: 'analytics', source: 'events', keys: 'id', values: 'name', status: 'FINISHED', lastRefreshStarted: '', lastRefreshFinished: '', nextRefresh: '', error: '', memoryUsage: '' };
  const cases = [
    { validate: isTablesResponse, key: 'tables', row: table, badRow: { ...table, rows: '123' } },
    { validate: isRoutineLoadsResponse, key: 'jobs', row: job, badRow: { ...job, id: 123 } },
    { validate: isDictionariesResponse, key: 'dictionaries', row: dictionary, badRow: { ...dictionary, status: null } },
  ];
  for (const { validate, key, row, badRow } of cases) {
    assert.equal(validate({ [key]: [] }), true);
    assert.equal(validate({ [key]: [row] }), true);
    for (const invalid of [null, [], {}, { [key]: null }, { [key]: [null] }, { [key]: [{}] }, { [key]: [badRow] }]) {
      assert.equal(validate(invalid), false);
    }
  }
});
