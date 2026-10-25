import { afterEach, expect, it, vi } from 'vitest'
import { inspectWorkbook, previewWorkbook, confirmImport } from './imports'
const response = (data: unknown) =>
  new Response(JSON.stringify({ data }), { headers: { 'Content-Type': 'application/json' } })
afterEach(() => vi.unstubAllGlobals())
it('rejects malformed workbook and row diagnostics', async () => {
  vi.stubGlobal(
    'fetch',
    vi
      .fn()
      .mockResolvedValue(response({ sheets: [{ name: 'x', rows: [{ number: '2', cells: {} }] }] })),
  )
  await expect(inspectWorkbook(new File(['x'], 'x.xlsx'))).rejects.toMatchObject({
    kind: 'invalid_response',
  })
  vi.stubGlobal(
    'fetch',
    vi
      .fn()
      .mockResolvedValue(
        response({
          token: 'x',
          expires_at: 'now',
          rows: [{ number: 2, input: {}, errors: [{}], warnings: [], candidates: [], raw: {} }],
        }),
      ),
  )
  await expect(
    previewWorkbook(new File(['x'], 'x.xlsx'), {
      sheet: 'x',
      header_row: 1,
      columns: { A: 'current_species_id' },
      values: {},
    }),
  ).rejects.toMatchObject({ kind: 'invalid_response' })
})
it('confirms only explicitly selected rows with the original token', async () => {
  const fetcher = vi
    .fn()
    .mockResolvedValue(
      response({ batch: { id: 1 }, rows: [{ row_number: 2, shiny_record_id: 3 }] }),
    )
  vi.stubGlobal('fetch', fetcher)
  await expect(confirmImport('preview-token', [2], true)).resolves.toMatchObject({
    batch: { id: 1 },
  })
  expect(JSON.parse(fetcher.mock.calls[0]![1].body)).toEqual({
    token: 'preview-token',
    rows: [2],
    acknowledge_warnings: true,
  })
})

it('does not report success if confirmed rows do not match the selection', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response({ batch: { id: 1 }, rows: [] })))
  await expect(confirmImport('token', [2], true)).rejects.toMatchObject({
    kind: 'invalid_response',
  })
})
