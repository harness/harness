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

import { useDecodedParams, useRoutes } from '@ar/hooks'
import type { RepositoryPackageType } from '@ar/common/types'
import type { VersionDetailsPathParams } from '@ar/routes/types'
import RepositoryIcon from '@ar/frameworks/RepositoryStep/RepositoryIcon'
import SetupClientButton from '@ar/components/SetupClientButton/SetupClientButton'
import GenericVersionName from './components/GenericVersionName/GenericVersionName'

interface GenericVersionHeaderProps {
  data: ArtifactVersionSummary
  iconSize?: number
}

export default function GenericVersionHeader(props: GenericVersionHeaderProps): JSX.Element {
  const { iconSize = 40, data } = props
  const { imageName, version, isLatestVersion = false, packageType } = data
  const pathParams = useDecodedParams<VersionDetailsPathParams>()
  const history = useHistory()
  const routes = useRoutes()

  const handleChangeVersion = (newVersion: string) => {
    history.push(
      routes.toARVersionDetails({
        repositoryIdentifier: pathParams.repositoryIdentifier,
        artifactIdentifier: pathParams.artifactIdentifier,
        versionIdentifier: newVersion
      })
    )
  }

  return (
    <Layout.Horizontal spacing="small" flex={{ alignItems: 'center' }}>
      <RepositoryIcon packageType={packageType as RepositoryPackageType} iconProps={{ size: iconSize }} />
      <GenericVersionName
        name={imageName}
        version={version}
        onChangeVersion={handleChangeVersion}
        isLatestVersion={isLatestVersion}
      />
      <Expander />
      <SetupClientButton
        repositoryIdentifier={pathParams.repositoryIdentifier}
        artifactIdentifier={pathParams.artifactIdentifier}
        versionIdentifier={pathParams.versionIdentifier}
        packageType={packageType as RepositoryPackageType}
      />
    </Layout.Horizontal>
  )
}
