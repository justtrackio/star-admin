import assert from 'node:assert/strict';
import { test } from 'node:test';
import { formatPartitionName } from '../src/utils/formatPartitionName';

test('formats timestamp partitions and preserves other names', () => {
  assert.equal(formatPartitionName('p20261017000000'), '2026-10-17 00:00:00Z');
  assert.equal(formatPartitionName('p20261017123456'), '2026-10-17 12:34:56Z');
  assert.equal(formatPartitionName('p20261017'), 'p20261017');
  assert.equal(formatPartitionName('custom_partition'), 'custom_partition');
});
