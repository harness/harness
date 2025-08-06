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

import React, { createContext, type FC, type PropsWithChildren } from 'react'
import { ArtifactSummary, useGetArtifactSummaryQuery } from '@harnessio/react-har-service-client'
import { PageError, PageSpinner } from '@harnessio/uicore'

import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import { useDecodedParams, useGetSpaceRef } from '@ar/hooks'
import type { ArtifactDetailsPathParams } from '@ar/routes/types'
import { LocalArtifactType } from '@ar/pages/repository-details/constants'

export interface ArtifactProviderProps {
  data: ArtifactSummary | undefined
  isReadonly: boolean
  refetch: () => void
}

export const ArtifactProviderContext = createContext<ArtifactProviderProps>({} as ArtifactProviderProps)

const ArtifactProvider: FC<PropsWithChildren<{ repoKey?: string; artifact?: string }>> = ({
  children,
  repoKey,
  artifact
}): JSX.Element => {
  const { repositoryIdentifier, artifactIdentifier, artifactType } = useDecodedParams<ArtifactDetailsPathParams>()
  const spaceRef = useGetSpaceRef(repoKey ?? repositoryIdentifier)
  const {
    data,
    isFetching: loading,
    error,
    refetch
  } = useGetArtifactSummaryQuery({
    registry_ref: spaceRef,
    artifact: encodeRef(artifact ?? artifactIdentifier),
    queryParams: {
      artifact_type: artifactType === LocalArtifactType.ARTIFACTS ? undefined : artifactType
    }
  })

  const responseData = data?.content?.data
  return (
    <ArtifactProviderContext.Provider value={{ data: responseData, isReadonly: false, refetch }}>
      {loading ? <PageSpinner /> : null}
      {error && !loading ? <PageError message={error.message} onClick={() => refetch()} /> : null}
      {!error && !loading ? children : null}
    </ArtifactProviderContext.Provider>
  )
}

export default ArtifactProvider
