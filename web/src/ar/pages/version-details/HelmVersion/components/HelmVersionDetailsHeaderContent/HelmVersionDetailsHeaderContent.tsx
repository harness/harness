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
import { Layout } from '@harnessio/uicore'
import { Expander } from '@blueprintjs/core'
import { useHistory } from 'react-router-dom'
import type { ArtifactVersionSummary } from '@harnessio/react-har-service-client'

import type { VersionDetailsPathParams } from '@ar/routes/types'
import { PageType, RepositoryPackageType } from '@ar/common/types'
import QuarantineBadge from '@ar/components/Badge/QuarantineBadge'
import { useDecodedParams, useParentHooks, useRoutes } from '@ar/hooks'
import RepositoryIcon from '@ar/frameworks/RepositoryStep/RepositoryIcon'
import VersionActionsWidget from '@ar/frameworks/Version/VersionActionsWidget'
import SetupClientButton from '@ar/components/SetupClientButton/SetupClientButton'

import AvailablityBadge, { AvailablityBadgeType } from '@ar/components/Badge/AvailablityBadge'
import type { HelmVersionDetailsQueryParams } from '../../types'
import HelmVersionName from './HelmVersionName'

interface HelmVersionDetailsHeaderContentProps {
  data: ArtifactVersionSummary
  iconSize?: number
}

export default function HelmVersionDetailsHeaderContent(props: HelmVersionDetailsHeaderContentProps) {
  const { iconSize = 40, data } = props
  const { imageName: name, version, packageType, isQuarantined, quarantineReason, isDeleted } = data
  const pathParams = useDecodedParams<VersionDetailsPathParams>()
  const { useQueryParams } = useParentHooks()
  const { tag } = useQueryParams<HelmVersionDetailsQueryParams>()
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

  return (
    <Layout.Horizontal spacing="small" flex={{ alignItems: 'center' }}>
      <RepositoryIcon packageType={packageType as RepositoryPackageType} iconProps={{ size: iconSize }} />
      <HelmVersionName name={name} version={version} tag={tag} onChange={handleChangeVersion} />
      {isQuarantined && <QuarantineBadge reason={quarantineReason} />}
      <AvailablityBadge type={isDeleted ? AvailablityBadgeType.ARCHIVED : AvailablityBadgeType.AVAILABLE} />
      <Expander />
      <SetupClientButton
        repositoryIdentifier={pathParams.repositoryIdentifier}
        artifactIdentifier={pathParams.artifactIdentifier}
        versionIdentifier={pathParams.versionIdentifier}
        packageType={packageType as RepositoryPackageType}
      />
      <VersionActionsWidget
        packageType={RepositoryPackageType.HELM}
        repoKey={pathParams.repositoryIdentifier}
        artifactKey={pathParams.artifactIdentifier}
        versionKey={pathParams.versionIdentifier}
        digest={pathParams.versionIdentifier}
        pageType={PageType.Details}
        data={data}
      />
    </Layout.Horizontal>
  )
}
