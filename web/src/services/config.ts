import { mapKeys } from 'lodash-es'
import qs from 'qs'

export const getConfigNew = (str: string): string => {
  // NOTE: Replace /^code\// with your service prefixes when running in standalone mode
  // I.e: 'code/api/v1' -> 'api/v1'     (standalone)
  //                   -> 'code/api/v1' (embedded inside Harness platform)
  if (window.STRIP_CODE_PREFIX) {
    str = str.replace(/^code\//, '')
  }

  return window.apiUrl ? `${window.apiUrl}/${str}` : window.location.pathname.replace('ng/', '') + str
}

export interface GetUsingFetchProps<
  _TData = any,
  _TError = any,
  TQueryParams = {
    [key: string]: any
  },
  TPathParams = {
    [key: string]: any
  }
> {
  queryParams?: TQueryParams
  pathParams?: TPathParams
  requestOptions?: RequestInit
  mock?: _TData
}

export const getUsingFetch = <
  TData = any,
  _TError = any,
  TQueryParams = {
    [key: string]: any
  },
  TPathParams = {
    [key: string]: any
  }
>(
  base: string,
  path: string,
  props: {
    queryParams?: TQueryParams
    pathParams?: TPathParams
    requestOptions?: RequestInit
    mock?: TData
  },
  signal?: RequestInit['signal']
): Promise<TData> => {
  if (props.mock) return Promise.resolve(props.mock)
  let url = base + path
  if (props.queryParams && Object.keys(props.queryParams).length) {
    url += `?${qs.stringify(props.queryParams)}`
  }
  return fetch(url, {
    signal,
    ...(props.requestOptions || {}),
    headers: getHeaders(props.requestOptions?.headers)
  }).then(res => {
    const contentType = res.headers.get('content-type') || ''
    if (contentType.toLowerCase().indexOf('application/json') > -1) {
      return res.json()
    }
    return res.text()
  })
}

export interface MutateUsingFetchProps<
  _TData = any,
  _TError = any,
  TQueryParams = {
    [key: string]: any
  },
  TRequestBody = any,
  TPathParams = {
    [key: string]: any
  }
> {
  body: TRequestBody
  queryParams?: TQueryParams
  pathParams?: TPathParams
  requestOptions?: RequestInit
  mock?: _TData
}

export const mutateUsingFetch = <
  TData = any,
  _TError = any,
  TQueryParams = {
    [key: string]: any
  },
  TRequestBody = any,
  TPathParams = {
    [key: string]: any
  }
>(
  method: string,
  base: string,
  path: string,
  props: {
    body: TRequestBody
    queryParams?: TQueryParams
    pathParams?: TPathParams
    requestOptions?: RequestInit
    mock?: TData
  },
  signal?: RequestInit['signal']
): Promise<TData> => {
  if (props.mock) return Promise.resolve(props.mock)
  let url = base + path
  if (method === 'DELETE' && typeof props.body === 'string') {
    url += `/${props.body}`
  }
  if (props.queryParams && Object.keys(props.queryParams).length) {
    url += `?${qs.stringify(props.queryParams)}`
  }

  let body: BodyInit | null = null

  if (props.body instanceof FormData) {
    body = props.body
  } else if (typeof props.body === 'object') {
    try {
      body = JSON.stringify(props.body)
    } catch {
      body = props.body as any
    }
  } else {
    body = props.body as any
  }

  return fetch(url, {
    method,
    body,
    signal,
    ...(props.requestOptions || {}),
    headers: getHeaders(props.requestOptions?.headers)
  }).then(res => {
    const contentType = res.headers.get('content-type') || ''
    if (contentType.toLowerCase().indexOf('application/json') > -1) {
      return res.json()
    }
    return res.text()
  })
}

const getHeaders = (headers: RequestInit['headers'] = {}): RequestInit['headers'] => {
  const retHeaders: RequestInit['headers'] = {
    'content-type': 'application/json'
  }

  // add/overwrite passed headers
  Object.assign(
    retHeaders,
    mapKeys(headers, (_value, key) => key.toLowerCase())
  )

  return retHeaders
}
