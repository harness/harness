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

import { useStrings } from '@ar/frameworks/strings'
import { useParentHooks } from '@ar/hooks'
import { ButtonTabs, ButtonTab } from '@ar/components/ButtonTabs/ButtonTabs'

import { DockerArtifactDetailsTab } from './constants'
import DockerVersionLayersContent from './DockerVersionLayersContent'
import DockerManifestDetailsContent from './DockerManifestDetailsContent'
import type { DockerVersionDetailsQueryParams } from './types'

export default function DockerArtifactDetailsContent(): JSX.Element {
  const { getString } = useStrings()
  const { useUpdateQueryParams, useQueryParams } = useParentHooks()
  const { updateQueryParams } = useUpdateQueryParams()
  const { detailsTab = DockerArtifactDetailsTab.LAYERS } = useQueryParams<DockerVersionDetailsQueryParams>()

  const handleTabChange = useCallback(
    (nextTab: DockerArtifactDetailsTab): void => {
      updateQueryParams({ detailsTab: nextTab })
    },
    [updateQueryParams]
  )

  return (
    <Layout.Vertical padding="large" spacing="large">
      <ButtonTabs small bold selectedTabId={detailsTab} onChange={handleTabChange}>
        <ButtonTab
          id={DockerArtifactDetailsTab.LAYERS}
          icon="layers"
          iconProps={{ size: 12 }}
          panel={<DockerVersionLayersContent />}
          title={getString('versionDetails.artifactDetails.tabs.layers')}
        />
        <ButtonTab
          id={DockerArtifactDetailsTab.MANIFEST}
          icon="document"
          iconProps={{ size: 12 }}
          panel={<DockerManifestDetailsContent />}
          title={getString('versionDetails.artifactDetails.tabs.manifest')}
        />
      </ButtonTabs>
    </Layout.Vertical>
  )
}
