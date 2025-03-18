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

import React, { createContext, useContext, type PropsWithChildren } from 'react'
import { PageError, PageSpinner } from '@harnessio/uicore'
import { type ArtifactDetail, useGetArtifactDetailsQuery } from '@harnessio/react-har-service-client'

import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import { useDecodedParams, useGetSpaceRef } from '@ar/hooks'
import type { VersionDetailsPathParams } from '@ar/routes/types'

interface VersionOverviewProviderProps<T = ArtifactDetail> {
  data: T
  refetch: () => void
}

export const VersionOverviewContext = createContext<VersionOverviewProviderProps>(
  {} as VersionOverviewProviderProps<ArtifactDetail>
)

export const useVersionOverview = <T,>(): VersionOverviewProviderProps<T> => {
  const context = useContext(VersionOverviewContext)
  return context as VersionOverviewProviderProps<T>
}

const VersionOverviewProvider = (props: PropsWithChildren<unknown>) => {
  const pathParams = useDecodedParams<VersionDetailsPathParams>()
  const spaceRef = useGetSpaceRef()

  const {
    data,
    isFetching: loading,
    error,
    refetch
  } = useGetArtifactDetailsQuery({
    registry_ref: spaceRef,
    artifact: encodeRef(pathParams.artifactIdentifier),
    version: pathParams.versionIdentifier,
    queryParams: {}
  })
  const responseData = data?.content?.data

  return (
    <>
      {loading ? <PageSpinner /> : null}
      {error && !loading ? <PageError message={error.message} onClick={() => refetch()} /> : null}
      {!error && !loading && responseData ? (
        <VersionOverviewContext.Provider value={{ data: responseData, refetch }}>
          {props.children}
        </VersionOverviewContext.Provider>
      ) : null}
    </>
  )
}

export default VersionOverviewProvider
