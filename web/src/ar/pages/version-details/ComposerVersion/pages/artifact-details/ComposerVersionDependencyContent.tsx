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

import { useVersionOverview } from '@ar/pages/version-details/context/VersionOverviewProvider'
import type { IDependencyList } from '@ar/pages/version-details/components/ArtifactDependencyListTable/types'
import ArtifactDependencyListTable from '@ar/pages/version-details/components/ArtifactDependencyListTable/ArtifactDependencyListTable'

import type { ComposerArtifactDetails } from '../../types'

export default function ComposerVersionDependencyContent() {
  const { data } = useVersionOverview<ComposerArtifactDetails>()
  const metadata = data.metadata

  // create a dependencies array from deps and dev_deps
  const dependencies = useMemo(() => {
    const deps = metadata?.require
    const dev_deps = metadata?.['require-dev']
    const finalDependencies: IDependencyList = []
    if (deps) {
      Object.keys(deps).forEach(key => {
        finalDependencies.push({ name: key, version: deps[key] })
      })
    }
    if (dev_deps) {
      Object.keys(dev_deps).forEach(key => {
        finalDependencies.push({ name: key, version: dev_deps[key] })
      })
    }
    return finalDependencies
  }, [metadata])

  return <ArtifactDependencyListTable data={dependencies} />
}
