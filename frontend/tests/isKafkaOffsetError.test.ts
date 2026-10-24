import assert from 'node:assert/strict';
import { test } from 'node:test';
import { isKafkaOffsetError } from '../src/utils/isKafkaOffsetError';

test('offers recovery for Kafka offset errors only', () => {
  assert.equal(isKafkaOffsetError("ErrorReason{errCode = 104, msg='be 10003 abort task: requested offset not available on the broker: Broker: Offset out of range (broker 0)'}"), true);
  assert.equal(isKafkaOffsetError('Requested offset not available'), true);
  assert.equal(isKafkaOffsetError('Broker: Offset out of range'), true);
  assert.equal(isKafkaOffsetError('too many error rows'), false);
  assert.equal(isKafkaOffsetError('permission denied'), false);
  assert.equal(isKafkaOffsetError(''), false);
});
