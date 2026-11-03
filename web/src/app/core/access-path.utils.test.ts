import assert from 'node:assert/strict'
import { test } from 'node:test'
import { accessPath } from './access-path.utils.ts'

test('access requests preserve explicit zero and false values while escaping identifiers and opaque cursors', () => {
  assert.equal(accessPath('/groups/{group}/members/{user}', { group: 'a/b', user: 'u?x=1' }, { project: '', limit: 0, archived: false, cursor: 'a+b/=', missing: undefined }), '/groups/a%2Fb/members/u%3Fx%3D1?project=&limit=0&archived=false&cursor=a%2Bb%2F%3D')
  assert.throws(() => accessPath('/users/{user}', {}), /Missing access path parameter/)
  assert.throws(() => accessPath('/users/{user}', { user: '' }), /Missing access path parameter/)
})
