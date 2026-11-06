const SECTOR = 512
const FREE = 0xffffffff
const END = 0xfffffffe
const FATSECT = 0xfffffffd

function record(id, data) {
  const head = Buffer.alloc(4)
  head.writeUInt16LE(id, 0)
  head.writeUInt16LE(data.length, 2)
  return Buffer.concat([head, data])
}

function bof(type) {
  const d = Buffer.alloc(16)
  d.writeUInt16LE(0x0600, 0)
  d.writeUInt16LE(type, 2)
  d.writeUInt16LE(0x0dbb, 4)
  d.writeUInt16LE(0x07cc, 6)
  d.writeUInt32LE(0x6, 12)
  return record(0x0809, d)
}

function label(row, col, text) {
  const chars = Buffer.from(String(text), 'utf16le')
  const d = Buffer.alloc(9)
  d.writeUInt16LE(row, 0)
  d.writeUInt16LE(col, 2)
  d.writeUInt16LE(0x0f, 4)
  d.writeUInt16LE(String(text).length, 6)
  d.writeUInt8(1, 8)
  return record(0x0204, Buffer.concat([d, chars]))
}

function number(row, col, value) {
  const d = Buffer.alloc(14)
  d.writeUInt16LE(row, 0)
  d.writeUInt16LE(col, 2)
  d.writeUInt16LE(0x0f, 4)
  d.writeDoubleLE(value, 6)
  return record(0x0203, d)
}

function workbookStream(sheetName, rows) {
  const name = Buffer.from(sheetName, 'utf16le')
  const sheetHead = Buffer.alloc(8)
  sheetHead.writeUInt8(sheetName.length, 6)
  sheetHead.writeUInt8(1, 7)
  const boundSize = 4 + sheetHead.length + name.length
  const globalsSize = bof(0x0005).length + boundSize + 4
  sheetHead.writeUInt32LE(globalsSize, 0)
  const globals = Buffer.concat([bof(0x0005), record(0x0085, Buffer.concat([sheetHead, name])), record(0x000a, Buffer.alloc(0))])
  const cells = []
  rows.forEach((r, ri) =>
    r.forEach((v, ci) => {
      if (v === '' || v === null || v === undefined) return
      cells.push(typeof v === 'number' ? number(ri, ci, v) : label(ri, ci, v))
    }),
  )
  const sheet = Buffer.concat([bof(0x0010), ...cells, record(0x000a, Buffer.alloc(0))])
  let stream = Buffer.concat([globals, sheet])
  const size = Math.max(4096, Math.ceil(stream.length / SECTOR) * SECTOR)
  stream = Buffer.concat([stream, Buffer.alloc(size - stream.length)])
  return stream
}

function dirEntry({ name = '', type = 0, child = FREE, start = 0, size = 0 }) {
  const e = Buffer.alloc(128)
  if (name) {
    Buffer.from(name, 'utf16le').copy(e, 0)
    e.writeUInt16LE((name.length + 1) * 2, 0x40)
  }
  e.writeUInt8(type, 0x42)
  e.writeUInt8(type ? 1 : 0, 0x43)
  e.writeUInt32LE(FREE, 0x44)
  e.writeUInt32LE(FREE, 0x48)
  e.writeUInt32LE(child, 0x4c)
  e.writeUInt32LE(start, 0x74)
  e.writeUInt32LE(size, 0x78)
  return e
}

export function buildXLS(rows, sheetName = 'Sheet1') {
  const stream = workbookStream(sheetName, rows)
  const n = stream.length / SECTOR
  if (n + 2 > SECTOR / 4) throw new Error('xls fixture quá lớn')
  const header = Buffer.alloc(SECTOR)
  Buffer.from([0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1]).copy(header, 0)
  header.writeUInt16LE(0x003e, 0x18)
  header.writeUInt16LE(0x0003, 0x1a)
  header.writeUInt16LE(0xfffe, 0x1c)
  header.writeUInt16LE(9, 0x1e)
  header.writeUInt16LE(6, 0x20)
  header.writeUInt32LE(1, 0x2c)
  header.writeUInt32LE(1, 0x30)
  header.writeUInt32LE(4096, 0x38)
  header.writeUInt32LE(END, 0x3c)
  header.writeUInt32LE(END, 0x44)
  for (let i = 0; i < 109; i++) header.writeUInt32LE(i === 0 ? 0 : FREE, 0x4c + i * 4)
  const fat = Buffer.alloc(SECTOR, 0xff)
  fat.writeUInt32LE(FATSECT, 0)
  fat.writeUInt32LE(END, 4)
  for (let i = 0; i < n; i++) fat.writeUInt32LE(i === n - 1 ? END : 3 + i, (2 + i) * 4)
  const dir = Buffer.concat([
    dirEntry({ name: 'Root Entry', type: 5, child: 1, start: END }),
    dirEntry({ name: 'Workbook', type: 2, start: 2, size: stream.length }),
    dirEntry({}),
    dirEntry({}),
  ])
  return Buffer.concat([header, fat, dir, stream])
}
