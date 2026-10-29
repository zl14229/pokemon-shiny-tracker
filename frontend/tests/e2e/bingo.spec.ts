import { expect, test } from '@playwright/test'
async function createCard(page: import('@playwright/test').Page) {
  await page.goto('/#/planner/bingo')
  await page.getByRole('checkbox', { name: '朱', exact: true }).check()
  await page.getByRole('checkbox', { name: '紫', exact: true }).check()
  await page.getByRole('button', { name: '创建 Bingo', exact: true }).click()
  await expect(page).toHaveURL(/#\/planner\/plans\/\d+$/)
  await expect(page.getByRole('article')).toHaveCount(9)
  await expect(page.getByRole('button', { name: '核对新成果', exact: true })).toBeEnabled()
  return Number(page.url().split('/').at(-1))
}
test('M5 Bingo persists multiple cards, protects rerolls, reviews new records and preserves a completed line', async ({
  page,
  request,
}) => {
  const errors: string[] = []
  page.on('pageerror', (e) => errors.push(e.message))
  const first = await createCard(page),
    second = await createCard(page)
  expect(first).not.toBe(second)
  await page.goto(`/#/planner/plans/${first}`)
  await expect(page.getByRole('button', { name: '核对新成果', exact: true })).toBeEnabled()
  let card = (await (await request.get(`/api/v1/planner/plans/${first}`)).json()).data
  await page
    .getByRole('article')
    .first()
    .getByRole('button', { name: '开始此格', exact: true })
    .click()
  await expect(
    page
      .getByRole('article')
      .first()
      .getByRole('link', { name: /查看 Hunt #/ }),
  ).toBeVisible()
  const started = card.targets[0]
  expect(
    (
      await request.post(`/api/v1/planner/plans/${first}/targets/${started.id}/reroll`, {
        data: { request_id: 'browser-started-reroll-001', expected_target_key: started.target_key },
      })
    ).status(),
  ).toBe(409)
  const old = card.targets[8].target_key
  await page
    .getByRole('article')
    .nth(8)
    .getByRole('button', { name: '重抽未开始格', exact: true })
    .click()
  await expect(page.getByText('未开始格子已重新抽取，原目标保留在历史中。')).toBeVisible()
  card = (await (await request.get(`/api/v1/planner/plans/${first}`)).json()).data
  expect(card.targets[8].target_key).not.toBe(old)
  expect(new Set(card.targets.map((t: { target_key: string }) => t.target_key)).size).toBe(9)
  const records: { id: number; created_at: string; day: string; local: string }[] = []
  const zone = card.bingo.calendar_zone
  const startDay = await page.evaluate(
    ({ at, zone }) =>
      new Intl.DateTimeFormat('en-CA', {
        timeZone: zone,
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
      }).format(new Date(at)),
    { at: card.bingo.started_at, zone },
  )
  // datetime-local carries whole seconds. Ensure truncating a new record's
  // timestamp still proves it is strictly after the challenge, even on fast CI.
  await expect
    .poll(() => Math.floor(Date.now() / 1000) * 1000)
    .toBeGreaterThan(Date.parse(card.bingo.started_at))
  for (const t of card.targets.slice(0, 3)) {
    const day = await page.evaluate(
      (zone) =>
        new Intl.DateTimeFormat('en-CA', {
          timeZone: zone,
          year: 'numeric',
          month: '2-digit',
          day: '2-digit',
        }).format(new Date()),
      zone,
    )
    const form = t.snapshot.choice.evidence[0].form_id
    const response = await request.post('/api/v1/shiny-records', {
      data: {
        hunt_id: t.hunt_id,
        original_species_id: t.species_id,
        original_form_id: form,
        current_species_id: t.species_id,
        current_form_id: form,
        original_game_id: t.game_id,
        current_game_id: t.game_id,
        obtained_date: day,
        source: 'self_hunted',
        holding_status: 'held',
        notes: 'T028 synthetic browser acceptance individual',
      },
    })
    expect(response.status()).toBe(201)
    const r = (await response.json()).data
    expect(Math.floor(Date.parse(r.created_at) / 1000) * 1000).toBeGreaterThan(
      Date.parse(card.bingo.started_at),
    )
    const local = await page.evaluate((at) => {
      const d = new Date(Math.floor(Date.parse(at) / 1000) * 1000)
      return new Date(d.getTime() - d.getTimezoneOffset() * 60000).toISOString().slice(0, 19)
    }, r.created_at)
    records.push({ id: r.id, created_at: r.created_at, day, local })
  }
  await page.getByRole('button', { name: '核对新成果', exact: true }).click()
  await expect(page.getByRole('button', { name: '核对新成果', exact: true })).toBeEnabled()
  for (let i = 0; i < 3; i++) {
    const article = page.getByRole('article').nth(i),
      record = records[i]!
    if (record.day === startDay) {
      await expect(article).toContainText('有记录待核验，尚未完成')
      await article.getByRole('button', { name: '补充时间并人工核验', exact: true }).click()
      await page.getByLabel('具体获得时间', { exact: true }).fill(record.local)
      await page
        .getByRole('textbox', { name: '核验依据', exact: true })
        .fill('Synthetic browser fixture: capture time reviewed against new individual timestamp')
      await page
        .getByRole('checkbox', {
          name: '我已核对该个体在挑战创建后亲自合法获得，时间与原记录一致',
          exact: true,
        })
        .check()
      await page.getByRole('button', { name: '确认时间证据并完成', exact: true }).click()
    }
    await expect(article).toContainText('已完成')
    await expect(
      article.getByRole('link', { name: `完成个体 #${record.id}`, exact: true }),
    ).toBeVisible()
    await expect(article.getByRole('button', { name: '重抽未开始格', exact: true })).toHaveCount(0)
  }
  await expect(page.getByText(/已连线 1 条/)).toBeVisible()
  expect(
    (
      await request.post(`/api/v1/planner/plans/${first}/targets/${started.id}/reroll`, {
        data: {
          request_id: 'browser-completed-reroll-001',
          expected_target_key: started.target_key,
        },
      })
    ).status(),
  ).toBe(409)
  await page.getByRole('button', { name: '结束已连线挑战', exact: true }).click()
  await expect(page.getByText('挑战已完成 · 历史保留', { exact: true })).toBeVisible()
  await page.route('https://**', (r) => r.abort())
  await page.reload()
  await expect(page.getByText('挑战已完成 · 历史保留', { exact: true })).toBeVisible()
  await expect(page.getByText(/已连线 1 条/)).toBeVisible()
  await expect(page.getByRole('article')).toHaveCount(9)
  await page.setViewportSize({ width: 390, height: 844 })
  await expect
    .poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
    .toBe(true)
  await page.goto(`/#/planner/plans/${second}`)
  await expect(page.getByRole('button', { name: '核对新成果', exact: true })).toBeEnabled()
  await page.getByRole('button', { name: '放弃挑战并保留历史', exact: true }).click()
  await expect(page.getByText('挑战已放弃 · 历史保留', { exact: true })).toBeVisible()
  await expect(page.getByRole('article')).toHaveCount(9)
  expect(errors).toEqual([])
})
test('M5 Bingo 5x5 refuses incomplete real coverage and leaves no partial challenge', async ({
  page,
  request,
}) => {
  const before = (await (await request.get('/api/v1/planner/plans')).json()).meta.total
  await page.goto('/#/planner/bingo')
  await page.getByRole('checkbox', { name: '朱', exact: true }).check()
  await page.getByRole('checkbox', { name: '紫', exact: true }).check()
  await page.getByRole('combobox', { name: '卡片尺寸', exact: true }).selectOption('5')
  await page.getByRole('button', { name: '创建 Bingo', exact: true }).click()
  await expect(page.getByRole('alert')).toContainText('需要 25 个唯一目标')
  await expect(page.getByText('候选数量不足，不能创建完整卡片。')).toBeVisible()
  expect((await (await request.get('/api/v1/planner/plans')).json()).meta.total).toBe(before)
})
