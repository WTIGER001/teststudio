import fs from 'node:fs'
import path from 'node:path'

function crc32(data) {
  let crc = 0xffffffff
  for (const value of data) { crc ^= value; for (let bit = 0; bit < 8; bit++) crc = crc & 1 ? 0xedb88320 ^ crc >>> 1 : crc >>> 1 }
  return (crc ^ 0xffffffff) >>> 0
}
export function sourceZIP(root) {
  const entries = []
  function collect(directory) {
    for (const name of fs.readdirSync(directory).sort()) {
      const filename = path.join(directory, name), stat = fs.lstatSync(filename)
      if (stat.isSymbolicLink()) throw new Error('Demo source cannot contain symbolic links')
      if (stat.isDirectory()) collect(filename)
      else if (stat.isFile()) entries.push({ name: Buffer.from(path.relative(root, filename).split(path.sep).join('/')), data: fs.readFileSync(filename) })
      else throw new Error('Demo source requires regular files')
    }
  }
  collect(root)
  const files = [], directory = []
  let offset = 0
  for (const entry of entries) {
    const crc = crc32(entry.data), header = Buffer.alloc(30), index = Buffer.alloc(46)
    header.writeUInt32LE(0x04034b50); header.writeUInt16LE(20, 4); header.writeUInt32LE(crc, 14); header.writeUInt32LE(entry.data.length, 18); header.writeUInt32LE(entry.data.length, 22); header.writeUInt16LE(entry.name.length, 26)
    index.writeUInt32LE(0x02014b50); index.writeUInt16LE(20, 4); index.writeUInt16LE(20, 6); index.writeUInt32LE(crc, 16); index.writeUInt32LE(entry.data.length, 20); index.writeUInt32LE(entry.data.length, 24); index.writeUInt16LE(entry.name.length, 28); index.writeUInt32LE(offset, 42)
    files.push(header, entry.name, entry.data); directory.push(index, entry.name); offset += header.length + entry.name.length + entry.data.length
  }
  const central = Buffer.concat(directory), end = Buffer.alloc(22)
  end.writeUInt32LE(0x06054b50); end.writeUInt16LE(entries.length, 8); end.writeUInt16LE(entries.length, 10); end.writeUInt32LE(central.length, 12); end.writeUInt32LE(offset, 16)
  return Buffer.concat([...files, central, end])
}
