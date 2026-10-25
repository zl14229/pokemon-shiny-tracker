import { expect, test, type Page } from '@playwright/test'
import { execFileSync } from 'node:child_process'
const bytes = () => execFileSync('python3', ['../scripts/anonymous_workbook.py'])
async function map(page: Page, col: string, field: string) {
  const control = page.getByRole('combobox', { name: `映射列${col}`, exact: true })
  await control.focus()
  await control.press('ArrowDown')
  await page
    .getByRole('listbox', { name: `映射列${col}`, exact: true })
    .getByRole('option', { name: field, exact: true })
    .click()
}
async function preview(page: Page) {
  await page.goto('/#/data/import')
  await page.getByLabel('选择 Excel 文件', { exact: true }).setInputFiles({
    name: 'anonymous.xlsx',
    mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    buffer: bytes(),
  })
  await map(page, 'A', '当前物种全国编号')
  await map(page, 'B', '获得日期（ISO）')
  await map(page, 'C', '球种')
  await map(page, 'D', '备注（多列拼接原值）')
  await page.getByRole('button', { name: '生成只读预览', exact: true }).click()
  await expect(page.getByRole('heading', { name: '预览与逐行校验' })).toBeVisible()
}
test('Excel mapping preview, row errors, explicit confirmation and no fabricated Hunt', async ({
  page,
  request,
}) => {
  const before = (await (await request.get('/api/v1/shiny-records')).json()).meta.total
  const hunts = (await (await request.get('/api/v1/hunts')).json()).meta.total
  await preview(page)
  expect((await (await request.get('/api/v1/shiny-records')).json()).meta.total).toBe(before)
  await expect(page.getByRole('checkbox', { name: '选择第3行', exact: true })).toBeDisabled()
  await page.getByRole('checkbox', { name: '选择第2行', exact: true }).press('Space')
  await expect(page.getByRole('button', { name: '确认导入选中个体' })).toBeDisabled()
  await page.getByRole('checkbox', { name: /我已核对选中行/ }).press('Space')
  await page.getByRole('button', { name: '确认导入选中个体' }).click()
  await expect(page.getByRole('heading', { name: '导入成功', exact: true })).toBeVisible()
  expect((await (await request.get('/api/v1/shiny-records')).json()).meta.total).toBe(before + 1)
  expect((await (await request.get('/api/v1/hunts')).json()).meta.total).toBe(hunts)
  await page.getByRole('button', { name: '查看个体', exact: true }).first().click()
  await expect(page.getByRole('link', { name: /个体 #/ }).first()).toBeVisible()
  const batch = (await (await request.get('/api/v1/imports/batches')).json()).data[0]
  expect(batch.selected_json).toBe('[2]')
  await page.screenshot({ path: 'test-results/import-desktop.png', fullPage: true })
  await page.setViewportSize({ width: 390, height: 844 })
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(
    true,
  )
  await page.screenshot({ path: 'test-results/import-mobile.png', fullPage: true })
})
test('committed confirmation with lost response retries same token without duplicate', async ({
  page,
  request,
}) => {
  await preview(page)
  await page.getByRole('checkbox', { name: '选择第2行', exact: true }).press('Space')
  await page.getByRole('checkbox', { name: /我已核对选中行/ }).press('Space')
  const before = (await (await request.get('/api/v1/shiny-records')).json()).meta.total
  let fail = true
  await page.route('**/api/v1/imports/confirm', async (route) => {
    const response = await route.fetch()
    if (fail) {
      fail = false
      await route.fulfill({ status: 502, body: 'unknown proxy result' })
    } else await route.fulfill({ response })
  })
  await page.getByRole('button', { name: '确认导入选中个体' }).click()
  await expect(page.getByRole('button', { name: '用相同令牌安全重试' })).toBeEnabled()
  await expect(page.getByRole('checkbox', { name: '选择第4行', exact: true })).toBeDisabled()
  await page.getByRole('button', { name: '用相同令牌安全重试' }).click()
  await expect(page.getByRole('heading', { name: '导入成功', exact: true })).toBeVisible()
  expect((await (await request.get('/api/v1/shiny-records')).json()).meta.total).toBe(before + 1)
})
