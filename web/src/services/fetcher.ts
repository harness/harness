export type FetcherOptions<TBody, THeaders, TQueryParams, TPathParams, TFetcherOptions = unknown> = {
  baseUrl: string
  url: string
  method: string
  body?: TBody
  headers?: THeaders
  queryParams?: TQueryParams
  pathParams?: TPathParams
  fetcherOptions?: TFetcherOptions
}

export const getBaseUrl = (servicePath: string) => {
  if (window.location.hash) {
    return window.apiUrl ? `${window.apiUrl}/${servicePath}` : window.location.pathname.replace('ng/', '') + servicePath
  } else {
    return window.apiUrl ? `${window.apiUrl}/${servicePath}` : `/${servicePath}`
  }
}

export type ParseError = { payload: { message: string; status?: number } }
export type ErrorStatus = { payload: { status?: number } }
export type ErrorWrapper<TError> = TError | ParseError
export type QueryParams = Record<string, string | string[] | undefined>
export type PathParams = Record<string, string>

export function resolveUrl(url: string, queryParams?: QueryParams, pathParams?: PathParams) {
  let query = ''
  if (queryParams !== undefined) {
    const params = new URLSearchParams()
    Object.entries(queryParams).forEach(([k, v]) => {
      if (v !== undefined) {
        if (Array.isArray(v)) {
          v.forEach(val => params.append(k, val))
        } else {
          params.append(k, v)
        }
      }
    })

    query = new URLSearchParams(params).toString()
  }

  if (query) {
    query = `?${query}`
  }

  if (pathParams) {
    url = url.replace(/\{\w*}/g, key => {
      key = key.slice(1, -1)
      const replacement = pathParams[key]
      if (replacement === undefined) {
        throw new Error(`No value specified for path parameter ${key}`)
      }

      return replacement
    })
  }

  return url + query
}

export async function apiFetch<
  TData,
  TError,
  TBody extends {} | undefined | null,
  THeaders extends {},
  TQueryParams extends {},
  TPathParams extends {},
>({
  baseUrl,
  url,
  method,
  body,
  headers,
  pathParams,
  queryParams,
}: FetcherOptions<TBody, THeaders, TQueryParams, TPathParams>): Promise<TData> {
  const response = await window.fetch(`${baseUrl}${resolveUrl(url, queryParams, pathParams)}`, {
    method: method.toUpperCase(),
    body: body ? JSON.stringify(body) : undefined,
    headers: {
      'Content-Type': 'application/json',
      ...headers,
    },
  })

  let error: ErrorWrapper<TError>
  if (!response.ok) {
    try {
      error = {
        payload: await response.json(),
      }
      error.payload.status = error.payload.status ? error.payload.status : response.status || 0
    } catch (e) {
      error = {
        payload: {
          message: `Failed to fetch: ${response.statusText || response.status}`,
          status: response.status || 0,
        },
      }
    }

    throw error
  }

  try {
    return response.status === 204 ? ({} as TData) : await response.json()
  } catch (e) {
    error = {
      payload: { message: `Failed to fetch: invalid response`, status: response.status || 0 },
    }

    throw error
  }
}
