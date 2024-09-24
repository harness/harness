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
import { defaultTo } from 'lodash-es'
import { useParams } from 'react-router-dom'
import { Expander } from '@blueprintjs/core'
import { Button, ButtonVariation, Layout, getErrorInfoFromErrorObject, useToaster } from '@harnessio/uicore'
import { useUpdateArtifactLabelsMutation, type ArtifactSummary } from '@harnessio/react-har-service-client'

import { useGetSpaceRef } from '@ar/hooks'
import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import { useStrings } from '@ar/frameworks/strings/String'
import type { RepositoryPackageType } from '@ar/common/types'
import type { ArtifactDetailsPathParams } from '@ar/routes/types'
import WeeklyDownloads from '@ar/components/PageTitle/WeeklyDownloads'
import CreatedAndModifiedAt from '@ar/components/PageTitle/CreatedAndModifiedAt'
import ArtifactTags from '@ar/components/PageTitle/ArtifactTags'
import NameAndDescription from '@ar/components/PageTitle/NameAndDescription'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'
import { useSetupClientModal } from '@ar/pages/repository-details/hooks/useSetupClientModal/useSetupClientModal'

import RepositoryIcon from '@ar/frameworks/RepositoryStep/RepositoryIcon'
import { ArtifactProviderContext } from '../../context/ArtifactProvider'

import css from './ArtifactDetailsHeader.module.scss'

interface ArtifactDetailsHeaderContentProps {
  iconSize?: number
}

function ArtifactDetailsHeaderContent(props: ArtifactDetailsHeaderContentProps): JSX.Element {
  const { iconSize = 40 } = props
  const { data } = useContext(ArtifactProviderContext)
  const spaceRef = useGetSpaceRef()
  const { getString } = useStrings()
  const { showSuccess, showError, clear } = useToaster()
  const pathParams = useParams<ArtifactDetailsPathParams>()

  const { repositoryIdentifier, artifactIdentifier } = pathParams
  const { packageType, imageName, modifiedAt, createdAt, downloadsCount, labels } = data as ArtifactSummary

  const { mutateAsync: modifyArtifactLabels } = useUpdateArtifactLabelsMutation()

  const handleUpdateArtifactLabels = async (newLabels: string[]) => {
    try {
      const response = await modifyArtifactLabels({
        body: { labels: newLabels },
        registry_ref: spaceRef,
        artifact: encodeRef(artifactIdentifier)
      })
      if (response.content.status === 'SUCCESS') {
        clear()
        showSuccess(getString('artifactDetails.labelsUpdated'))
      }
      return true
    } catch (e: any) {
      showError(getErrorInfoFromErrorObject(e, true))
      return false
    }
  }

  const [showSetupClientModal] = useSetupClientModal({
    repoKey: repositoryIdentifier,
    artifactKey: artifactIdentifier,
    packageType: packageType as RepositoryPackageType
  })

  return (
    <Layout.Vertical spacing="small" className={css.headerContainer}>
      <Layout.Horizontal spacing="small" className={css.horizontalContainer}>
        <RepositoryIcon packageType={packageType as RepositoryPackageType} iconProps={{ size: iconSize }} />
        <Layout.Vertical
          className={css.nameAndDescriptionContainer}
          spacing="small"
          flex={{ justifyContent: 'space-between', alignItems: 'flex-start' }}>
          <NameAndDescription name={imageName} hideDescription />
          <WeeklyDownloads downloads={downloadsCount} label={getString('artifactDetails.downloadsThisWeek')} />
        </Layout.Vertical>
        <Expander />
        <Layout.Vertical
          className={css.actionContainer}
          spacing="small"
          flex={{ justifyContent: 'space-between', alignItems: 'flex-end' }}>
          <Layout.Horizontal spacing="large">
            <CreatedAndModifiedAt createdAt={Number(createdAt)} modifiedAt={Number(modifiedAt)} />
            <Button
              variation={ButtonVariation.PRIMARY}
              text={getString('actions.setupClient')}
              onClick={() => {
                showSetupClientModal()
              }}
              icon="setting"
            />
          </Layout.Horizontal>
          <ArtifactTags
            onChange={handleUpdateArtifactLabels}
            labels={defaultTo(labels, [])}
            placeholder={getString('artifactDetails.artifactLabelInputPlaceholder')}
            permission={{
              permission: PermissionIdentifier.EDIT_ARTIFACT_REGISTRY,
              resource: {
                resourceType: ResourceType.ARTIFACT_REGISTRY,
                resourceIdentifier: repositoryIdentifier
              }
            }}
          />
        </Layout.Vertical>
      </Layout.Horizontal>
    </Layout.Vertical>
  )
}

export default ArtifactDetailsHeaderContent
