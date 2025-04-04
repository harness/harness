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

import { useVersionOverview } from '@ar/pages/version-details/context/VersionOverviewProvider'
import type { IDependencyList } from '@ar/pages/version-details/components/ArtifactDependencyListTable/types'
import ArtifactDependencyListTable from '@ar/pages/version-details/components/ArtifactDependencyListTable/ArtifactDependencyListTable'

import type { NpmVersionDetailsConfig } from '../../types'

export default function NpmVersionDependencyContent() {
  const { data } = useVersionOverview<NpmVersionDetailsConfig>()
  const versionMetatdata = data.metadata?.versions?.[data.version]
  const dependencies = versionMetatdata?.dependencies || {}

  const dependencyList: IDependencyList = Object.entries(dependencies).map(([key, value]) => ({
    name: key,
    version: value as string
  }))

  return <ArtifactDependencyListTable data={dependencyList} />
}
