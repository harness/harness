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
import ArtifactDependencyListTable from '@ar/pages/version-details/components/ArtifactDependencyListTable/ArtifactDependencyListTable'

import type { HuggingfaceArtifactDetails } from '../../types'

export default function HuggingfaceVersionDependencyContent() {
  const { data } = useVersionOverview<HuggingfaceArtifactDetails>()
  const { metadata } = data

  const dependencies = useMemo(() => {
    const _dependencies = metadata?.dependencies || []
    return _dependencies.map(dep => {
      const { name, version } = dep
      return { name, version }
    })
  }, [metadata])

  return <ArtifactDependencyListTable data={dependencies} />
}
