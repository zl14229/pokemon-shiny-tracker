import { expect, test } from '@playwright/test'
test('M5 Random Hunt saves a reviewed snapshot, converts one Hunt and reloads offline', async ({
  page,
  request,
}) => {
  const errors: string[] = []
  page.on('pageerror', (e) => errors.push(e.message))
  const before = (await (await request.get('/api/v1/hunts')).json()).meta.total
  await page.goto('/#/planner')
  await page.getByRole('checkbox', { name: '朱', exact: true }).check()
  await page.getByRole('checkbox', { name: '紫', exact: true }).check()
  await page.getByRole('spinbutton', { name: '目标数量' }).fill('2')
  await page.getByRole('button', { name: '生成随机目标', exact: true }).click()
  await expect(page.getByRole('article')).toHaveCount(2)
  const names = await page
    .getByRole('article')
    .evaluateAll((nodes) => nodes.map((n) => n.getAttribute('aria-label')))
  expect(new Set(names).size).toBe(2)
  expect((await (await request.get('/api/v1/hunts')).json()).meta.total).toBe(before)
  await page.getByRole('article').first().getByText('方法、条件与来源', { exact: true }).click()
  await expect(page.getByRole('article').first()).toContainText('遭遇率：未知')
  await expect(page.getByRole('article').first()).toContainText('1/4096')
  await page.getByRole('button', { name: '保存本次结果', exact: true }).click()
  await expect(page).toHaveURL(/#\/planner\/plans\/\d+$/)
  const planId = Number(page.url().split('/').at(-1))
  await expect(page.getByRole('button', { name: '转为 Hunt', exact: true })).toHaveCount(2)
  await page.getByRole('button', { name: '转为 Hunt', exact: true }).first().click()
  await expect(page.getByRole('link', { name: /查看 Hunt #/ })).toHaveCount(1)
  expect((await (await request.get('/api/v1/hunts')).json()).meta.total).toBe(before + 1)
  await page.route('https://**', (r) => r.abort())
  await page.reload()
  await expect(page.getByRole('article')).toHaveCount(2)
  await expect(page.getByRole('link', { name: /查看 Hunt #/ })).toHaveCount(1)
  const saved = (await (await request.get(`/api/v1/planner/plans/${planId}`)).json()).data
  const target = saved.targets.find((t: { hunt_id: number | null }) => t.hunt_id !== null)
  expect(
    (
      await request.post(`/api/v1/planner/plans/${planId}/targets/${target.id}/hunt`, { data: {} })
    ).status(),
  ).toBe(200)
  expect((await (await request.get('/api/v1/hunts')).json()).meta.total).toBe(before + 1)
  await page.setViewportSize({ width: 390, height: 844 })
  await expect
    .poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
    .toBe(true)
  expect(errors).toEqual([])
})
test('M5 empty and insufficient candidates stay honest; source failure clears stale results', async ({
  page,
  request,
}) => {
  await page.goto('/#/planner')
  await page.getByRole('checkbox', { name: '紅', exact: true }).check()
  await page.getByRole('spinbutton', { name: '目标数量' }).fill('25')
  await page.getByRole('button', { name: '生成随机目标', exact: true }).click()
  await expect(
    page.getByText('当前没有符合条件的已核实目标。请调整版本或等待资料核验；不会自动联网补全。'),
  ).toBeVisible()
  await expect(page.getByRole('button', { name: '保存本次结果', exact: true })).toBeDisabled()
  await expect(page.getByRole('article')).toHaveCount(0)
  await page.getByRole('checkbox', { name: '紅', exact: true }).uncheck()
  await page.getByRole('checkbox', { name: '朱', exact: true }).check()
  await page.getByRole('checkbox', { name: '紫', exact: true }).check()
  await page.getByRole('spinbutton', { name: '目标数量' }).fill('100')
  await page.getByRole('combobox', { name: '抽取维度', exact: true }).selectOption('form')
  const pool = (
    await (
      await request.post('/api/v1/planner/pool', {
        data: { rules: { game_ids: [3, 4], dimension: 'form', count: 100 } },
      })
    ).json()
  ).data
  await page.getByRole('button', { name: '生成随机目标', exact: true }).click()
  await expect(page.getByRole('article')).toHaveCount(pool.coverage.actual)
  await expect(page.getByText(/候选数量不足：请求 100 个/)).toBeVisible()
  await page.route('**/api/v1/planner/random', (r) =>
    r.fulfill({
      status: 503,
      contentType: 'application/json',
      body: JSON.stringify({
        error: { code: 'reference_unavailable', message: 'secret must not render' },
      }),
    }),
  )
  await page.getByRole('button', { name: '生成随机目标', exact: true }).click()
  await expect(page.getByRole('alert')).toContainText('不会补入未知目标')
  await expect(page.getByText('secret must not render')).toHaveCount(0)
  await expect(page.getByRole('article')).toHaveCount(0)
})
