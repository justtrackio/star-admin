import assert from 'node:assert/strict';
import { test } from 'node:test';
import { isFrontendConfigResponse } from '../src/api/snapshots';

test('validates config snapshots with version-specific columns and empty values', () => {
  const columns = ['Key', 'Value', 'Type', 'IsMutable', 'AliasNames', 'Comment'];
  const setting = { Key: 'query_port', Value: '9030', Type: 'int', IsMutable: 'false', AliasNames: '[]', Comment: '' };
  assert.equal(isFrontendConfigResponse({ columns, settings: [] }), true);
  for (const Value of ['', '0', 'false', '******']) {
    assert.equal(isFrontendConfigResponse({ columns, settings: [{ ...setting, Value }] }), true);
  }
  assert.equal(isFrontendConfigResponse({ columns: [...columns, 'MasterOnly'], settings: [{ ...setting, MasterOnly: 'true' }] }), true);
  for (const invalid of [null, {}, { columns: ['Key'], settings: [] }, { columns: [null], settings: [] },
    { columns, settings: null }, { columns, settings: [null] }, { columns, settings: [{ ...setting, Value: 9030 }] },
    { columns, settings: [{ Key: 'missing-fields', Value: '' }] }]) {
    assert.equal(isFrontendConfigResponse(invalid), false);
  }
});
