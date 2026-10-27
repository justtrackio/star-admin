import assert from 'node:assert/strict';
import { test } from 'node:test';
import { formatFreshness } from '../src/utils/formatFreshness';

test('formats refresh age at unit boundaries and handles missing or future times', () => {
  for (const [seconds, expected] of [
    [undefined, '—'], [null, '—'], [NaN, '—'], [-5, '0s'], [0, '0s'], [59.9, '59s'],
    [60, '1m 0s'], [3599, '59m 59s'], [3600, '1h 0m'],
    [86399, '23h 59m'], [86400, '1d 0h'], [176400, '2d 1h'],
  ] as const) {
    assert.equal(formatFreshness(seconds), expected);
  }
});
