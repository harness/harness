import { apiFetch, getBaseUrl } from 'services/fetcher'
import type { StoContext } from './stoContext'

export type { ErrorWrapper } from 'services/fetcher'

const baseUrl = process.env.STO_API_URL || getBaseUrl('sto')

export type StoFetcherOptions<TBody, THeaders, TQueryParams, TPathParams> = {
  url: string
  method: string
  body?: TBody
  headers?: THeaders
  queryParams?: TQueryParams
  pathParams?: TPathParams
} & StoContext['fetcherOptions']

export async function stoFetch<
  TData,
  TError,
  TBody extends {} | undefined | null,
  THeaders extends {},
  TQueryParams extends {},
  TPathParams extends {},
>(options: {
  headers?: { authorization: string }
  method: string
  queryParams?: any
  signal: AbortSignal | undefined
  url: string
}): Promise<TData> {
  // @ts-ignore
  return apiFetch<TData, TError, TBody, THeaders, TQueryParams, TPathParams>({ ...options, baseUrl })
}
