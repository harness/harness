import type { QueryKey, UseQueryOptions } from '@tanstack/react-query'
import { useState } from 'react'
import useEventListener from '@use-it/event-listener'

export type QueryOperation = {
  path: string
  operationId: string
  variables: unknown
}

export type RequestContext<TQueryOperation extends QueryOperation> = {
  fetcherOptions: {
    /**
     * Headers to inject in the fetcher
     */
    headers?: {
      authorization: string
    }
    /**
     * Query params to inject in the fetcher
     */
    queryParams?: {}
  }
  queryOptions: {
    /**
     * Set this to `false` to disable automatic refetching when the query mounts or changes query keys.
     * Defaults to `true`.
     */
    enabled?: boolean
  }
  /**
   * Query key manager.
   */
  queryKeyFn: (operation: TQueryOperation) => QueryKey
}

const API_TOKEN_KEY = 'token'

const useAPIToken = () => {
  const initial = window.localStorage.getItem(API_TOKEN_KEY) || process.env.TOKEN || ''
  const [token, setToken] = useState<string>(initial)

  useEventListener('storage', ({ key, newValue }: StorageEvent) => {
    if (key === API_TOKEN_KEY) {
      setToken(newValue || '')
    }
  })

  return decodeToken(token)
}

function decodeToken(token: string): string {
  // Base64-decode the stored session token
  const decoded = atob(token)

  // Remove the wrapping, urlencoded '"' characters
  return decoded.replace(/%22/g, '')
}

export function useRequestContext<
  TQueryFnData,
  TError,
  TData,
  TQueryKey extends QueryKey,
  TQueryOperation extends QueryOperation,
>(
  _queryOptions?: Omit<UseQueryOptions<TQueryFnData, TError, TData, TQueryKey>, 'queryKey' | 'queryFn'>,
): RequestContext<TQueryOperation> {
  const token = useAPIToken()

  return {
    fetcherOptions: {
      headers: {
        authorization: `Bearer ${token}`,
      },
    },
    queryOptions: {},
    queryKeyFn: operation => {
      const queryKey: unknown[] = hasPathParams(operation)
        ? operation.path
            .split('/')
            .filter(Boolean)
            .map(i => resolvePathParam(i, operation.variables.pathParams))
        : operation.path.split('/').filter(Boolean)

      if (hasQueryParams(operation)) {
        queryKey.push(operation.variables.queryParams)
      }

      if (hasBody(operation)) {
        queryKey.push(operation.variables.body)
      }

      return queryKey
    },
  }
}

// Helpers
export function resolvePathParam(key: string, pathParams: Record<string, string>) {
  if (key.startsWith('{') && key.endsWith('}')) {
    return pathParams[key.slice(1, -1)]
  }
  return key
}

export function hasPathParams<TQueryOperation extends QueryOperation>(
  operation: TQueryOperation,
): operation is TQueryOperation & {
  variables: { pathParams: Record<string, string> }
} {
  return Boolean((operation.variables as any).pathParams)
}

export function hasBody<TQueryOperation extends QueryOperation>(
  operation: TQueryOperation,
): operation is TQueryOperation & {
  variables: { body: Record<string, unknown> }
} {
  return Boolean((operation.variables as any).body)
}

export function hasQueryParams<TQueryOperation extends QueryOperation>(
  operation: TQueryOperation,
): operation is TQueryOperation & {
  variables: { queryParams: Record<string, unknown> }
} {
  return Boolean((operation.variables as any).queryParams)
}
