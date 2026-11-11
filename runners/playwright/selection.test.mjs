import assert from 'node:assert/strict'
import { test } from 'node:test'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { selectionArguments } from './selection.mjs'

test('untagged selection names one project, file, and title path without editing source', () => {
 const root = fs.mkdtempSync(path.join(os.tmpdir(), 'teststudio-selection-'))
 try {
 const filename = path.join(root, 'selection.txt')
 const args = selectionArguments({ root }, { selector: '@TEST-1', source: 'browser/search.spec.ts', framework: { project: 'chromium', titlePath: ['Search', 'Find a story'] } }, filename)
 assert.deepEqual(args, ['--test-list', filename, '--no-deps'])
 assert.equal(fs.readFileSync(filename, 'utf8'), `[chromium] › ${root}/browser/search.spec.ts › Search › Find a story\n`)
 assert.throws(() => selectionArguments({ root }, { selector: '@TEST-1', source: '../escape.ts', framework: { project: 'chromium', titlePath: ['Search'] } }, filename), /escapes/)
 assert.throws(() => selectionArguments({ root }, { selector: '@TEST-1', source: 'search.ts', framework: { project: 'chromium', titlePath: ['Search\nOther'] } }, filename), /Invalid/)
 } finally { fs.rmSync(root, { recursive: true, force: true }) }
})
