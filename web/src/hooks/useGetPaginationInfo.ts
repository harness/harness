import { useMemo } from 'react'

export function useGetPaginationInfo(response: Nullable<Response>) {
  const totalItems = useMemo(() => parseInt(response?.headers?.get('x-total') || '0'), [response])
  const totalPages = useMemo(() => parseInt(response?.headers?.get('x-total-pages') || '0'), [response])
  const pageSize = useMemo(() => parseInt(response?.headers?.get('x-per-page') || '0'), [response])
  const X_NEXT_PAGE = useMemo(() => parseInt(response?.headers?.get('x-next-page') || '0'), [response])
  const X_PREV_PAGE = useMemo(() => parseInt(response?.headers?.get('x-prev-page') || '0'), [response])

  return { totalItems, totalPages, pageSize, X_NEXT_PAGE, X_PREV_PAGE }
}
