import assert from 'node:assert/strict'
import { createServer, preview } from 'vite'

function address(server) {
  const bound = server.address()
  assert(bound && typeof bound !== 'string')
  return `http://127.0.0.1:${bound.port}`
}

const dev = await createServer({ server: { host: '127.0.0.1', port: 0 } })
try {
  await dev.listen()
  const base = address(dev.httpServer)
  const response = await fetch(base)
  assert.equal(response.status, 200)
  assert.match(await response.text(), /id="app"/)
  const module = await fetch(`${base}/src/App.vue`)
  assert.equal(module.status, 200)
  assert.match(await module.text(), /Pokémon Shiny Tracker/)
} finally {
  await dev.close()
}

const built = await preview({ preview: { host: '127.0.0.1', port: 0 } })
try {
  const base = address(built.httpServer)
  const response = await fetch(base)
  assert.equal(response.status, 200)
  const html = await response.text()
  const asset = html.match(/src="(\/assets\/[^\"]+\.js)"/)
  assert(asset, 'production HTML must reference built JavaScript')
  const javascript = await fetch(`${base}${asset[1]}`)
  assert.equal(javascript.status, 200)
  assert((await javascript.text()).length > 0)
} finally {
  await new Promise((resolve, reject) => built.httpServer.close((error) => error ? reject(error) : resolve()))
}
console.log('PASS: loopback Vite startup, Vue transform, production preview and built JavaScript')
