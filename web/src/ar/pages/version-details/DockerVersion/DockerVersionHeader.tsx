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
import { Expander } from '@blueprintjs/core'
import { useHistory } from 'react-router-dom'
import { Layout } from '@harnessio/uicore'
import type { ArtifactVersionSummary } from '@harnessio/react-har-service-client'

import { PageType, RepositoryPackageType } from '@ar/common/types'
import type { VersionDetailsPathParams } from '@ar/routes/types'
import { useAppStore, useDecodedParams, useParentHooks, useRoutes } from '@ar/hooks'
import QuarantineBadge from '@ar/components/Badge/QuarantineBadge'
import RepositoryIcon from '@ar/frameworks/RepositoryStep/RepositoryIcon'
import VersionActionsWidget from '@ar/frameworks/Version/VersionActionsWidget'
import SetupClientButton from '@ar/components/SetupClientButton/SetupClientButton'

import AvailablityBadge, { AvailablityBadgeType } from '@ar/components/Badge/AvailablityBadge'
import DockerVersionName from './components/DockerVersionName/DockerVersionName'
import type { DockerVersionDetailsQueryParams } from './types'

interface DockerVersionHeaderProps {
  data: ArtifactVersionSummary
  iconSize?: number
}

export default function DockerVersionHeader(props: DockerVersionHeaderProps): JSX.Element {
  const { iconSize = 40, data } = props
  const { imageName: name, version, packageType, isQuarantined, quarantineReason, isDeleted } = data
  const { isCurrentSessionPublic } = useAppStore()
  const pathParams = useDecodedParams<VersionDetailsPathParams>()
  const { useUpdateQueryParams, useQueryParams } = useParentHooks()
  const { updateQueryParams } = useUpdateQueryParams()
  const { digest, tag } = useQueryParams<DockerVersionDetailsQueryParams>()
  const history = useHistory()
  const routes = useRoutes()

  const handleChangeVersion = (newVersion: string, newTag?: string) => {
    history.push(
      routes.toARVersionDetails({
        repositoryIdentifier: pathParams.repositoryIdentifier,
        artifactIdentifier: pathParams.artifactIdentifier,
        versionIdentifier: newVersion,
        artifactType: pathParams.artifactType,
        tag: newTag
      })
    )
  }

  const handleChangeDigest = (newDigest: string) => {
    updateQueryParams({
      digest: newDigest
    })
  }

  return (
    <Layout.Horizontal spacing="small" flex={{ alignItems: 'center' }}>
      <RepositoryIcon packageType={packageType as RepositoryPackageType} iconProps={{ size: iconSize }} />
      <DockerVersionName
        name={name}
        version={version}
        digest={digest}
        onChangeVersion={handleChangeVersion}
        onChangeDigest={handleChangeDigest}
        isLatestVersion={false}
        tag={tag}
      />
      {isQuarantined && <QuarantineBadge reason={quarantineReason} />}
      <AvailablityBadge type={isDeleted ? AvailablityBadgeType.ARCHIVED : AvailablityBadgeType.AVAILABLE} />
      <Expander />
      <SetupClientButton
        repositoryIdentifier={pathParams.repositoryIdentifier}
        artifactIdentifier={pathParams.artifactIdentifier}
        versionIdentifier={pathParams.versionIdentifier}
        packageType={packageType as RepositoryPackageType}
      />
      {!isCurrentSessionPublic && (
        <VersionActionsWidget
          packageType={RepositoryPackageType.DOCKER}
          repoKey={pathParams.repositoryIdentifier}
          artifactKey={pathParams.artifactIdentifier}
          versionKey={pathParams.versionIdentifier}
          digest={digest}
          pageType={PageType.Details}
          data={data}
        />
      )}
    </Layout.Horizontal>
  )
}
