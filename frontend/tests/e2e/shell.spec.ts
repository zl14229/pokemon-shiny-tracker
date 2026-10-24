import { expect, test } from '@playwright/test'

test('application starts and Element Plus interaction works on desktop and mobile', async ({ page }) => {
  const errors: string[] = []
  page.on('pageerror', (error) => errors.push(error.message))
  for (const width of [1280, 390]) {
    await page.setViewportSize({ width, height: 820 })
    await page.goto('/')
    await expect(page.getByRole('heading', { name: 'Pokémon Shiny Tracker' })).toBeVisible()
    await page.getByRole('button', { name: '查看说明' }).click()
    await expect(page.locator('#usage-guide')).toBeVisible()
    await page.getByRole('button', { name: '收起说明' }).click()
    await expect(page.locator('#usage-guide')).toHaveCount(0)
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
  }
  expect(errors).toEqual([])
})
