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

import React from 'react'
import { Page } from '@harnessio/uicore'
import { useGetArtifactDetailsQuery } from '@harnessio/react-har-service-client'

import { useDecodedParams, useGetSpaceRef } from '@ar/hooks'
import type { VersionDetailsPathParams } from '@ar/routes/types'

import type { GenericArtifactDetails } from '../../types'
import GeneralInformationCard from '../overview/GeneralInformationCard'

import css from './ossDetails.module.scss'

export default function OSSGeneralInfoContent() {
  const pathParams = useDecodedParams<VersionDetailsPathParams>()
  const spaceRef = useGetSpaceRef()
  const {
    data,
    isFetching: loading,
    error,
    refetch
  } = useGetArtifactDetailsQuery(
    {
      registry_ref: spaceRef,
      artifact: pathParams.artifactIdentifier,
      version: pathParams.versionIdentifier,
      queryParams: {}
    },
    {
      enabled: false
    }
  )

  const response = data?.content?.data as GenericArtifactDetails

  return (
    <Page.Body
      className={css.pageBody}
      loading={loading}
      error={error?.message || error}
      retryOnError={() => refetch()}>
      {response && <GeneralInformationCard className={css.cardContainer} data={response} />}
    </Page.Body>
  )
}
