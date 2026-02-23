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

import React, { useContext, useRef } from 'react'
import { flushSync } from 'react-dom'
import {
  ExpandingSearchInput,
  ExpandingSearchInputHandle,
  Layout,
  Page,
  PageSpinner,
  Button,
  ButtonVariation
} from '@harnessio/uicore'

import { DEFAULT_PAGE_INDEX } from '@ar/constants'
import { useStrings } from '@ar/frameworks/strings'
import { VersionFilesContext } from '@ar/pages/version-details/context/VersionFilesProvider'
import ArtifactFileListTable from '@ar/pages/version-details/components/ArtifactFileListTable/ArtifactFileListTable'

interface ArtifactFilesContentProps {
  minimal?: boolean
}
export default function ArtifactFilesContent(props: ArtifactFilesContentProps): JSX.Element {
  const { minimal } = props
  const { getString } = useStrings()
  const searchRef = useRef({} as ExpandingSearchInputHandle)
  const { data, loading, error, refetch, updateQueryParams, sort, queryParams } = useContext(VersionFilesContext)

  const hasFilter = !!queryParams?.searchTerm

  const handleClearFilters = (): void => {
    flushSync(() => searchRef.current.clear?.())
    updateQueryParams({ searchTerm: undefined, page: DEFAULT_PAGE_INDEX })
  }

  const table = data ? (
    <ArtifactFileListTable
      data={data}
      gotoPage={pageNumber => updateQueryParams({ page: pageNumber })}
      setSortBy={sortArr => {
        updateQueryParams({ sort: sortArr, page: DEFAULT_PAGE_INDEX })
      }}
      sortBy={sort}
      minimal={minimal}
    />
  ) : null

  if (minimal) {
    if (loading) {
      return <PageSpinner />
    }
    return table ?? <></>
  }

  return (
    <Layout.Vertical spacing="medium">
      <ExpandingSearchInput
        ref={searchRef}
        alwaysExpanded
        width={400}
        placeholder={getString('search')}
        onChange={text => {
          updateQueryParams({ searchTerm: text || undefined, page: DEFAULT_PAGE_INDEX })
        }}
        defaultValue={queryParams?.searchTerm ?? ''}
      />
      <Page.Body
        loading={loading}
        error={error?.message}
        retryOnError={() => refetch()}
        noData={{
          when: () => !loading && !data?.files?.length,
          icon: 'document',
          messageTitle: hasFilter
            ? getString('noResultsFound')
            : getString('versionDetails.artifactFiles.noFilesTitle'),
          button: hasFilter ? (
            <Button text={getString('clearFilters')} variation={ButtonVariation.LINK} onClick={handleClearFilters} />
          ) : undefined
        }}>
        {data && table}
      </Page.Body>
    </Layout.Vertical>
  )
}
