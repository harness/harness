import { useMemo } from 'react'
import { X_TOTAL, X_TOTAL_PAGES, X_PER_PAGE } from 'utils/Utils'

export function useGetPaginationInfo(response: Nullable<Response>) {
  const totalItems = useMemo(() => parseInt(response?.headers?.get(X_TOTAL) || '0'), [response])
  const totalPages = useMemo(() => parseInt(response?.headers?.get(X_TOTAL_PAGES) || '0'), [response])
  const pageSize = useMemo(() => parseInt(response?.headers?.get(X_PER_PAGE) || '0'), [response])

  return { totalItems, totalPages, pageSize }
}
