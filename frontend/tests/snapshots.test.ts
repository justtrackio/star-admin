import assert from 'node:assert/strict';
import { test } from 'node:test';
import {
  isDictionariesResponse,
  isResourceGroupsResponse,
  isNodesResponse,
  isRoutineLoadsResponse,
  isTablesResponse,
} from '../src/api/snapshots';

test('accepts complete snapshots and rejects malformed live updates', () => {
  const table = { database: 'analytics', name: 'events', engine: 'OLAP', rows: null, dataBytes: 123 };
  const job = {
    id: '9007199254740993',
    database: 'analytics',
    table: 'events',
    name: 'kafka',
    state: 'RUNNING',
    topic: 'events',
    loadedRows: '100',
    errorRows: '0',
    rowsPerSecond: '10',
    progress: '{}',
    reason: '',
    taskConsumeSecond: '15',
  };
  const dictionary = {
    id: '1',
    name: 'lookup',
    catalog: 'default_catalog',
    database: 'analytics',
    source: 'events',
    keys: 'id',
    values: 'name',
    status: 'FINISHED',
    lastRefreshStarted: '',
    lastRefreshFinished: '',
    nextRefresh: '',
    error: '',
    memoryUsage: '',
  };
  const cases = [
    { validate: isTablesResponse, key: 'tables', row: table, badRow: { ...table, rows: '123' } },
    { validate: isRoutineLoadsResponse, key: 'jobs', row: job, badRow: { ...job, id: 123 } },
    {
      validate: isDictionariesResponse,
      isNodesResponse,
      key: 'dictionaries',
      row: dictionary,
      badRow: { ...dictionary, status: null },
    },
  ];
  for (const { validate, key, row, badRow } of cases) {
    assert.equal(validate({ [key]: [] }), true);
    assert.equal(validate({ [key]: [row] }), true);
    for (const invalid of [null, [], {}, { [key]: null }, { [key]: [null] }, { [key]: [{}] }, { [key]: [badRow] }]) {
      assert.equal(validate(invalid), false);
    }
  }
});

test('validates node snapshots for all types and rejects invalid metadata', () => {
  const node = {
    id: '9007199254740993',
    type: 'compute',
    host: 'cn.local',
    role: '',
    alive: true,
    decommissioned: false,
    status: '',
    warehouse: '',
    cpuCores: 8,
    memoryLimit: '',
    cpuUsed: '',
    memoryUsed: '',
    runningQueries: 0,
    tablets: null,
    lastStartTime: '',
    lastHeartbeat: '',
    version: '',
    error: '',
    dataCache: '',
    details: { BrpcPort: '8060' },
  };
  assert.equal(isNodesResponse({ nodes: [] }), true);
  for (const type of ['frontend', 'backend', 'compute']) {
    assert.equal(isNodesResponse({ nodes: [{ ...node, type }] }), true);
  }
  for (const invalid of [
    null,
    {},
    { nodes: [null] },
    ...[
      { type: 'broker' },
      { type: ['compute'] },
      { id: 123 },
      { alive: 'true' },
      { cpuCores: '8' },
      { details: null },
      { details: { port: 8060 } },
    ].map((fields) => ({ nodes: [{ ...node, ...fields }] })),
  ]) {
    assert.equal(isNodesResponse(invalid), false);
  }
});

test('validates routine consumption limits without losing the configured value', () => {
  const job = {
    id: '1',
    database: 'analytics',
    table: 'events',
    name: 'kafka',
    state: 'RUNNING',
    topic: 'events',
    loadedRows: '100',
    errorRows: '0',
    rowsPerSecond: '10',
    progress: '{}',
    reason: '',
    taskConsumeSecond: '30',
  };
  assert.equal(isRoutineLoadsResponse({ jobs: [job] }), true);
  assert.equal(isRoutineLoadsResponse({ jobs: [{ ...job, taskConsumeSecond: '' }] }), true);
  assert.equal(isRoutineLoadsResponse({ jobs: [{ ...job, taskConsumeSecond: 30 }] }), false);
});

test('validates resource groups with multiple classifiers and unknown configuration fields', () => {
  const group = {
    id: '9007199254740993',
    name: 'analytics',
    properties: { mem_limit: '50%', concurrency_limit: '0', future_limit: '' },
    classifiers: ["(id=1, user='analytics')", "(id=2, db='analytics')"],
  };
  assert.equal(isResourceGroupsResponse({ groups: [] }), true);
  assert.equal(isResourceGroupsResponse({ groups: [group] }), true);
  assert.equal(isResourceGroupsResponse({ groups: [{ ...group, properties: {}, classifiers: [] }] }), true);
  for (const invalid of [
    null,
    [],
    {},
    { groups: null },
    { groups: [null] },
    ...[
      { id: 123 },
      { properties: null },
      { properties: { mem_limit: 50 } },
      { classifiers: 'rule' },
      { classifiers: [null] },
    ].map((fields) => ({ groups: [{ ...group, ...fields }] })),
  ]) {
    assert.equal(isResourceGroupsResponse(invalid), false);
  }
});
