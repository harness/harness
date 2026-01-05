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

import React, { createContext, useEffect, useMemo } from 'react'
import { noop } from 'lodash-es'
import { Layout } from '@harnessio/uicore'

import { useAppStore, useBulkDownloadFile, useParentHooks } from '@ar/hooks'
import { PreferenceScope } from '@ar/constants'
import type { AsyncRequest } from './types'
import AsyncDownloadRequest from './AsyncDownloadRequest'

import css from './AsyncDownloadRequestsProvider.module.scss'

const REQUESTS_FROM_LAST_24_HOURS = 24 * 60 * 60 * 1000

type AsyncDownloadRequestsContextType = {
  addRequest: (requestKey: string, registryId: string) => void
  removeRequest: (key: string) => void
}

export const AsyncDownloadRequestsContext = createContext<AsyncDownloadRequestsContextType>({
  addRequest: noop,
  removeRequest: noop
})

function AsyncDownloadRequestsProvider({ children }: { children: React.ReactNode }) {
  const { scope } = useAppStore()
  const { usePreferenceStore } = useParentHooks()
  const isBulkDownloadFileEnabled = useBulkDownloadFile()
  const { accountId } = scope
  const { preference: requests = [], setPreference: setRequests } = usePreferenceStore<AsyncRequest[] | undefined>(
    PreferenceScope.USER,
    'AsyncDownloadRequests'
  )

  const addRequest = (requestKey: string, registryId: string) => {
    const request: AsyncRequest = {
      key: requestKey,
      registryId,
      accountId: accountId as string,
      time: new Date().valueOf()
    }
    setRequests([...(requests || []), request])
  }

  const removeRequest = (key: string) => {
    setRequests((requests || []).filter(r => r.key !== key))
  }

  const allowedRequestsToShowInUI = useMemo(() => {
    return requests?.filter(request => request.accountId === accountId)
  }, [requests, accountId])

  useEffect(() => {
    setRequests(requests?.filter(r => r.time > Date.now() - REQUESTS_FROM_LAST_24_HOURS))
  }, [])

  return (
    <AsyncDownloadRequestsContext.Provider value={{ addRequest, removeRequest }}>
      {children}
      {isBulkDownloadFileEnabled && (
        <Layout.Vertical className={css.requestsContainer} flex={{ alignItems: 'center' }} spacing="medium">
          {allowedRequestsToShowInUI?.map(request => (
            <AsyncDownloadRequest key={request.key} requestKey={request.key} onRemove={removeRequest} />
          ))}
        </Layout.Vertical>
      )}
    </AsyncDownloadRequestsContext.Provider>
  )
}

export default AsyncDownloadRequestsProvider
