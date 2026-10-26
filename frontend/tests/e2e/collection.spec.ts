import { expect, test, type Page } from '@playwright/test'
async function choose(page: Page, label: string, index = 0) {
  const combo = page.getByRole('combobox', { name: label, exact: true })
  await expect(combo).toBeEnabled()
  await combo.focus()
  await combo.press('ArrowDown')
  await page
    .getByRole('listbox', { name: label, exact: true })
    .getByRole('option')
    .nth(index)
    .click()
  await expect(page.getByRole('listbox', { name: label, exact: true })).toBeHidden()
}
async function namedOption(page: Page, label: string, name: string) {
  const combo = page.getByRole('combobox', { name: label, exact: true })
  await expect(combo).toBeEnabled()
  await combo.focus()
  await combo.press('ArrowDown')
  await page
    .getByRole('listbox', { name: label, exact: true })
    .getByRole('option', { name, exact: true })
    .click()
  await expect(page.getByRole('listbox', { name: label, exact: true })).toBeHidden()
}
async function readyNew(page: Page) {
  await page.goto('/#/collection/new')
  await expect(page.getByRole('button', { name: '保存个体', exact: true })).toBeEnabled()
  await choose(page, '当前物种')
}

test('M2 end-to-end: create/edit Hunt, register associated individual, complete and edit same identity', async ({
  page,
  request,
}) => {
  const errors: string[] = []
  page.on('pageerror', (e) => errors.push(e.message))
  await page.goto('/#/hunts/new')
  await expect(page.getByRole('button', { name: '保存任务', exact: true })).toBeEnabled()
  await choose(page, '目标物种')
  await page.getByRole('textbox', { name: '任务备注', exact: true }).fill('M2 full journey')
  await page.getByRole('button', { name: '保存任务', exact: true }).click()
  await expect(page).toHaveURL(/#\/hunts\/\d+$/)
  const hid = Number(page.url().split('/').at(-1))
  await page.getByRole('link', { name: '编辑任务', exact: true }).click()
  await page.getByRole('textbox', { name: '累计次数', exact: true }).fill('5')
  await page.getByRole('textbox', { name: '计数单位', exact: true }).fill('encounters')
  await page.getByRole('button', { name: '保存任务', exact: true }).click()
  await expect(page).toHaveURL(new RegExp(`#\\/hunts\\/${hid}$`))
  await page.getByRole('button', { name: '放弃任务' }).click()
  await expect(page.getByText('已放弃', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: '重新进行' }).click()
  await expect(page.getByText('进行中', { exact: true })).toBeVisible()
  await page.getByRole('link', { name: '登记关联成果', exact: true }).click()
  await expect(page.getByRole('button', { name: '保存个体', exact: true })).toBeEnabled()
  await choose(page, '当前物种')
  await page.getByRole('textbox', { name: '昵称', exact: true }).fill('M2 linked identity')
  await page.getByRole('textbox', { name: '球种', exact: true }).fill('高级球')
  await page.getByRole('textbox', { name: '等级', exact: true }).fill('30')
  await page.getByRole('button', { name: '保存个体', exact: true }).click()
  await expect(page).toHaveURL(/#\/collection\/\d+$/)
  const sid = Number(page.url().split('/').at(-1))
  const initial = (await (await request.get(`/api/v1/shiny-records/${sid}`)).json()).data
  expect(initial).toMatchObject({
    hunt_id: hid,
    original_species_id: null,
    original_game_id: null,
    obtained_date: null,
    source: 'unknown',
    holding_status: 'unknown',
    level: 30,
    ball: '高级球',
  })
  await page.getByRole('link', { name: `查看 Hunt #${hid}`, exact: true }).click()
  await expect(page.getByText('1 个有效个体')).toBeVisible()
  await page.getByRole('button', { name: '标为完成' }).click()
  await expect(page.getByText('已完成', { exact: true })).toBeVisible()
  await page.locator(`a[href="#/collection/${sid}"]`).click()
  await page.getByRole('link', { name: '编辑个体', exact: true }).click()
  await choose(page, '当前物种', 1)
  await namedOption(page, '物种变化原因', '进化（用户记录）')
  await choose(page, '当前保存游戏', 1)
  await page.getByRole('textbox', { name: '当前保存位置', exact: true }).fill('HOME Box M2')
  await namedOption(page, '持有状态', '已交换')
  await page.getByRole('textbox', { name: '本次事件日期', exact: true }).fill('2024-02')
  await page.getByRole('textbox', { name: '变更说明', exact: true }).fill('M2 用户记录变化')
  await page.getByRole('button', { name: '保存个体', exact: true }).click()
  await expect(page).toHaveURL(new RegExp(`#\\/collection\\/${sid}$`))
  await expect(page.getByText('进化（用户记录）', { exact: true })).toBeVisible()
  await expect(page.getByText('已交换', { exact: true }).first()).toBeVisible()
  const changed = (await (await request.get(`/api/v1/shiny-records/${sid}`)).json()).data
  expect(changed).toMatchObject({
    id: sid,
    hunt_id: hid,
    original_species_id: null,
    original_game_id: null,
    current_species_id: 2,
    current_game_id: 2,
    current_location: 'HOME Box M2',
    ball: '高级球',
    level: 30,
  })
  expect(
    (await (await request.get('/api/v1/shiny-records?q=M2+linked+identity')).json()).meta.total,
  ).toBe(1)
  expect(errors).toEqual([])
})

test('independent historical individual, full edit, combined filters and responsive details', async ({
  page,
  request,
}) => {
  const errors: string[] = []
  page.on('pageerror', (e) => errors.push(e.message))
  await readyNew(page)
  await page.getByRole('textbox', { name: '昵称', exact: true }).fill('M2 historical individual')
  await choose(page, '原始获得游戏')
  await namedOption(page, '来源', '交换')
  await namedOption(page, '持有状态', '持有中')
  await page.getByRole('textbox', { name: '获得日期', exact: true }).fill('2018')
  await page.getByRole('textbox', { name: '球种', exact: true }).fill('精灵球')
  await page.getByRole('textbox', { name: '性格', exact: true }).fill('原始性格记录')
  await page.getByRole('textbox', { name: '等级', exact: true }).fill('42')
  await page.getByRole('button', { name: '保存个体', exact: true }).click()
  await expect(page).toHaveURL(/#\/collection\/\d+$/)
  const id = Number(page.url().split('/').at(-1))
  await expect(page.getByText('未关联 Hunt', { exact: true })).toBeVisible()
  await expect(page.getByText('精灵球', { exact: true })).toBeVisible()
  await page.getByRole('link', { name: '编辑个体', exact: true }).click()
  await choose(page, '当前保存游戏', 1)
  await namedOption(page, '持有状态', '已放生')
  await page.getByRole('textbox', { name: '当前保存位置', exact: true }).fill('盒子02')
  await page.getByRole('textbox', { name: '个体备注', exact: true }).fill('M2 历史资料保留')
  await page.getByRole('button', { name: '保存个体', exact: true }).click()
  await expect(page).toHaveURL(new RegExp(`#\\/collection\\/${id}$`))
  const saved = (await (await request.get(`/api/v1/shiny-records/${id}`)).json()).data
  expect(saved).toMatchObject({
    hunt_id: null,
    original_species_id: null,
    original_game_id: 1,
    current_game_id: 2,
    obtained_date: '2018',
    ball: '精灵球',
    level: 42,
    nature: '原始性格记录',
    source: 'trade',
    holding_status: 'released',
  })
  await page.goto(
    '/#/collection?q=M2+historical+individual&holding_status=released&source=trade&has_hunt=false&original_game_id=1&current_game_id=2&page_size=20',
  )
  const region = page.getByRole('region', { name: '闪光收藏列表' })
  await expect(region.getByText('M2 historical individual', { exact: false })).toBeVisible()
  await expect(region.locator('tbody').getByText('已放生', { exact: true })).toBeVisible()
  await page.screenshot({
    path: 'test-results/collection-desktop.png',
    fullPage: true,
    animations: 'disabled',
  })
  await page.setViewportSize({ width: 390, height: 844 })
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
  await page.screenshot({
    path: 'test-results/collection-mobile.png',
    fullPage: true,
    animations: 'disabled',
  })
  await page.goto(`/#/collection/${id}`)
  await expect(page.getByText('精灵球', { exact: true })).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
  expect(errors).toEqual([])
})

test('collection pagination uses real totals and stable cross-page results', async ({
  page,
  request,
}) => {
  for (let i = 0; i < 21; i++) {
    const r = await request.post('/api/v1/shiny-records', {
      data: { current_species_id: 1, nickname: `M2 page ${i}`, notes: 'M2 pagination fixture' },
    })
    expect(r.status()).toBe(201)
  }
  await page.goto('/#/collection?q=M2+pagination+fixture&page_size=20&sort=id&order=asc')
  const region = page.getByRole('region', { name: '闪光收藏列表' })
  await expect(region.getByText('共 21 条', { exact: true })).toBeVisible()
  await expect(region.locator('tbody tr')).toHaveCount(20)
  const firstPage = await region
    .locator('tbody a.record-link')
    .evaluateAll((links) => links.map((a) => a.getAttribute('href')))
  await region.getByRole('button', { name: '下一页', exact: true }).click()
  await expect(page).toHaveURL(/page=2/)
  await expect(region.locator('tbody tr')).toHaveCount(1)
  const last = await region.locator('tbody a.record-link').getAttribute('href')
  expect(firstPage).not.toContain(last)
  await expect(region.getByText('共 21 条', { exact: true })).toBeVisible()
  await region.getByRole('button', { name: '上一页', exact: true }).click()
  await expect(region.locator('tbody tr')).toHaveCount(20)
})

test('invalid collection input does not write and retains the draft', async ({ page }) => {
  let writes = 0
  page.on('request', (r) => {
    if (r.method() === 'POST' && r.url().endsWith('/shiny-records')) writes++
  })
  await page.goto('/#/collection/new')
  await expect(page.getByRole('button', { name: '保存个体', exact: true })).toBeEnabled()
  await page.getByRole('textbox', { name: '等级', exact: true }).fill('101')
  await page.getByRole('textbox', { name: '获得日期', exact: true }).fill('2023-02-29')
  await page.getByRole('textbox', { name: '个体备注', exact: true }).fill('未保存草稿')
  await page.getByRole('button', { name: '保存个体', exact: true }).click()
  await expect(page.getByText('请选择当前物种。', { exact: true })).toBeVisible()
  await expect(page.getByText('请输入有效的年、年月或日期', { exact: false })).toBeVisible()
  await expect(page.getByRole('textbox', { name: '个体备注', exact: true })).toHaveValue(
    '未保存草稿',
  )
  expect(writes).toBe(0)
})

test('last outcome detach conflict is visible and original association stays intact', async ({
  page,
  request,
}) => {
  const h = (await (await request.post('/api/v1/hunts', { data: { species_id: 1 } })).json()).data
  const s = (
    await (
      await request.post('/api/v1/shiny-records', {
        data: { current_species_id: 1, hunt_id: h.id },
      })
    ).json()
  ).data
  const update = await request.put(`/api/v1/hunts/${h.id}`, {
    data: {
      ...h,
      id: undefined,
      created_at: undefined,
      updated_at: undefined,
      status: 'completed',
    },
  })
  expect(update.status()).toBe(200)
  await page.goto(`/#/collection/${s.id}/edit`)
  await page.getByRole('button', { name: '清空关联', exact: true }).click()
  await page.getByRole('button', { name: '保存个体', exact: true }).click()
  await expect(page.getByText('无法移走已完成任务的最后有效成果。', { exact: false })).toBeVisible()
  expect((await (await request.get(`/api/v1/shiny-records/${s.id}`)).json()).data.hunt_id).toBe(
    h.id,
  )
})

test('collection list failure is redacted and retry recovers', async ({ page }) => {
  await page.route('**/api/v1/shiny-records?**', (r) =>
    r.fulfill({
      status: 500,
      json: { error: { code: 'internal_error', message: 'private-collection-error' } },
    }),
  )
  await page.goto('/#/collection')
  await expect(page.getByText('本地服务暂时不可用，请稍后重试。', { exact: true })).toBeVisible()
  await expect(page.locator('body')).not.toContainText('private-collection-error')
  await page.unroute('**/api/v1/shiny-records?**')
  await page.getByRole('button', { name: '重试', exact: true }).click()
  await expect(page.getByText('本地服务暂时不可用，请稍后重试。', { exact: true })).toHaveCount(0)
})

test('ambiguous individual write is not repeated after a committed result', async ({
  page,
  request,
}) => {
  let writes = 0
  await page.route('**/api/v1/shiny-records', async (r) => {
    if (r.request().method() !== 'POST') return r.continue()
    writes++
    await r.fetch()
    await r.fulfill({ status: 502, contentType: 'text/html', body: 'private-write-detail' })
  })
  await readyNew(page)
  await page.getByRole('textbox', { name: '昵称', exact: true }).fill('M2 ambiguous individual')
  await page.getByRole('button', { name: '保存个体', exact: true }).click()
  await expect(page.getByText('保存结果暂未确认。', { exact: false })).toBeVisible()
  await expect(page.getByRole('button', { name: '保存个体', exact: true })).toBeDisabled()
  await expect(page.locator('body')).not.toContainText('private-write-detail')
  expect(writes).toBe(1)
  expect(
    (await (await request.get('/api/v1/shiny-records?q=M2+ambiguous+individual')).json()).meta
      .total,
  ).toBe(1)
})
