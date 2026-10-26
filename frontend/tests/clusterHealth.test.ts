import assert from 'node:assert/strict';
import { test } from 'node:test';
import {
  isDictionaryHealthy,
  isNodeHealthy,
  isRoutineHealthy,
  isViewHealthy,
  resourceStatus,
} from '../src/utils/clusterHealth';

test('nodes require positive availability and no decommissioning or errors', () => {
  const node = { alive: true, decommissioned: false, error: '' };
  assert.equal(isNodeHealthy(node), true);
  for (const fields of [{ alive: false }, { alive: null }, { decommissioned: true }, { error: 'Heartbeat failed' }]) {
    assert.equal(isNodeHealthy({ ...node, ...fields }), false);
  }
});

test('routine health accepts scheduled work but rejects paused, terminal and unknown states', () => {
  for (const state of ['RUNNING', 'NEED_SCHEDULE']) assert.equal(isRoutineHealthy({ state }), true);
  for (const state of ['PAUSED', 'UNSTABLE', 'CANCELLED', 'STOPPED', ''])
    assert.equal(isRoutineHealthy({ state }), false);
});

test('dictionary errors override a successful or in-progress status', () => {
  for (const status of ['FINISHED', 'REFRESHING', 'COMMITTING']) {
    assert.equal(isDictionaryHealthy({ status, error: '' }), true);
    assert.equal(isDictionaryHealthy({ status, error: 'Refresh rolled back' }), false);
  }
  for (const status of ['CANCELLED', 'UNINITIALIZED', ''])
    assert.equal(isDictionaryHealthy({ status, error: '' }), false);
});

test('views require active status and a known non-failed refresh state without errors', () => {
  const view = { active: 'true', inactiveReason: '', lastRefreshState: 'SUCCESS', lastRefreshError: '' };
  for (const lastRefreshState of ['SUCCESS', 'SKIPPED', 'PENDING', 'RUNNING', 'MERGED']) {
    assert.equal(isViewHealthy({ ...view, lastRefreshState }), true);
  }
  for (const fields of [
    { active: 'false' },
    { active: '' },
    { inactiveReason: 'Missing table' },
    { lastRefreshState: 'FAILED' },
    { lastRefreshState: '' },
    { lastRefreshError: 'Timeout' },
  ]) {
    assert.equal(isViewHealthy({ ...view, ...fields }), false);
  }
});

test('cards never report green for errors, loading, empty resources, or partial health', () => {
  assert.equal(resourceStatus(undefined, 0, null), 'loading');
  assert.equal(resourceStatus(0, 0, null), 'empty');
  assert.equal(resourceStatus(0, 0, null, true), 'unhealthy');
  assert.equal(resourceStatus(3, 3, null), 'healthy');
  assert.equal(resourceStatus(3, 2, null), 'unhealthy');
  assert.equal(resourceStatus(3, 3, new Error('Disconnected')), 'unavailable');
  assert.equal(resourceStatus(undefined, 0, new Error('No snapshot')), 'unavailable');
});
