import { writeFileSync } from 'node:fs'
export default () => {
 if (!JSON.parse(process.env.TESTSTUDIO_WORK_ITEM ?? '{}').test?.id) {
  writeFileSync('/output/setup-executed', 'executed')
  throw new Error('Global setup must not run during listing')
 }
}
