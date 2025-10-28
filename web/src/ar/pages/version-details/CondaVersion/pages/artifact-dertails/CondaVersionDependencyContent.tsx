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

import type { CondaArtifactDetails } from '../../types'

export default function CondaVersionDependencyContent() {
  const { data } = useVersionOverview<CondaArtifactDetails>()
  const metadata = data.metadata?.metadata
  const deps = metadata?.depends

  // ["python >=3.13,<3.14.0a0", "python_abi 3.13.* *_cp313"]
  const dependencies = useMemo(() => {
    if (!deps) return []
    return deps.reduce((acc: IDependencyList, dep) => {
      const parts = dep.split(' ')
      const name = parts[0]
      const version_req = parts.slice(1).join(' ')
      return [...acc, { name: name, version: version_req }]
    }, [])
  }, [deps])

  return <ArtifactDependencyListTable data={dependencies} />
}
