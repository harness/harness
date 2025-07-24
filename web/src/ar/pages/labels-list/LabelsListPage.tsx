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

import React, { useRef } from 'react'
import { flushSync } from 'react-dom'
import { Expander } from '@blueprintjs/core'
import { ExpandingSearchInput, Button, ButtonVariation, type ExpandingSearchInputHandle, Page } from '@harnessio/uicore'

import { useGetSpaceRef, useParentHooks } from '@ar/hooks'
import { DEFAULT_PAGE_INDEX } from '@ar/constants'
import { useStrings } from '@ar/frameworks/strings'

import { useListSpaceLabels } from 'services/code'
import { useParsePaginationInfo } from 'components/ResourceListingPagination/ResourceListingPagination'

import type { PaginationResponse } from './types'
import LabelsListTable from './LabelsListTable'
import { LabelsListPageQueryParams, useLabelsQueryParamOptions } from './utils'
import CreateLabelButton from './components/CreateLabelButton/CreateLabelButton'

import css from './LabelsListPage.module.scss'

function LabelsListPage() {
  const { getString } = useStrings()
  const { useQueryParams, useUpdateQueryParams } = useParentHooks()

  const searchRef = useRef({} as ExpandingSearchInputHandle)
  const queryParamOptions = useLabelsQueryParamOptions()
  const queryParams = useQueryParams<LabelsListPageQueryParams>(queryParamOptions)
  const { searchTerm } = queryParams
  const { updateQueryParams } = useUpdateQueryParams<Partial<LabelsListPageQueryParams>>()

  const spaceRef = useGetSpaceRef()

  const { data, loading, error, refetch, response } = useListSpaceLabels({
    space_ref: spaceRef,
    queryParams: {
      page: queryParams.page,
      limit: queryParams.size,
      query: searchTerm,
      inherited: false
    }
  })
  const pagination: PaginationResponse = useParsePaginationInfo(response)

  const handleClearFilters = (): void => {
    flushSync(searchRef.current.clear)
    updateQueryParams({
      page: DEFAULT_PAGE_INDEX,
      searchTerm: undefined
    })
  }

  const hasFilter = !!searchTerm
  return (
    <>
      <Page.SubHeader className={css.subHeader}>
        <div className={css.subHeaderItems}>
          <CreateLabelButton refetch={refetch} />
          <Expander />
          <ExpandingSearchInput
            alwaysExpanded
            width={200}
            placeholder={getString('search')}
            onChange={text => {
              updateQueryParams({ searchTerm: text || undefined, page: DEFAULT_PAGE_INDEX })
            }}
            defaultValue={searchTerm}
            ref={searchRef}
          />
        </div>
      </Page.SubHeader>
      <Page.Body
        className={css.pageBody}
        loading={loading}
        error={error?.message}
        retryOnError={() => refetch()}
        noData={{
          when: () => !pagination.totalItems, // TODO: change to itemCount once BE fixes the issue with paginated response
          icon: 'label',
          // image: getEmptyStateIllustration(hasFilter, module),
          messageTitle: hasFilter ? getString('noResultsFound') : getString('labelsList.table.noData'),
          button: hasFilter ? (
            <Button text={getString('clearFilters')} variation={ButtonVariation.LINK} onClick={handleClearFilters} />
          ) : (
            <CreateLabelButton refetch={refetch} />
          )
        }}>
        {data && (
          <LabelsListTable
            labels={data}
            pagination={pagination}
            gotoPage={pageNumber => updateQueryParams({ page: pageNumber })}
            onPageSizeChange={newSize => updateQueryParams({ size: newSize, page: DEFAULT_PAGE_INDEX })}
            reload={() => refetch()}
          />
        )}
      </Page.Body>
    </>
  )
}

export default LabelsListPage
