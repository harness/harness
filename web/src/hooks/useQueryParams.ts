import React from 'react'
import { useLocation } from 'react-router-dom'
import qs from 'qs'
import type { IParseOptions } from 'qs'

export interface UseQueryParamsOptions<T> extends IParseOptions {
  processQueryParams?(data: any): T
}

export function useQueryParams<T = unknown>(options?: UseQueryParamsOptions<T>): T {
  const { search } = useLocation()

  const queryParams = React.useMemo(() => {
    const params = qs.parse(search, { ignoreQueryPrefix: true, ...options })

    if (typeof options?.processQueryParams === 'function') {
      return options.processQueryParams(params)
    }

    return params
  }, [search, options, options?.processQueryParams])

  return queryParams as unknown as T
}
