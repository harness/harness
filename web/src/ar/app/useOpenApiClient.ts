/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { useRef } from 'react'
import { HARServiceAPIClient } from '@harnessio/react-har-service-client'
import { SSCAManagerAPIClient } from '@harnessio/react-ssca-manager-client'
import { NGManagerServiceAPIClient } from '@harnessio/react-ng-manager-client'
import { HARServiceV2APIClient } from '@harnessio/react-har-service-v2-client'

import type { CustomUtils } from '@ar/MFEAppTypes'

interface useOpenApiClientProps {
  on401: () => void
  customUtils: CustomUtils
}

export default function useOpenApiClient({ on401, customUtils }: useOpenApiClientProps) {
  const responseInterceptor = (response: Response): Response => {
    if (!response.ok && response.status === 401) {
      on401()
    }
    return response
  }

  const requestInterceptor = (request: Request) => {
    request.headers.delete('Authorization')
    // add custom headers if available
    const customHeader = customUtils.getCustomHeaders()
    Object.entries(customHeader).map(([key, value]) => {
      request.headers.set(key, value)
    })
    return request
  }

  useRef<HARServiceAPIClient>(
    new HARServiceAPIClient({
      responseInterceptor,
      requestInterceptor,
      urlInterceptor: (url: string) => {
        // TODO: remove this once we release new version of ngui
        const apiPrefix = customUtils.getApiBaseUrl('').replace(/\/api\/v1\/?$/, '')
        return `${apiPrefix}/api/v1${url}`
      }
    })
  )

  useRef<HARServiceV2APIClient>(
    new HARServiceV2APIClient({
      responseInterceptor,
      requestInterceptor,
      urlInterceptor: (url: string) => {
        // TODO: remove this once we release new version of ngui
        const apiPrefix = customUtils.getApiBaseUrl('').replace(/\/api\/v1\/?$/, '')
        return `${apiPrefix}/api/v2${url}`
      }
    })
  )

  useRef<SSCAManagerAPIClient>(
    new SSCAManagerAPIClient({
      responseInterceptor,
      requestInterceptor,
      urlInterceptor: (url: string) => {
        return window.getApiBaseUrl(`/ssca-manager${url}`)
      }
    })
  )

  useRef<NGManagerServiceAPIClient>(
    new NGManagerServiceAPIClient({
      responseInterceptor,
      requestInterceptor,
      urlInterceptor: (url: string) => {
        return window.getApiBaseUrl(url)
      }
    })
  )
}
