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

import { DEFAULT_PAGE_INDEX } from '@ar/constants'
import { VersionFilesContext } from '@ar/pages/version-details/context/VersionFilesProvider'
import ArtifactFileListTable from '@ar/pages/version-details/components/ArtifactFileListTable/ArtifactFileListTable'

interface ArtifactFilesContentProps {
  minimal?: boolean
}
export default function ArtifactFilesContent(props: ArtifactFilesContentProps): JSX.Element {
  const { minimal } = props
  const { data, updateQueryParams, sort } = useContext(VersionFilesContext)
  return (
    <ArtifactFileListTable
      data={data}
      gotoPage={pageNumber => updateQueryParams({ page: pageNumber })}
      setSortBy={sortArr => {
        updateQueryParams({ sort: sortArr, page: DEFAULT_PAGE_INDEX })
      }}
      sortBy={sort}
      minimal={minimal}
    />
  )
}
