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
import { useGetDockerArtifactManifestsQuery } from '@harnessio/react-har-service-client'

import { useGetSpaceRef } from '@ar/hooks'
import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import { useStrings } from '@ar/frameworks/strings/String'
import DigestListTable from './components/DigestListTable/DigestListTable'

import css from './DigestList.module.scss'

interface DigestListPageProps {
  repoKey: string
  artifact: string
  version: string
}

export default function DigestListPage(props: DigestListPageProps): JSX.Element {
  const { repoKey, artifact, version } = props
  const { getString } = useStrings()
  const spaceRef = useGetSpaceRef(repoKey)

  const {
    data,
    isFetching: loading,
    error,
    refetch
  } = useGetDockerArtifactManifestsQuery({
    registry_ref: spaceRef,
    artifact: encodeRef(artifact),
    version
  })

  const responseData = data?.content.data.manifests

  return (
    <Page.Body
      className={css.pageBody}
      loading={loading}
      error={error?.message}
      retryOnError={() => refetch()}
      noData={{
        when: () => !responseData?.length,
        messageTitle: getString('digestList.table.noDigestTitle'),
        message: getString('digestList.table.aboutDigest')
      }}>
      {responseData && <DigestListTable version={version} data={responseData} />}
    </Page.Body>
  )
}
