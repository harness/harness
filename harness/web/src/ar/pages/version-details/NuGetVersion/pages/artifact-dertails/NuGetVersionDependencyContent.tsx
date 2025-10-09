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

import React, { useMemo } from 'react'
import { Collapse, Layout, Text } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'

import { useStrings } from '@ar/frameworks/strings'
import { useVersionOverview } from '@ar/pages/version-details/context/VersionOverviewProvider'
import ArtifactDependencyListTable from '@ar/pages/version-details/components/ArtifactDependencyListTable/ArtifactDependencyListTable'

import type { IDependencyGroup, NugetArtifactDetails } from '../../types'

import css from './NugetArtifactDetails.module.scss'

export default function NuGetVersionDependencyContent() {
  const { getString } = useStrings()
  const { data } = useVersionOverview<NugetArtifactDetails>()

  const dependencies = useMemo(() => {
    const list: IDependencyGroup[] = []
    if (data.metadata?.metadata.dependencies?.dependencies) {
      list.push({
        dependencies: data.metadata.metadata.dependencies.dependencies,
        targetFramework: getString('all')
      })
    }
    if (data.metadata?.metadata.dependencies?.groups) {
      list.push(...data.metadata.metadata.dependencies.groups)
    }
    return list
  }, [data])

  if (!dependencies.length) {
    return <Text>{getString('versionDetails.noDependencies')}</Text>
  }

  return (
    <Layout.Vertical spacing="medium">
      {dependencies.map(each => (
        <Collapse
          key={each.targetFramework}
          expandedIcon="chevron-down"
          collapsedIcon="chevron-right"
          collapseClassName={css.collapseMain}
          isOpen
          heading={<Text font={{ variation: FontVariation.H4 }}>{each.targetFramework}</Text>}>
          {!each.dependencies?.length && (
            <Text margin={{ top: 'medium' }}>{getString('versionDetails.noDependencies')}</Text>
          )}
          {each.dependencies?.length && (
            <ArtifactDependencyListTable
              data={each.dependencies.map(e => ({
                name: e.id,
                version: e.version
              }))}
            />
          )}
        </Collapse>
      ))}
    </Layout.Vertical>
  )
}
