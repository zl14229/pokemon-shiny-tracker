import { expect, test, type Page } from '@playwright/test'
async function choose(page: Page, label: string) {
  await page.getByRole('combobox', { name: label, exact: true }).click()
  await page.getByRole('listbox', { name: label, exact: true }).getByRole('option').first().click()
}
async function newHunt(page: Page, notes: string) {
  await page.goto('/#/hunts/new')
  await expect(page.getByRole('button', { name: '保存任务', exact: true })).toBeEnabled()
  await choose(page, '目标物种')
  await page.getByRole('textbox', { name: '任务备注', exact: true }).fill(notes)
}

test('real Hunt create/edit/status/results and responsive list', async ({ page, request }) => {
  const errors: string[] = []
  page.on('pageerror', (e) => errors.push(e.message))
  await newHunt(page, 'T010 browser journey')
  await choose(page, '目标形态')
  await choose(page, '目标游戏')
  await page.getByRole('textbox', { name: '累计次数', exact: true }).fill('0')
  await page.getByRole('textbox', { name: '计数单位', exact: true }).fill('eggs')
  await page.getByRole('textbox', { name: '累计耗时（秒）', exact: true }).fill('60')
  await page.getByRole('textbox', { name: '开始日期', exact: true }).fill('2024')
  await page.getByRole('button', { name: '保存任务', exact: true }).click()
  await expect(page).toHaveURL(/#\/hunts\/\d+$/)
  const id = Number(page.url().split('/').at(-1))
  await expect(page.getByText('0 eggs', { exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: '标为完成' })).toBeDisabled()
  await page.getByRole('link', { name: '编辑任务', exact: true }).click()
  await expect(page.getByRole('textbox', { name: '开始日期', exact: true })).toHaveValue('2024')
  await expect(page.getByRole('textbox', { name: '累计次数', exact: true })).toHaveValue('0')
  await page.getByRole('textbox', { name: '累计次数', exact: true }).fill('12')
  await page.getByRole('textbox', { name: '累计耗时（秒）', exact: true }).fill('90')
  await page.getByRole('textbox', { name: '结束日期', exact: true }).fill('2024-02')
  await page.getByRole('button', { name: '保存任务', exact: true }).click()
  await expect(page).toHaveURL(new RegExp(`#\\/hunts\\/${id}$`))
  await expect(page.getByText('12 eggs', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: '放弃任务' }).click()
  await expect(page.getByText('已放弃', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: '重新进行' }).click()
  await expect(page.getByText('进行中', { exact: true })).toBeVisible()
  const saved = (await (await request.get(`/api/v1/hunts/${id}`)).json()).data
  expect(saved).toMatchObject({
    encounter_count: 12,
    start_date: '2024',
    end_date: '2024-02',
    game_id: 1,
    form_id: 1,
  })
  const result = await request.post('/api/v1/shiny-records', {
    data: { current_species_id: saved.species_id, hunt_id: id },
  })
  expect(result.status()).toBe(201)
  await page.getByRole('button', { name: '刷新详情' }).click()
  await expect(page.getByText('1 个有效个体')).toBeVisible()
  await page.getByRole('button', { name: '标为完成' }).click()
  await expect(page.getByText('已完成', { exact: true })).toBeVisible()
  await page.goto('/#/hunts?q=T010+browser+journey&page_size=20')
  await expect(page.getByText('12 eggs', { exact: true })).toBeVisible()
  await expect(
    page.getByRole('region', { name: '刷闪任务列表' }).getByText('已完成', { exact: true }),
  ).toBeVisible()
  await page.screenshot({
    path: 'test-results/hunt-desktop.png',
    fullPage: true,
    animations: 'disabled',
  })
  await page.setViewportSize({ width: 390, height: 844 })
  await expect(page.getByRole('heading', { name: '刷闪任务', exact: true })).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
  await page.screenshot({
    path: 'test-results/hunt-mobile.png',
    fullPage: true,
    animations: 'disabled',
  })
  expect(errors).toEqual([])
})

test('form invalid input stays visible and no write is sent', async ({ page }) => {
  let writes = 0
  page.on('request', (r) => {
    if (r.method() === 'POST' && r.url().endsWith('/hunts')) writes++
  })
  await page.goto('/#/hunts/new')
  await expect(page.getByRole('button', { name: '保存任务', exact: true })).toBeEnabled()
  await page.getByRole('textbox', { name: '累计次数', exact: true }).fill('-1')
  await page.getByRole('button', { name: '保存任务', exact: true }).click()
  await expect(page.getByText('请选择目标物种。', { exact: true })).toBeVisible()
  await expect(page.getByRole('textbox', { name: '累计次数', exact: true })).toHaveValue('-1')
  await choose(page, '目标物种')
  await page.getByRole('textbox', { name: '累计次数', exact: true }).fill('0')
  await page.getByRole('textbox', { name: '开始日期', exact: true }).fill('2023-02-29')
  await page.getByRole('button', { name: '保存任务', exact: true }).click()
  await expect(page.getByText('已知次数必须填写计数单位，例如 eggs 或 encounters。')).toBeVisible()
  await expect(page.getByText('请输入有效的年、年月或日期', { exact: false })).toBeVisible()
  expect(writes).toBe(0)
})

test('list failure is redacted and retry recovers', async ({ page }) => {
  await page.route('**/api/v1/hunts?**', (r) =>
    r.fulfill({
      status: 500,
      json: { error: { code: 'internal_error', message: 'private-detail' } },
    }),
  )
  await page.goto('/#/hunts')
  await expect(page.getByText('本地服务暂时不可用，请稍后重试。', { exact: true })).toBeVisible()
  await expect(page.locator('body')).not.toContainText('private-detail')
  await page.unroute('**/api/v1/hunts?**')
  await page.getByRole('button', { name: '重试', exact: true }).click()
  await expect(page.getByText('本地服务暂时不可用，请稍后重试。')).toHaveCount(0)
})

test('uncertain committed create cannot be submitted twice', async ({ page, request }) => {
  let writes = 0
  await page.route('**/api/v1/hunts', async (route) => {
    if (route.request().method() !== 'POST') return route.continue()
    writes++
    await route.fetch()
    await route.fulfill({ status: 502, contentType: 'text/html', body: 'private-proxy-detail' })
  })
  await newHunt(page, 'T010 ambiguous write')
  await page.getByRole('button', { name: '保存任务', exact: true }).click()
  await expect(page.getByText('保存结果暂未确认。', { exact: false })).toBeVisible()
  await expect(page.getByRole('button', { name: '保存任务', exact: true })).toBeDisabled()
  await expect(page.locator('body')).not.toContainText('private-proxy-detail')
  expect(writes).toBe(1)
  const result = await (await request.get('/api/v1/hunts?q=T010+ambiguous+write')).json()
  expect(result.meta.total).toBe(1)
})

test('stale edit is rejected before overwriting a newer record', async ({ page, request }) => {
  const created = (
    await (
      await request.post('/api/v1/hunts', { data: { species_id: 1, notes: 'T010 stale original' } })
    ).json()
  ).data
  await page.goto(`/#/hunts/${created.id}/edit`)
  await expect(page.getByRole('textbox', { name: '任务备注', exact: true })).toHaveValue(
    'T010 stale original',
  )
  const updated = await request.put(`/api/v1/hunts/${created.id}`, {
    data: {
      ...created,
      id: undefined,
      created_at: undefined,
      updated_at: undefined,
      notes: 'newer note',
    },
  })
  expect(updated.status()).toBe(200)
  await page.getByRole('textbox', { name: '任务备注', exact: true }).fill('unsaved edit')
  await page.getByRole('button', { name: '保存任务', exact: true }).click()
  await expect(page.getByText('记录已在其他页面修改，请先重新加载再编辑。')).toBeVisible()
  await expect(page.getByRole('textbox', { name: '任务备注', exact: true })).toHaveValue(
    'unsaved edit',
  )
  expect((await (await request.get(`/api/v1/hunts/${created.id}`)).json()).data.notes).toBe(
    'newer note',
  )
})
