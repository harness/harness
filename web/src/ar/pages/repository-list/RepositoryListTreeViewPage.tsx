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
import { Expander } from '@blueprintjs/core'
import { HarnessDocTooltip, Page, GridListToggle, Views } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import { DEFAULT_PAGE_INDEX } from '@ar/constants'
import type { RepositoryPackageType } from '@ar/common/types'
import { RepositoryListViewTypeEnum } from '@ar/contexts/AppStoreContext'
import { useAppStore, useGetRepositoryListViewType, useParentComponents, useParentHooks } from '@ar/hooks'
import PackageTypeSelector from '@ar/components/PackageTypeSelector/PackageTypeSelector'
import TableFilterCheckbox from '@ar/components/TableFilterCheckbox/TableFilterCheckbox'

import { useTreeViewRepositoriesQueryParamOptions } from './utils'
import type { TreeViewRepositoryQueryParams } from './utils'
import { CreateRepository } from './components/CreateRepository/CreateRepository'
import RepositoryTypeSelector from './components/RepositoryTypeSelector/RepositoryTypeSelector'
import RepositoryListTreeView from './components/RepositoryListTreeView/RepositoryListTreeView'

import css from './RepositoryListPage.module.scss'

function RepositoryListTreeViewPage(): JSX.Element {
  const { getString } = useStrings()
  const { NGBreadcrumbs } = useParentComponents()
  const { useQueryParams, useUpdateQueryParams } = useParentHooks()
  const { updateQueryParams } = useUpdateQueryParams<Partial<TreeViewRepositoryQueryParams>>()
  const { setRepositoryListViewType } = useAppStore()
  const repositoryListViewType = useGetRepositoryListViewType()

  const queryParamOptions = useTreeViewRepositoriesQueryParamOptions()
  const queryParams = useQueryParams<TreeViewRepositoryQueryParams>(queryParamOptions)
  const { packageTypes, configType, compact } = queryParams

  return (
    <>
      <Page.Header
        className={css.pageHeader}
        title={
          <div className="ng-tooltip-native">
            <h2 data-tooltip-id="artifactRepositoriesPageHeading">{getString('repositoryList.pageHeading')}</h2>
            <HarnessDocTooltip tooltipId="artifactRepositoriesPageHeading" useStandAlone={true} />
          </div>
        }
        breadcrumbs={<NGBreadcrumbs links={[]} />}
      />
      <Page.SubHeader className={css.subHeader}>
        <div className={css.subHeaderItems}>
          <CreateRepository />
          <RepositoryTypeSelector
            value={configType}
            onChange={val => {
              updateQueryParams({ configType: val, page: DEFAULT_PAGE_INDEX })
            }}
          />
          <PackageTypeSelector
            value={(packageTypes?.split(',') || []) as RepositoryPackageType[]}
            onChange={val => {
              updateQueryParams({ packageTypes: val.join(','), page: DEFAULT_PAGE_INDEX })
            }}
          />
          <Expander />
          <TableFilterCheckbox
            value={compact}
            label={getString('repositoryList.compact')}
            disabled={false}
            onChange={val => {
              updateQueryParams({ compact: val })
            }}
          />
          <GridListToggle
            initialSelectedView={repositoryListViewType === RepositoryListViewTypeEnum.LIST ? Views.LIST : Views.GRID}
            icons={{ left: 'SplitView' }}
            onViewToggle={newView => {
              if (newView === Views.GRID) return
              setRepositoryListViewType(RepositoryListViewTypeEnum.LIST)
            }}
          />
        </div>
      </Page.SubHeader>
      <Page.Body className={css.treeViewPageBody}>
        <RepositoryListTreeView />
      </Page.Body>
    </>
  )
}

export default RepositoryListTreeViewPage
