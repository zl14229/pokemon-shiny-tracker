import { expect, test } from '@playwright/test'
const tinyPNG = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aO2UAAAAASUVORK5CYII=',
  'base64',
)
test('attachment upload content link and safe removal on real individual', async ({
  page,
  request,
}) => {
  const s = (
    await (
      await request.post('/api/v1/shiny-records', {
        data: { current_species_id: 1, notes: 'T013 anonymous' },
      })
    ).json()
  ).data
  await page.goto(`/#/collection/${s.id}`)
  const panel = page.getByRole('region', { name: '附件与链接', exact: true })
  await expect(panel.getByText('暂无附件或链接', { exact: true })).toBeVisible()
  await panel
    .getByLabel('上传附件文件', { exact: true })
    .setInputFiles({ name: '匿名图片.png', mimeType: 'image/png', buffer: tinyPNG })
  await expect(panel.getByRole('img', { name: '匿名图片.png', exact: true })).toBeVisible()
  await panel
    .getByLabel('上传附件文件', { exact: true })
    .setInputFiles({
      name: '匿名笔记.txt',
      mimeType: 'text/plain',
      buffer: Buffer.from('anonymous attachment'),
    })
  await expect(panel.getByText('匿名笔记.txt', { exact: true })).toBeVisible()
  await panel.getByRole('textbox', { name: '附件链接名称', exact: true }).fill('proof video')
  await panel
    .getByRole('textbox', { name: '附件链接地址', exact: true })
    .fill('https://example.com/proof')
  await panel.getByRole('button', { name: '关联外部链接', exact: true }).click()
  const link = panel.getByRole('link', { name: '打开外部链接', exact: true })
  await expect(link).toHaveAttribute('href', 'https://example.com/proof')
  await expect(link).toHaveAttribute('rel', 'noopener noreferrer')
  const list = (await (await request.get(`/api/v1/shiny-records/${s.id}/attachments`)).json()).data
  expect(list).toHaveLength(3)
  const text = list.find((a: { name: string }) => a.name === '匿名笔记.txt')
  const content = await request.get(`/api/v1/attachments/${text.id}/content`)
  expect(await content.text()).toBe('anonymous attachment')
  expect(content.headers()['x-content-type-options']).toBe('nosniff')
  await panel.getByRole('button', { name: `移除附件${text.id}`, exact: true }).click()
  await page.getByRole('button', { name: '确认移除', exact: true }).click()
  await expect(panel.getByText('匿名笔记.txt', { exact: true })).toHaveCount(0)
  expect((await request.get(`/api/v1/attachments/${text.id}/content`)).status()).toBe(404)
  await page.setViewportSize({ width: 390, height: 844 })
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(
    true,
  )
  await page.screenshot({ path: 'test-results/attachments-mobile.png', fullPage: true })
})
test('rejects invalid file and URL and locks ambiguous committed upload until list checked', async ({
  page,
  request,
}) => {
  const s = (
    await (await request.post('/api/v1/shiny-records', { data: { current_species_id: 1 } })).json()
  ).data
  await page.goto(`/#/collection/${s.id}`)
  const panel = page.getByRole('region', { name: '附件与链接', exact: true })
  await expect(panel.getByRole('button', { name: '上传本地附件', exact: true })).toBeEnabled()
  await panel
    .getByLabel('上传附件文件', { exact: true })
    .setInputFiles({
      name: 'bad.exe',
      mimeType: 'application/octet-stream',
      buffer: Buffer.from('bad'),
    })
  await expect(panel.getByRole('alert')).toContainText('请选择')
  await panel.getByRole('textbox', { name: '附件链接名称', exact: true }).fill('bad')
  await panel
    .getByRole('textbox', { name: '附件链接地址', exact: true })
    .fill('javascript:alert(1)')
  await panel.getByRole('button', { name: '关联外部链接', exact: true }).click()
  await expect(panel.getByRole('alert')).toContainText('HTTP/HTTPS')
  await page.route(`**/api/v1/shiny-records/${s.id}/attachments`, async (route) => {
    if (route.request().method() === 'POST') {
      await route.fetch()
      await route.fulfill({ status: 502, body: 'private failure' })
    } else await route.continue()
  })
  await panel
    .getByLabel('上传附件文件', { exact: true })
    .setInputFiles({
      name: 'saved.txt',
      mimeType: 'text/plain',
      buffer: Buffer.from('committed once'),
    })
  await expect(panel.getByRole('status')).toContainText('写入结果不明')
  await expect(panel.getByRole('button', { name: '上传本地附件', exact: true })).toBeDisabled()
  await panel.getByRole('button', { name: '刷新附件', exact: true }).click()
  await expect(panel.getByText('saved.txt', { exact: true })).toBeVisible()
  expect(
    (await (await request.get(`/api/v1/shiny-records/${s.id}/attachments`)).json()).data,
  ).toHaveLength(1)
})
