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

import React, { useCallback } from 'react'
import { Container, Layout } from '@harnessio/uicore'

import { useParentHooks } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { ButtonTab, ButtonTabs } from '@ar/components/ButtonTabs/ButtonTabs'
import VersionFilesProvider from '@ar/pages/version-details/context/VersionFilesProvider'
import ReadmeFileContent from '@ar/pages/version-details/components/ReadmeFileContent/ReadmeFileContent'
import { useVersionOverview } from '@ar/pages/version-details/context/VersionOverviewProvider'
import ArtifactFilesContent from '@ar/pages/version-details/components/ArtifactFileListTable/ArtifactFilesContent'

import RPMVersionDependencyContent from './RPMVersionDependencyContent'
import { RPMArtifactDetails, RPMArtifactDetailsTabEnum, type RPMVersionDetailsQueryParams } from '../../types'

export default function RPMVersionArtifactDetailsPage() {
  const { getString } = useStrings()
  const { useUpdateQueryParams, useQueryParams } = useParentHooks()
  const { updateQueryParams } = useUpdateQueryParams()
  const { detailsTab = RPMArtifactDetailsTabEnum.ReadMe } = useQueryParams<RPMVersionDetailsQueryParams>()

  const { data } = useVersionOverview<RPMArtifactDetails>()
  const { metadata } = data

  const handleTabChange = useCallback(
    (nextTab: RPMArtifactDetailsTabEnum): void => {
      updateQueryParams({ detailsTab: nextTab })
    },
    [updateQueryParams]
  )

  return (
    <Layout.Vertical padding="large" spacing="large">
      <ButtonTabs small bold selectedTabId={detailsTab} onChange={handleTabChange}>
        <ButtonTab
          id={RPMArtifactDetailsTabEnum.ReadMe}
          icon="document"
          iconProps={{ size: 12 }}
          panel={
            <Container>
              <ReadmeFileContent source={metadata?.version_metadata?.description || getString('noReadme')} />
            </Container>
          }
          title={getString('versionDetails.artifactDetails.tabs.readme')}
        />
        <ButtonTab
          id={RPMArtifactDetailsTabEnum.Files}
          icon="document"
          iconProps={{ size: 12 }}
          panel={
            <VersionFilesProvider>
              <ArtifactFilesContent />
            </VersionFilesProvider>
          }
          title={getString('versionDetails.artifactDetails.tabs.files')}
        />
        <ButtonTab
          id={RPMArtifactDetailsTabEnum.Dependencies}
          icon="layers"
          iconProps={{ size: 12 }}
          panel={<RPMVersionDependencyContent />}
          title={getString('versionDetails.artifactDetails.tabs.dependencies')}
        />
      </ButtonTabs>
    </Layout.Vertical>
  )
}
