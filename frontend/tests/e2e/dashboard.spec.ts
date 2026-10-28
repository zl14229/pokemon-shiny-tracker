import { expect, test } from '@playwright/test'
test('M6 dashboard matches the real snapshot and M4 dex on desktop and mobile', async ({
  page,
  request,
}) => {
  const response = await request.get('/api/v1/dashboard')
  expect(response.ok()).toBe(true)
  const v = (await response.json()).data
  for (const width of [1280, 390]) {
    await page.setViewportSize({ width, height: 820 })
    await page.goto('/#/home')
    await expect(page.getByTestId('historical-individuals')).toHaveText(
      v.collection.historical_individuals.toLocaleString('zh-CN'),
    )
    await expect(page.getByTestId('held-individuals')).toHaveText(
      v.collection.held_individuals.toLocaleString('zh-CN'),
    )
    await expect(page.getByRole('region', { name: '收藏总览' })).toBeVisible()
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
  }
  for (const d of v.dex) {
    const r = await request.get(`/api/v1/dex?dimension=${d.dimension}&basis=${d.basis}`)
    expect((await r.json()).summary).toEqual(d.summary)
  }
  await page.route('**/api/v1/dashboard**', (route) => route.abort())
  await page.getByRole('button', { name: '刷新总览' }).click()
  await expect(page.getByTestId('historical-individuals')).toHaveCount(0)
  await expect(page.getByRole('button', { name: '重试', exact: true })).toBeVisible()
})
