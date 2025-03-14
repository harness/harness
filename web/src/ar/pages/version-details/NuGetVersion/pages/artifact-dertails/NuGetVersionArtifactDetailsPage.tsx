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
import { Layout } from '@harnessio/uicore'

import { useParentHooks } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { ButtonTab, ButtonTabs } from '@ar/components/ButtonTabs/ButtonTabs'
import VersionFilesProvider from '@ar/pages/version-details/context/VersionFilesProvider'
import VersionDependencyListProvider from '@ar/pages/version-details/context/VersionDependencyListProvider'

import NuGetVersionFilesContent from './NuGetVersionFilesContent'
import NuGetVersionDependencyContent from './NuGetVersionDependencyContent'
import { NuGetArtifactDetailsTabEnum, type NuGetVersionDetailsQueryParams } from '../../types'

export default function NuGetVersionArtifactDetailsPage() {
  const { getString } = useStrings()
  const { useUpdateQueryParams, useQueryParams } = useParentHooks()
  const { updateQueryParams } = useUpdateQueryParams()
  const { detailsTab = NuGetArtifactDetailsTabEnum.Files } = useQueryParams<NuGetVersionDetailsQueryParams>()

  const handleTabChange = useCallback(
    (nextTab: NuGetArtifactDetailsTabEnum): void => {
      updateQueryParams({ detailsTab: nextTab })
    },
    [updateQueryParams]
  )

  return (
    <Layout.Vertical padding="large" spacing="large">
      <ButtonTabs small bold selectedTabId={detailsTab} onChange={handleTabChange}>
        <ButtonTab
          id={NuGetArtifactDetailsTabEnum.Files}
          icon="document"
          iconProps={{ size: 12 }}
          panel={
            <VersionFilesProvider>
              <NuGetVersionFilesContent />
            </VersionFilesProvider>
          }
          title={getString('versionDetails.artifactDetails.tabs.files')}
        />
        <ButtonTab
          id={NuGetArtifactDetailsTabEnum.Dependencies}
          icon="layers"
          iconProps={{ size: 12 }}
          panel={
            <VersionDependencyListProvider>
              <NuGetVersionDependencyContent />
            </VersionDependencyListProvider>
          }
          title={getString('versionDetails.artifactDetails.tabs.dependencies')}
        />
      </ButtonTabs>
    </Layout.Vertical>
  )
}
