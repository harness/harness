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

import React, { FC, PropsWithChildren, createContext } from 'react'
import classNames from 'classnames'
import { Page } from '@harnessio/uicore'
import { ArtifactVersionSummary, useGetArtifactVersionSummaryQuery } from '@harnessio/react-har-service-client'

import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import { useGetSpaceRef, useParentHooks } from '@ar/hooks'

import type { DockerVersionDetailsQueryParams } from '../DockerVersion/types'

import css from '../VersionDetails.module.scss'

interface VersionProviderProps {
  data: ArtifactVersionSummary | undefined
  isReadonly: boolean
  refetch: () => void
}

export const VersionProviderContext = createContext<VersionProviderProps>({} as VersionProviderProps)

interface VersionProviderSpcs {
  repoKey: string
  artifactKey: string
  versionKey: string
  className?: string
}

const VersionProvider: FC<PropsWithChildren<VersionProviderSpcs>> = ({
  children,
  repoKey,
  artifactKey,
  versionKey,
  className
}): JSX.Element => {
  const spaceRef = useGetSpaceRef(repoKey)
  const { useQueryParams } = useParentHooks()
  const { digest } = useQueryParams<DockerVersionDetailsQueryParams>()

  const {
    data,
    isFetching: loading,
    error,
    refetch
  } = useGetArtifactVersionSummaryQuery({
    registry_ref: spaceRef,
    artifact: encodeRef(artifactKey),
    version: versionKey,
    queryParams: {
      digest
    }
  })

  const responseData = data?.content?.data

  return (
    <VersionProviderContext.Provider value={{ data: responseData, isReadonly: false, refetch }}>
      <Page.Body
        className={classNames(className, {
          [css.pageBody]: !error?.message
        })}
        loading={loading}
        error={error?.message}
        retryOnError={() => refetch()}>
        {loading ? null : children}
      </Page.Body>
    </VersionProviderContext.Provider>
  )
}

export default VersionProvider
