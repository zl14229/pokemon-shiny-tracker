import { expect, test } from '@playwright/test'

test('M4 dual dex follows one evolving and transferred identity and works offline', async ({
  page,
  request,
}) => {
  const errors: string[] = []
  page.on('pageerror', (e) => errors.push(e.message))
  const species = (await (await request.get('/api/v1/reference/species?limit=500')).json()).data
  const a = species.find((s: { national_dex: number }) => s.national_dex === 1)
  const b = species.find((s: { national_dex: number }) => s.national_dex === 2)
  const fa = (
    await (await request.get(`/api/v1/reference/forms?species_id=${a.id}`)).json()
  ).data.find((f: { slug: string }) => f.slug === 'bulbasaur')
  const fb = (
    await (await request.get(`/api/v1/reference/forms?species_id=${b.id}`)).json()
  ).data.find((f: { slug: string }) => f.slug === 'ivysaur')
  const input = {
    original_species_id: a.id,
    original_form_id: fa.id,
    current_species_id: a.id,
    current_form_id: fa.id,
    original_game_id: 1,
    current_game_id: 1,
    holding_status: 'held',
    source: 'self_hunted',
  }
  const created = await request.post('/api/v1/shiny-records', { data: input })
  expect(created.status()).toBe(201)
  const id = (await created.json()).data.id
  const evolved = {
    ...input,
    current_species_id: b.id,
    current_form_id: fb.id,
    change_type: 'evolution',
  }
  expect((await request.put(`/api/v1/shiny-records/${id}`, { data: evolved })).status()).toBe(200)
  expect(
    (
      await request.put(`/api/v1/shiny-records/${id}`, {
        data: { ...evolved, change_type: '', current_game_id: 2 },
      })
    ).status(),
  ).toBe(200)
  await page.goto('/#/dex?q=bulbasaur')
  const card = page.getByRole('article', { name: '妙蛙种子', exact: true })
  await expect(card).toBeVisible()
  await expect(card).toContainText('历史 1 · 当前持有 0')
  await expect(card.getByRole('link', { name: `查看个体 #${id}`, exact: true })).toBeVisible()
  await expect(page.getByText('分母待核实', { exact: true })).toBeVisible()
  await expect(card).toContainText('中文名待核实')
  await page.getByRole('combobox', { name: '统计维度', exact: true }).selectOption('form')
  await page.getByRole('button', { name: '应用筛选', exact: true }).click()
  await expect(card).toContainText('历史 1 · 当前持有 0')
  await page.getByRole('combobox', { name: '收藏口径', exact: true }).selectOption('current')
  await page.getByRole('combobox', { name: '记录状态', exact: true }).selectOption('obtained')
  await page.getByRole('button', { name: '应用筛选', exact: true }).click()
  await expect(page.getByText('此筛选下暂无目标，可更换条件查看。')).toBeVisible()
  await page.goto('/#/dex?scope=game&game_id=2&basis=current&q=ivysaur&acquired=obtained')
  await expect(page.getByRole('article', { name: '妙蛙草', exact: true })).toContainText(
    '当前持有 1',
  )
  await page.goto('/#/dex?scope=game&game_id=2&basis=historical&q=bulbasaur&acquired=obtained')
  await expect(page.getByText('此筛选下暂无目标，可更换条件查看。')).toBeVisible()
  await page.route('https://**', (route) => route.abort())
  await page.goto('/#/dex?q=妙蛙种子')
  await page.reload()
  await expect(card).toBeVisible()
  await expect(card.getByRole('img', { name: '妙蛙种子 图片暂不可用' })).toBeVisible()
  expect(errors).toEqual([])
})

test('M4 verified missing list stays unknown, pagination and mobile do not overflow', async ({
  page,
}) => {
  await page.goto('/#/dex')
  await expect(page.getByRole('heading', { name: '闪光图鉴', exact: true })).toBeVisible()
  await expect(page.getByRole('article')).toHaveCount(24)
  const first = await page.getByRole('article').first().getAttribute('aria-label')
  await page.getByRole('button', { name: '下一页', exact: true }).click()
  await expect(page.getByText('第 2 页', { exact: true })).toBeVisible()
  expect(await page.getByRole('article').first().getAttribute('aria-label')).not.toBe(first)
  await page.getByRole('button', { name: '查看已核实缺失', exact: true }).click()
  await expect(page.getByText('当前范围尚无已核实的合法目标，缺失清单待资料核验。')).toBeVisible()
  await expect(page.getByText('分母待核实', { exact: true })).toBeVisible()
  await page.setViewportSize({ width: 390, height: 844 })
  await page.getByRole('button', { name: '重置', exact: true }).click()
  await expect(page.getByRole('article')).toHaveCount(24)
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
  await page.screenshot({ path: 'test-results/m4-dex-mobile.png', fullPage: true })
  await page.getByRole('article').first().screenshot({ path: 'test-results/m4-dex-card.png' })
})

test('M4 dex read failure is redacted and retry recovers', async ({ page }) => {
  let fail = true
  await page.route('**/api/v1/dex?**', async (route) => {
    if (fail) {
      fail = false
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({
          error: { code: 'internal_error', message: 'secret must not render' },
        }),
      })
    } else await route.continue()
  })
  await page.goto('/#/dex')
  await expect(page.getByText('本地服务暂时不可用，请稍后重试。', { exact: true })).toBeVisible()
  await expect(page.getByText('secret must not render')).toHaveCount(0)
  await page.getByRole('button', { name: '重试', exact: true }).click()
  await expect(page.getByRole('article')).toHaveCount(24)
})
