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
import { Button, ButtonVariation, Layout } from '@harnessio/uicore'
import type { ArtifactVersionSummary } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { RepositoryPackageType } from '@ar/common/types'
import type { VersionDetailsPathParams } from '@ar/routes/types'
import { useDecodedParams, useParentHooks, useRoutes } from '@ar/hooks'
import { useSetupClientModal } from '@ar/pages/repository-details/hooks/useSetupClientModal/useSetupClientModal'
import RepositoryIcon from '@ar/frameworks/RepositoryStep/RepositoryIcon'

import DockerVersionName from './components/DockerVersionName/DockerVersionName'
import type { DockerVersionDetailsQueryParams } from './types'

interface DockerVersionHeaderProps {
  data: ArtifactVersionSummary
  iconSize?: number
}

export default function DockerVersionHeader(props: DockerVersionHeaderProps): JSX.Element {
  const { iconSize = 40, data } = props
  const { imageName: name, version, isLatestVersion = false, packageType } = data
  const pathParams = useDecodedParams<VersionDetailsPathParams>()
  const { useUpdateQueryParams, useQueryParams } = useParentHooks()
  const { updateQueryParams } = useUpdateQueryParams()
  const { digest } = useQueryParams<DockerVersionDetailsQueryParams>()
  const { getString } = useStrings()
  const history = useHistory()
  const routes = useRoutes()

  const [showSetupClientModal] = useSetupClientModal({
    repoKey: pathParams.repositoryIdentifier,
    artifactKey: pathParams.artifactIdentifier,
    versionKey: pathParams.versionIdentifier,
    packageType: RepositoryPackageType.DOCKER
  })

  const handleChangeVersion = (newVersion: string) => {
    history.push(
      routes.toARVersionDetails({
        repositoryIdentifier: pathParams.repositoryIdentifier,
        artifactIdentifier: pathParams.artifactIdentifier,
        versionIdentifier: newVersion
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
        isLatestVersion={isLatestVersion}
      />
      <Expander />
      <Button
        variation={ButtonVariation.PRIMARY}
        text={getString('actions.setupClient')}
        icon="setting"
        onClick={() => {
          showSetupClientModal()
        }}
      />
    </Layout.Horizontal>
  )
}
