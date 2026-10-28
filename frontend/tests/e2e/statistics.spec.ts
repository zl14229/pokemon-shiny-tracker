import { expect, test } from '@playwright/test'
test('M6 statistics charts use real filters, known effort and responsive SVG', async ({
  page,
  request,
}) => {
  const errors: string[] = []
  page.on('pageerror', (e) => errors.push(e.message))
  const species = (await (await request.get('/api/v1/reference/species?limit=1')).json()).data[0]
  const method = `m6-statistics-${Date.now()}`
  const input = {
    species_id: species.id,
    game_id: 1,
    method,
    count_unit: 'encounters',
    encounter_count: 120,
    duration_seconds: 600,
    start_date: '2024-02-01',
    status: 'in_progress',
  }
  const create = await request.post('/api/v1/hunts', { data: input })
  expect(create.status()).toBe(201)
  const h = (await create.json()).data
  for (let i = 0; i < 2; i++) {
    const r = await request.post('/api/v1/shiny-records', {
      data: {
        hunt_id: h.id,
        original_species_id: species.id,
        current_species_id: species.id,
        original_game_id: 1,
        current_game_id: 2,
        source: 'self_hunted',
        holding_status: i ? 'released' : 'held',
        obtained_date: '2024-02-02',
      },
    })
    expect(r.status()).toBe(201)
  }
  expect(
    (
      await request.put(`/api/v1/hunts/${h.id}`, { data: { ...input, status: 'completed' } })
    ).status(),
  ).toBe(200)
  await page.goto('/#/statistics')
  await expect(page.getByTestId('statistics-totals')).toBeVisible()
  await page.getByLabel('统计方法', { exact: true }).fill(method)
  await page.getByLabel('统计计数单位', { exact: true }).fill('encounters')
  await page.getByRole('button', { name: '应用统计筛选' }).click()
  await expect(page.getByTestId('statistics-totals')).toHaveText('1 个Hunt · 2 只获得个体')
  await expect(page.locator('.statistics-chart svg')).toHaveCount(7)
  await expect(page.locator('tbody')).toContainText('60')
  await expect(page.locator('tbody')).toContainText('300')
  for (const width of [1280, 390]) {
    await page.setViewportSize({ width, height: 820 })
    await expect
      .poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
      .toBe(true)
    await expect(page.locator('.statistics-chart').first()).toBeVisible()
  }
  await page.getByLabel('统计起始日期', { exact: true }).fill('2024-02-01')
  await page.getByLabel('统计结束日期', { exact: true }).fill('2024-02-29')
  await page.getByRole('button', { name: '应用统计筛选' }).click()
  await expect(page.getByTestId('statistics-totals')).toHaveText('1 个Hunt · 2 只获得个体')
  await expect(page.locator('tbody')).toContainText('未知／不可计算')
  await page.route('**/api/v1/statistics**', (route) => route.abort())
  await page.getByRole('button', { name: '应用统计筛选' }).click()
  await expect(page.locator('.statistics-grid')).toHaveCount(0)
  await expect(page.getByRole('button', { name: '重试', exact: true })).toBeVisible()
  expect(errors).toEqual([])
})
