import { useLocation, useHistory } from 'react-router-dom'
import qs from 'qs'
import type { IStringifyOptions } from 'qs'

import { useQueryParams } from './useQueryParams'

export interface UseUpdateQueryParamsReturn<T> {
  updateQueryParams(values: T, options?: IStringifyOptions, replaceHistory?: boolean): void
  replaceQueryParams(values: T, options?: IStringifyOptions, replaceHistory?: boolean): void
}

export function useUpdateQueryParams<T = Record<string, string>>(
  globalOptions?: IStringifyOptions
): UseUpdateQueryParamsReturn<T> {
  const { pathname } = useLocation()
  const { push, replace } = useHistory()
  const queryParams = useQueryParams<T>()

  return {
    updateQueryParams(values: T, options?: IStringifyOptions, replaceHistory = false): void {
      const path = `${pathname}?${qs.stringify({ ...queryParams, ...values }, { ...globalOptions, ...options })}`
      replaceHistory ? replace(path) : push(path)
    },
    replaceQueryParams(values: T, options?: IStringifyOptions, replaceHistory = false): void {
      const path = `${pathname}?${qs.stringify(values, { ...globalOptions, ...options })}`
      replaceHistory ? replace(path) : push(path)
    }
  }
}
