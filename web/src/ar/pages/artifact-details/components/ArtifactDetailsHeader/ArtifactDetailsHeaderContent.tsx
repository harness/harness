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

import React, { useContext } from 'react'
import { Expander } from '@blueprintjs/core'
import { Layout } from '@harnessio/uicore'
import type { ArtifactSummary } from '@harnessio/react-har-service-client'

import { useDecodedParams } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings/String'
import type { RepositoryPackageType } from '@ar/common/types'
import type { ArtifactDetailsPathParams } from '@ar/routes/types'
import WeeklyDownloads from '@ar/components/PageTitle/WeeklyDownloads'
import CreatedAndModifiedAt from '@ar/components/PageTitle/CreatedAndModifiedAt'
import NameAndDescription from '@ar/components/PageTitle/NameAndDescription'
import SetupClientButton from '@ar/components/SetupClientButton/SetupClientButton'

import RepositoryIcon from '@ar/frameworks/RepositoryStep/RepositoryIcon'
import { ArtifactProviderContext } from '../../context/ArtifactProvider'

import css from './ArtifactDetailsHeader.module.scss'

interface ArtifactDetailsHeaderContentProps {
  iconSize?: number
}

function ArtifactDetailsHeaderContent(props: ArtifactDetailsHeaderContentProps): JSX.Element {
  const { iconSize = 40 } = props
  const { data } = useContext(ArtifactProviderContext)
  const { getString } = useStrings()
  const pathParams = useDecodedParams<ArtifactDetailsPathParams>()

  const { repositoryIdentifier, artifactIdentifier } = pathParams
  const { packageType, imageName, modifiedAt, createdAt, downloadsCount } = data as ArtifactSummary

  return (
    <Layout.Vertical spacing="small" className={css.headerContainer}>
      <Layout.Horizontal spacing="small" className={css.horizontalContainer}>
        <RepositoryIcon packageType={packageType as RepositoryPackageType} iconProps={{ size: iconSize }} />
        <Layout.Vertical
          className={css.nameAndDescriptionContainer}
          spacing="small"
          flex={{ justifyContent: 'space-between', alignItems: 'flex-start' }}>
          <NameAndDescription name={imageName} hideDescription />
          <WeeklyDownloads downloads={downloadsCount} label={getString('artifactDetails.totalDownloads')} />
        </Layout.Vertical>
        <Expander />
        <Layout.Vertical
          className={css.actionContainer}
          spacing="small"
          flex={{ justifyContent: 'space-between', alignItems: 'flex-end' }}>
          <Layout.Horizontal spacing="large">
            <CreatedAndModifiedAt createdAt={Number(createdAt)} modifiedAt={Number(modifiedAt)} />
            <SetupClientButton
              repositoryIdentifier={repositoryIdentifier}
              artifactIdentifier={artifactIdentifier}
              packageType={packageType as RepositoryPackageType}
            />
          </Layout.Horizontal>
        </Layout.Vertical>
      </Layout.Horizontal>
    </Layout.Vertical>
  )
}

export default ArtifactDetailsHeaderContent
