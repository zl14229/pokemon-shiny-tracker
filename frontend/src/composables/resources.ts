import { onBeforeUnmount, ref, shallowRef } from 'vue'
import { tracker, BeforeWriteError } from '../api/tracker'
import { ApiError } from '../api/client'
import { emptyCatalog } from '../domain/format'
export const errorText = (e: unknown) =>
  e instanceof ApiError || e instanceof BeforeWriteError ? e.message : '操作未完成，请稍后重试。'
export const uncertainWrite = (e: unknown) =>
  !(e instanceof BeforeWriteError) &&
  !(
    e instanceof ApiError &&
    e.kind === 'http' &&
    e.status !== null &&
    e.status >= 400 &&
    e.status < 500
  )
export function useCatalog() {
  const catalog = shallowRef(emptyCatalog()),
    catalogError = ref(''),
    catalogLoading = ref(false)
  let generation = 0,
    controller: AbortController | undefined
  async function loadCatalog() {
    const current = ++generation
    controller?.abort()
    controller = new AbortController()
    catalogLoading.value = true
    catalogError.value = ''
    try {
      const result = await tracker.catalog({ signal: controller.signal })
      if (current === generation) catalog.value = result
    } catch (e) {
      if (current === generation) catalogError.value = errorText(e)
    } finally {
      if (current === generation) catalogLoading.value = false
    }
  }
  onBeforeUnmount(() => {
    ++generation
    controller?.abort()
  })
  return { catalog, catalogError, catalogLoading, loadCatalog }
}
