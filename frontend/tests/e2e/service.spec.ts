import { expect, test } from '@playwright/test'

test('real frontend calls the real backend through the proxy', async ({ page }) => {
  await page.goto('/')
  await expect(page.locator('.service-card').getByRole('status')).toHaveText('连接正常')
  await page.getByRole('button', { name: '重新检查' }).click()
  await expect(page.locator('.service-card').getByRole('status')).toHaveText('连接正常')
})

for (const scenario of ['json-error', 'non-json', 'invalid-success', 'network', 'timeout']) {
  test(`user sees ${scenario} and retry recovers`, async ({ page }) => {
    await page.route('**/api/v1/health', async (route) => {
      if (scenario === 'json-error')
        await route.fulfill({
          status: 500,
          json: { error: { code: 'internal_error', message: 'private-detail' } },
        })
      else if (scenario === 'non-json')
        await route.fulfill({ status: 502, contentType: 'text/html', body: 'private-detail' })
      else if (scenario === 'invalid-success')
        await route.fulfill({ status: 200, json: { data: { status: 'unknown' } } })
      else if (scenario === 'network') await route.abort('failed')
      else {
        await new Promise((done) => setTimeout(done, 5_500))
        await route.abort().catch(() => {})
      }
    })
    await page.goto('/')
    await expect(page.getByRole('alert')).toBeVisible({ timeout: 10_000 })
    await expect(page.locator('.service-card').getByRole('status')).toHaveText('连接失败')
    await expect(page.locator('body')).not.toContainText('private-detail')
    await page.unroute('**/api/v1/health')
    await page.getByRole('button', { name: '重试连接' }).click()
    await expect(page.locator('.service-card').getByRole('status')).toHaveText('连接正常')
    await expect(page.getByRole('alert')).toHaveCount(0)
  })
}
