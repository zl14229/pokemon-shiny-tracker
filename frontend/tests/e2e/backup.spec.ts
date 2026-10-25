import { expect, test } from '@playwright/test'
import { execFileSync } from 'node:child_process'
import { mkdtemp, readFile, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
test('browser downloads JSON and verified ZIP then CLI restores to a new isolated directory', async ({
  page,
  request,
}) => {
  const dir = await mkdtemp(join(tmpdir(), 'pst-download-test-'))
  try {
    const r = (
      await (
        await request.post('/api/v1/shiny-records', {
          data: { current_species_id: 1, notes: 'T014 browser recovery evidence' },
        })
      ).json()
    ).data
    const a = (
      await (
        await request.post(`/api/v1/shiny-records/${r.id}/attachments`, {
          multipart: {
            file: {
              name: 'recovery.txt',
              mimeType: 'text/plain',
              buffer: Buffer.from('recover exact anonymous bytes'),
            },
          },
        })
      ).json()
    ).data
    await page.goto('/#/data/backup')
    const [jsonDownload] = await Promise.all([
      page.waitForEvent('download'),
      page.getByRole('button', { name: '下载 JSON 数据', exact: true }).click(),
    ])
    const jsonPath = join(dir, 'export.json')
    await jsonDownload.saveAs(jsonPath)
    const exported = JSON.parse(await readFile(jsonPath, 'utf8'))
    expect(exported.tables.shiny_records.some((x: { id: number }) => x.id === r.id)).toBe(true)
    const [zipDownload] = await Promise.all([
      page.waitForEvent('download'),
      page.getByRole('button', { name: '下载 ZIP 备份', exact: true }).click(),
    ])
    const zipPath = join(dir, 'backup.zip')
    await zipDownload.saveAs(zipPath)
    expect(
      execFileSync('../bin/shiny-data', ['verify', '--archive', zipPath], { encoding: 'utf8' }),
    ).toContain('PASS')
    const dest = join(dir, 'restored')
    expect(
      execFileSync('../bin/shiny-data', ['restore', '--archive', zipPath, '--destination', dest], {
        encoding: 'utf8',
      }),
    ).toContain('PASS')
    const restoredJSON = join(dir, 'restored.json')
    execFileSync('../bin/shiny-data', [
      'export',
      '--database',
      join(dest, 'database.sqlite3'),
      '--output',
      restoredJSON,
    ])
    const restored = JSON.parse(await readFile(restoredJSON, 'utf8'))
    expect(restored.tables.shiny_records.find((x: { id: number }) => x.id === r.id).notes).toBe(
      'T014 browser recovery evidence',
    )
    const metadata = restored.tables.attachments.find((x: { id: number }) => x.id === a.id)
    expect(await readFile(join(dest, 'database.sqlite3.files', metadata.storage_key), 'utf8')).toBe(
      'recover exact anonymous bytes',
    )
    expect((await request.get(`/api/v1/shiny-records/${r.id}`)).status()).toBe(200)
    await page.setViewportSize({ width: 390, height: 844 })
    expect(
      await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
    ).toBe(true)
    await page.screenshot({ path: 'test-results/backup-mobile.png', fullPage: true })
  } finally {
    await rm(dir, { recursive: true, force: true })
  }
})
test('backup generation failure is redacted and retry can download', async ({ page }) => {
  await page.goto('/#/data/backup')
  let fail = true
  await page.route('**/api/v1/data/backup', async (route) => {
    if (fail) {
      fail = false
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({
          error: { code: 'internal_error', message: 'private database path' },
        }),
      })
    } else await route.continue()
  })
  await page.getByRole('button', { name: '下载 ZIP 备份', exact: true }).click()
  await expect(page.getByRole('alert').filter({ hasText: '本地服务' })).toBeVisible()
  await expect(page.locator('body')).not.toContainText('private database path')
  const [download] = await Promise.all([
    page.waitForEvent('download'),
    page.getByRole('button', { name: '下载 ZIP 备份', exact: true }).click(),
  ])
  expect(download.suggestedFilename()).toBe('shiny-backup.zip')
})
