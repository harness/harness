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

import { useHistory, useParams } from 'react-router-dom'
import { getAllArtifactVersions } from '@harnessio/react-har-service-client'

import { useRoutes } from '@ar/hooks'
import { PageType } from '@ar/common/types'
import type { ArtifactDetailsPathParams } from '@ar/routes/types'
import { encodeRef, useGetSpaceRef } from '@ar/hooks/useGetSpaceRef'
import { LocalArtifactType, RepositoryDetailsTab } from '@ar/pages/repository-details/constants'
import { queryClient } from '@ar/utils/queryClient'

export function useUtilsForDeleteVersion() {
  const routes = useRoutes()
  const history = useHistory()
  const registryRef = useGetSpaceRef()
  const { repositoryIdentifier, artifactIdentifier, artifactType } = useParams<ArtifactDetailsPathParams>()

  async function handleRedirectToVersionListURL(): Promise<void> {
    try {
      await getAllArtifactVersions({
        registry_ref: registryRef,
        artifact: encodeRef(artifactIdentifier),
        queryParams: {
          size: 1
        }
      })
      queryClient.invalidateQueries(['GetAllArtifactVersions'])
      history.push(
        routes.toARArtifactDetails({
          repositoryIdentifier,
          artifactIdentifier,
          artifactType: artifactType ?? LocalArtifactType.ARTIFACTS
        })
      )
    } catch (e) {
      history.push(
        routes.toARRepositoryDetailsTab({
          repositoryIdentifier,
          tab: RepositoryDetailsTab.PACKAGES
        })
      )
    }
  }

  async function handleRedirectAfterDeleteVersion(pageType: PageType): Promise<void> {
    switch (pageType) {
      case PageType.Details:
      case PageType.Table:
        handleRedirectToVersionListURL()
        return
      default:
        queryClient.invalidateQueries(['GetAllHarnessArtifacts'])
        return
    }
  }

  return {
    handleRedirectAfterDeleteVersion
  }
}
