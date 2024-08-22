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

import React, { createContext } from 'react'
import type { FC, PropsWithChildren } from 'react'
import { noop } from 'lodash-es'
import { Page } from '@harnessio/uicore'
import { useGetRegistryQuery } from '@harnessio/react-har-service-client'

import { useGetSpaceRef } from '@ar/hooks'
import type { UpstreamRegistry } from '../types'

export interface UpstreamProxyConfigurationFormContextProps {
  data: UpstreamRegistry | undefined
  refetch: () => void
}

export const UpstreamProxyConfigurationFormContext = createContext<UpstreamProxyConfigurationFormContextProps>({
  data: undefined,
  refetch: noop
})

interface UpstreamProxyConfigurationFormProviderProps {
  repoKey: string
  className?: string
}

const UpstreamProxyConfigurationFormProvider: FC<PropsWithChildren<UpstreamProxyConfigurationFormProviderProps>> = ({
  children,
  repoKey,
  className
}): JSX.Element => {
  const spaceRef = useGetSpaceRef(repoKey)

  const {
    data: response,
    error,
    isFetching: loading,
    refetch
  } = useGetRegistryQuery({
    registry_ref: spaceRef
  })

  return (
    <UpstreamProxyConfigurationFormContext.Provider
      value={{ data: response?.content?.data as UpstreamProxyConfigurationFormContextProps['data'], refetch }}>
      <Page.Body className={className} loading={loading} error={error?.message} retryOnError={() => refetch()}>
        {loading ? null : children}
      </Page.Body>
    </UpstreamProxyConfigurationFormContext.Provider>
  )
}

export default UpstreamProxyConfigurationFormProvider
