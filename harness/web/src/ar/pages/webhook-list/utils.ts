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

import { useMemo } from 'react'
import type { ListWebhooksQueryQueryParams } from '@harnessio/react-har-service-client'

import { useParentHooks } from '@ar/hooks'
import { DEFAULT_PAGE_INDEX, DEFAULT_PAGE_SIZE } from '@ar/constants'
import type { UseQueryParamsOptions } from '@ar/__mocks__/hooks'

export type WebhookListPageQueryParams = ListWebhooksQueryQueryParams & {
  searchTerm?: string
  sort: string[]
}

export const useListWebhooksQueryParamOptions = (): UseQueryParamsOptions<WebhookListPageQueryParams> => {
  const { useQueryParamsOptions } = useParentHooks()
  const _options = useQueryParamsOptions(
    {
      page: DEFAULT_PAGE_INDEX,
      size: DEFAULT_PAGE_SIZE,
      sort: []
    },
    { ignoreEmptyString: false }
  )
  const options = useMemo(() => ({ ..._options, strictNullHandling: true }), [_options])

  return options
}
