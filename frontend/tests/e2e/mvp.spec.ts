import { expect, test, type Page } from '@playwright/test'
import { execFileSync, spawn, type ChildProcess } from 'node:child_process'
import { mkdtemp, readFile, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { createServer, type AddressInfo } from 'node:net'
import { stripVTControlCharacters } from 'node:util'

async function select(page: Page, label: string, option?: string) {
  const input = page.getByRole('combobox', { name: label, exact: true })
  await expect(input).toBeEnabled()
  await input.focus()
  await input.press('ArrowDown')
  const options = page.getByRole('listbox', { name: label, exact: true }).getByRole('option')
  await (option ? options.filter({ hasText: new RegExp(`^${option}$`) }) : options.first()).click()
  await expect(page.getByRole('listbox', { name: label, exact: true })).toBeHidden()
}
async function freePort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const server = createServer()
    server.once('error', reject)
    server.listen(0, '127.0.0.1', () => {
      const port = (server.address() as AddressInfo).port
      server.close((e) => (e ? reject(e) : resolve(port)))
    })
  })
}
async function start(
  command: string,
  args: string[],
  env: NodeJS.ProcessEnv,
  url: string,
  marker: string,
  owned: ChildProcess[],
) {
  const child = spawn(command, args, { env, stdio: ['ignore', 'pipe', 'pipe'] })
  owned.push(child)
  let logs = ''
  child.stdout?.on('data', (b) => {
    logs = (logs + String(b)).slice(-10000)
  })
  child.stderr?.on('data', (b) => {
    logs = (logs + String(b)).slice(-10000)
  })
  const deadline = Date.now() + 12000
  while (Date.now() < deadline) {
    if (child.exitCode !== null) throw new Error('Owned recovery process exited during startup')
    // A marker from our own process proves it bound the chosen port, before any
    // data requests are sent. Never reuse or terminate an existing service.
    if (stripVTControlCharacters(logs).includes(marker)) {
      try {
        if ((await fetch(url)).ok) return
      } catch {
        /* startup polling */
      }
    }
    await new Promise((resolve) => setTimeout(resolve, 50))
  }
  throw new Error('Owned recovery process failed readiness')
}
async function stop(child: ChildProcess) {
  if (child.exitCode !== null) return
  await new Promise<void>((resolve) => {
    const timer = setTimeout(() => child.kill('SIGKILL'), 5000)
    const done = () => {
      clearTimeout(timer)
      resolve()
    }
    child.once('exit', done)
    if (child.exitCode !== null) done()
    else child.kill('SIGTERM')
  })
}

test('MVP full user journey survives verified backup and isolated restored UI', async ({
  page,
  request,
}) => {
  test.setTimeout(60000)
  const errors: string[] = []
  page.on('pageerror', (e) => errors.push(e.message))
  const work = await mkdtemp(join(tmpdir(), 'pst-mvp-acceptance-'))
  const owned: ChildProcess[] = []
  try {
    await page.goto('/#/hunts/new')
    await select(page, '目标物种')
    await page.getByRole('textbox', { name: '累计次数', exact: true }).fill('0')
    await page.getByRole('textbox', { name: '计数单位', exact: true }).fill('eggs')
    await page.getByRole('textbox', { name: '开始日期', exact: true }).fill('2024')
    await page.getByRole('textbox', { name: '任务备注', exact: true }).fill('M3 full user journey')
    await page.getByRole('button', { name: '保存任务', exact: true }).click()
    await expect(page).toHaveURL(/#\/hunts\/\d+$/)
    const huntID = Number(page.url().split('/').at(-1))
    await page.getByRole('link', { name: '编辑任务', exact: true }).click()
    await page.getByRole('textbox', { name: '累计次数', exact: true }).fill('12')
    await page.getByRole('textbox', { name: '累计耗时（秒）', exact: true }).fill('90')
    await page.getByRole('button', { name: '保存任务', exact: true }).click()
    await expect(page).toHaveURL(new RegExp(`#\\/hunts\\/${huntID}$`))
    await page.getByRole('link', { name: '登记关联成果', exact: true }).click()
    await select(page, '当前物种')
    await page.getByRole('textbox', { name: '昵称', exact: true }).fill('M3 联调个体')
    await select(page, '持有状态', '持有中')
    await select(page, '来源', '自刷')
    await page.getByRole('textbox', { name: '获得日期', exact: true }).fill('2024-02')
    await page.getByRole('textbox', { name: '球种', exact: true }).fill('高级球')
    await page.getByRole('button', { name: '保存个体', exact: true }).click()
    await expect(page).toHaveURL(/#\/collection\/\d+$/)
    const shinyID = Number(page.url().split('/').at(-1))
    await page.goto(`/#/hunts/${huntID}`)
    await expect(page.getByRole('button', { name: '标为完成', exact: true })).toBeEnabled()
    await page.getByRole('button', { name: '标为完成', exact: true }).click()
    await expect(page.getByText('已完成', { exact: true })).toBeVisible()
    await page.goto(`/#/collection/${shinyID}/edit`)
    await expect(page.getByRole('textbox', { name: '昵称', exact: true })).toHaveValue(
      'M3 联调个体',
    )
    await page.getByRole('textbox', { name: '当前保存位置', exact: true }).fill('Pokémon HOME')
    await page
      .getByRole('textbox', { name: '个体备注', exact: true })
      .fill('M3 collection edited before backup')
    await page.getByRole('button', { name: '保存个体', exact: true }).click()
    await expect(page).toHaveURL(new RegExp(`#\\/collection\\/${shinyID}$`))
    const primary = (await (await request.get(`/api/v1/shiny-records/${shinyID}`)).json()).data
    expect(primary).toMatchObject({
      hunt_id: huntID,
      source: 'self_hunted',
      holding_status: 'held',
      obtained_date: '2024-02',
      original_species_id: null,
      original_game_id: null,
      current_location: 'Pokémon HOME',
    })

    const count = (await (await request.get('/api/v1/shiny-records')).json()).meta.total
    await page.goto('/#/data/import')
    await page.getByLabel('选择 Excel 文件', { exact: true }).setInputFiles({
      name: 'anonymous.xlsx',
      mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
      buffer: execFileSync('python3', ['../scripts/anonymous_workbook.py']),
    })
    for (const [col, field] of [
      ['A', '当前物种全国编号'],
      ['B', '获得日期（ISO）'],
      ['C', '球种'],
      ['D', '备注（多列拼接原值）'],
    ])
      await select(page, `映射列${col}`, field)
    await page.getByRole('button', { name: '生成只读预览', exact: true }).click()
    await expect(page.getByRole('heading', { name: '预览与逐行校验', exact: true })).toBeVisible()
    expect((await (await request.get('/api/v1/shiny-records')).json()).meta.total).toBe(count)
    await expect(page.getByRole('checkbox', { name: '选择第3行', exact: true })).toBeDisabled()
    await page.getByRole('checkbox', { name: '选择第2行', exact: true }).press('Space')
    await page.getByRole('checkbox', { name: /我已核对选中行/ }).press('Space')
    await page.getByRole('button', { name: '确认导入选中个体', exact: true }).click()
    await expect(page.getByRole('heading', { name: '导入成功', exact: true })).toBeVisible()
    const batch = (await (await request.get('/api/v1/imports/batches')).json()).data[0]
    const trace = (await (await request.get(`/api/v1/imports/batches/${batch.id}`)).json()).data
    const legacyID = trace.rows[0].shiny_record_id
    expect(
      (await (await request.get(`/api/v1/shiny-records/${legacyID}`)).json()).data,
    ).toMatchObject({
      hunt_id: null,
      source: 'unknown',
      original_species_id: null,
      obtained_date: '2020-02',
    })

    await page.goto(`/#/collection/${shinyID}`)
    const panel = page.getByRole('region', { name: '附件与链接', exact: true })
    await expect(panel.getByRole('button', { name: '上传本地附件', exact: true })).toBeEnabled()
    await panel.getByLabel('上传附件文件', { exact: true }).setInputFiles({
      name: 'mvp-evidence.txt',
      mimeType: 'text/plain',
      buffer: Buffer.from('MVP anonymous exact evidence bytes'),
    })
    await expect(panel.getByText('mvp-evidence.txt', { exact: true })).toBeVisible()
    await panel.getByRole('textbox', { name: '附件链接名称', exact: true }).fill('MVP proof link')
    await panel
      .getByRole('textbox', { name: '附件链接地址', exact: true })
      .fill('https://example.com/mvp-proof')
    await panel.getByRole('button', { name: '关联外部链接', exact: true }).click()
    await expect(panel.getByText('MVP proof link', { exact: true })).toBeVisible()
    const attachments = (
      await (await request.get(`/api/v1/shiny-records/${shinyID}/attachments`)).json()
    ).data
    const fileID = attachments.find((a: { kind: string }) => a.kind === 'file').id

    await page.goto('/#/data/backup')
    const [jsonDownload] = await Promise.all([
      page.waitForEvent('download'),
      page.getByRole('button', { name: '下载 JSON 数据', exact: true }).click(),
    ])
    const jsonPath = join(work, 'data.json')
    await jsonDownload.saveAs(jsonPath)
    const exported = JSON.parse(await readFile(jsonPath, 'utf8'))
    expect(exported.tables.shiny_records.find((r: { id: number }) => r.id === shinyID).notes).toBe(
      'M3 collection edited before backup',
    )
    const [zipDownload] = await Promise.all([
      page.waitForEvent('download'),
      page.getByRole('button', { name: '下载 ZIP 备份', exact: true }).click(),
    ])
    const archive = join(work, 'backup.zip')
    await zipDownload.saveAs(archive)
    expect(
      execFileSync('../bin/shiny-data', ['verify', '--archive', archive], { encoding: 'utf8' }),
    ).toContain('PASS')
    const restored = join(work, 'restored')
    expect(
      execFileSync(
        '../bin/shiny-data',
        ['restore', '--archive', archive, '--destination', restored],
        { encoding: 'utf8' },
      ),
    ).toContain('PASS')
    const backendPort = await freePort()
    await start(
      '../bin/shiny-tracker',
      [],
      {
        ...process.env,
        SHINY_DB_PATH: join(restored, 'database.sqlite3'),
        SHINY_HTTP_PORT: String(backendPort),
      },
      `http://127.0.0.1:${backendPort}/api/v1/health`,
      `server listening address=127.0.0.1:${backendPort}`,
      owned,
    )
    const backend = `http://127.0.0.1:${backendPort}/api/v1`
    const restoredHunt = (await (await request.get(`${backend}/hunts/${huntID}`)).json()).data
    expect(restoredHunt).toMatchObject({
      status: 'completed',
      encounter_count: 12,
      count_unit: 'eggs',
      duration_seconds: 90,
      end_date: null,
    })
    expect((await (await request.get(`${backend}/shiny-records/${shinyID}`)).json()).data).toEqual(
      primary,
    )
    expect(
      (await (await request.get(`${backend}/shiny-records/${legacyID}`)).json()).data.hunt_id,
    ).toBeNull()
    expect(
      (await (await request.get(`${backend}/shiny-records/${shinyID}/history`)).json()).data.some(
        (h: { kind: string }) => h.kind === 'location_change',
      ),
    ).toBe(true)
    expect(
      (await (await request.get(`${backend}/imports/batches/${batch.id}`)).json()).data.rows[0]
        .shiny_record_id,
    ).toBe(legacyID)
    expect(await (await request.get(`${backend}/attachments/${fileID}/content`)).text()).toBe(
      'MVP anonymous exact evidence bytes',
    )
    const vitePort = await freePort()
    const base = `http://127.0.0.1:${vitePort}`
    // Exercise CI's colored output even if the parent shell sets NO_COLOR.
    const viteEnv: NodeJS.ProcessEnv = {
      ...process.env,
      SHINY_API_PORT: String(backendPort),
      FORCE_COLOR: '1',
    }
    delete viteEnv.NO_COLOR
    await start(
      process.execPath,
      [
        'node_modules/vite/bin/vite.js',
        '--host',
        '127.0.0.1',
        '--port',
        String(vitePort),
        '--strictPort',
      ],
      viteEnv,
      base,
      `${base}/`,
      owned,
    )
    await page.goto(`${base}/#/collection/${shinyID}`)
    await expect(page.getByRole('heading', { name: 'M3 联调个体', exact: true })).toBeVisible()
    await expect(page.getByText('mvp-evidence.txt', { exact: true })).toBeVisible()
    await expect(page.getByText('MVP proof link', { exact: true })).toBeVisible()
    await page.screenshot({ path: 'test-results/mvp-restored-desktop.png', fullPage: true })
    await page.setViewportSize({ width: 390, height: 844 })
    expect(
      await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
    ).toBe(true)
    await page.screenshot({ path: 'test-results/mvp-restored-mobile.png', fullPage: true })
    expect(errors).toEqual([])
  } finally {
    for (const child of owned.reverse()) await stop(child)
    await rm(work, { recursive: true, force: true })
  }
})
