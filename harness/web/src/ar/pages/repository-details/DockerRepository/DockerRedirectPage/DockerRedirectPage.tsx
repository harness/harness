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

import React, { useEffect } from 'react'
import { useHistory } from 'react-router-dom'
import { getErrorInfoFromErrorObject, useToaster } from '@harnessio/uicore'
import { getDockerArtifactManifests } from '@harnessio/react-har-service-client'

import { useGetOCIVersionType, useGetSpaceRef, useRoutes } from '@ar/hooks'
import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import { useQueryParams } from 'hooks/useQueryParams'

import type { DockerRedirectPageQueryParams } from './types'
import { LocalArtifactType } from '../../constants'

export default function DockerRedirectPage() {
  const { registryId, artifactId, versionId, digest, versionDetailsTab, tag } =
    useQueryParams<DockerRedirectPageQueryParams>()

  const history = useHistory()

  const routes = useRoutes()
  const { clear, showError } = useToaster()
  const registryRef = useGetSpaceRef(registryId)
  const versionType = useGetOCIVersionType()

  const getDefaultDigest = () => {
    if (digest) return digest
    return getDockerArtifactManifests({
      registry_ref: registryRef,
      artifact: artifactId ? encodeRef(artifactId) : '',
      version: versionId ? decodeURIComponent(versionId) : '',
      queryParams: {
        version_type: versionType
      }
    })
      .then(res => {
        const manifests = res.content.data.manifests || []
        if (manifests.length) {
          return manifests[0].digest
        }
      })
      .catch(error => {
        clear()
        showError(getErrorInfoFromErrorObject(error))
      })
  }

  const getRedirectLink = async () => {
    if (registryId && artifactId && versionId && versionDetailsTab) {
      const defaultDigest = await getDefaultDigest()
      const url = routes.toARVersionDetailsTab({
        repositoryIdentifier: registryId,
        artifactIdentifier: artifactId,
        versionIdentifier: versionId,
        versionTab: versionDetailsTab,
        artifactType: LocalArtifactType.ARTIFACTS
      })
      const searchParams = new URLSearchParams()
      if (defaultDigest) searchParams.set('digest', defaultDigest)
      if (tag) searchParams.set('tag', tag)
      return `${url}?${searchParams.toString()}`
    }
    if (registryId && artifactId && versionId) {
      const defaultDigest = await getDefaultDigest()
      const url = routes.toARVersionDetails({
        repositoryIdentifier: registryId,
        artifactIdentifier: artifactId,
        versionIdentifier: versionId,
        artifactType: LocalArtifactType.ARTIFACTS
      })
      const searchParams = new URLSearchParams()
      if (defaultDigest) searchParams.set('digest', defaultDigest)
      if (tag) searchParams.set('tag', tag)
      return `${url}?${searchParams.toString()}`
    }
    if (registryId && artifactId) {
      return routes.toARArtifactDetails({
        repositoryIdentifier: registryId,
        artifactIdentifier: artifactId,
        artifactType: LocalArtifactType.ARTIFACTS
      })
    }
    if (registryId) {
      return routes.toARRepositoryDetails({
        repositoryIdentifier: registryId
      })
    }
    return routes.toARRepositories()
  }

  const init = async () => {
    const url = await getRedirectLink()
    history.replace(url)
  }

  useEffect(() => {
    init()
  }, [])

  return <></>
}
