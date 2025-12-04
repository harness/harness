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

import ComposerVersionDependencyContent from './ComposerVersionDependencyContent'
import {
  ComposerArtifactDetails,
  ComposerArtifactDetailsTabEnum,
  type ComposerVersionDetailsQueryParams
} from '../../types'

export default function ComposerVersionArtifactDetailsPage() {
  const { getString } = useStrings()
  const { useUpdateQueryParams, useQueryParams } = useParentHooks()
  const { updateQueryParams } = useUpdateQueryParams()
  const { detailsTab = ComposerArtifactDetailsTabEnum.ReadMe } = useQueryParams<ComposerVersionDetailsQueryParams>()

  const { data } = useVersionOverview<ComposerArtifactDetails>()
  const metadata = data.metadata

  const handleTabChange = useCallback(
    (nextTab: ComposerArtifactDetailsTabEnum): void => {
      updateQueryParams({ detailsTab: nextTab })
    },
    [updateQueryParams]
  )

  return (
    <Layout.Vertical padding="large" spacing="large">
      <ButtonTabs small bold selectedTabId={detailsTab} onChange={handleTabChange}>
        <ButtonTab
          id={ComposerArtifactDetailsTabEnum.ReadMe}
          icon="document"
          iconProps={{ size: 12 }}
          panel={
            <Container>
              <ReadmeFileContent source={metadata?.readme || getString('noReadme')} />
            </Container>
          }
          title={getString('versionDetails.artifactDetails.tabs.readme')}
        />
        <ButtonTab
          id={ComposerArtifactDetailsTabEnum.Files}
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
          id={ComposerArtifactDetailsTabEnum.Dependencies}
          icon="layers"
          iconProps={{ size: 12 }}
          panel={<ComposerVersionDependencyContent />}
          title={getString('versionDetails.artifactDetails.tabs.dependencies')}
        />
      </ButtonTabs>
    </Layout.Vertical>
  )
}
