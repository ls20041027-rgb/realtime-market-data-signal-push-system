import { ref, type Ref } from 'vue'
import type { PageData } from '@/types'

export interface UsePaginationOptions {
  pageSize?: number
}

export function usePagination<T>(
  loader: (page: number, pageSize: number) => Promise<PageData<T>>,
  options: UsePaginationOptions = {},
) {
  const items: Ref<T[]> = ref([]) as Ref<T[]>
  const total = ref(0)
  const page = ref(1)
  const pageSize = ref(options.pageSize ?? 20)
  const loading = ref(false)
  const error = ref<Error | null>(null)

  const load = async (targetPage = page.value) => {
    loading.value = true
    error.value = null
    try {
      const res = await loader(targetPage, pageSize.value)
      items.value = res.items
      total.value = res.total
      page.value = res.page
      pageSize.value = res.page_size
    } catch (err) {
      error.value = err as Error
    } finally {
      loading.value = false
    }
  }

  const setPage = (n: number) => load(n)
  const setPageSize = (n: number) => {
    pageSize.value = n
    load(1)
  }

  return { items, total, page, pageSize, loading, error, load, setPage, setPageSize }
}
