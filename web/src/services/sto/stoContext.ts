import type { QueryKey, UseQueryOptions } from '@tanstack/react-query'
import { useRequestContext } from 'services/context'
import type { RequestContext } from 'services/context'
import type { QueryOperation } from './stoComponents'

export type StoContext = RequestContext<QueryOperation>

/**
 * Context injected into every react-query hook wrappers
 *
 * @param queryOptions options from the useQuery wrapper
 */
export function useStoContext<
  TQueryFnData = unknown,
  TError = unknown,
  TData = TQueryFnData,
  TQueryKey extends QueryKey = QueryKey,
>(queryOptions?: Omit<UseQueryOptions<TQueryFnData, TError, TData, TQueryKey>, 'queryKey' | 'queryFn'>): StoContext {
  return useRequestContext<TQueryFnData, TError, TData, TQueryKey, QueryOperation>(queryOptions)
}
