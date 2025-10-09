/*
 * Copyright 2024 Harness Inc. All rights reserved.
 * Use of this source code is governed by the PolyForm Shield 1.0.0 license
 * that can be found in the licenses directory at the root of this repository, also available at
 * https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
 */

import qs from 'qs'
import { mapKeys } from 'lodash-es'

export const generateHeaders = (headers: RequestInit['headers'] = {}): RequestInit['headers'] => {
  const retHeaders: RequestInit['headers'] = {
    'content-type': 'application/json'
  }

  const token = localStorage.getItem('token')

  if (token && token.length > 0) {
    const parsedToken = JSON.parse(decodeURIComponent(atob(token)))
    retHeaders.Authorization = `Bearer ${parsedToken}`
  }

  Object.assign(
    retHeaders,
    mapKeys(headers, (_value, key) => key.toLowerCase())
  )

  return retHeaders
}

export const generateRequestObject = (
  method: string,
  endpoint: string,
  body?: any,
  queryParams?: { [key: string]: string },
  headerOptions?: HeadersInit,
  origin?: string
): object => {
  const headers = generateHeaders(headerOptions)
  const apiOrigin = origin ?? window.location.origin
  let url = `${apiOrigin}/${endpoint}`
  if (method === 'DELETE' && typeof body === 'string') {
    url += `/${body}`
  }
  if (queryParams && Object.keys(queryParams).length) {
    url += `?${qs.stringify(queryParams)}`
  }

  let requestBody: BodyInit | null = null

  if (body instanceof FormData) {
    requestBody = body
  } else if (typeof body === 'object') {
    try {
      requestBody = JSON.stringify(body)
    } catch {
      requestBody = body
    }
  } else {
    requestBody = body
  }
  return {
    method: method,
    url: url,
    body: requestBody,
    headers: headers
  }
}
