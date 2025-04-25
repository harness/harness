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

import React, { useMemo, useState } from 'react'
import { compact as lodashCompact } from 'lodash-es'
import { Switch } from 'react-router-dom'
import { useInfiniteQuery } from '@tanstack/react-query'
import { FontVariation } from '@harnessio/design-system'
import { Container, DropDown, Layout, Text } from '@harnessio/uicore'
import { type Error, getAllRegistries, type GetAllRegistriesOkResponse } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { useGetSpaceRef, useParentHooks, useRoutes } from '@ar/hooks'
import TreeBody from '@ar/components/TreeView/TreeBody'
import TreeNode from '@ar/components/TreeView/TreeNode'
import type { RepositoryPackageType } from '@ar/common/types'
import TreeNodeList from '@ar/components/TreeView/TreeNodeList'
import RouteProvider from '@ar/components/RouteProvider/RouteProvider'
import TreeLoadMoreNode from '@ar/components/TreeView/TreeLoadMoreNode'
import RepositoryProvider from '@ar/pages/repository-details/context/RepositoryProvider'
import {
  artifactDetailsPathProps,
  repositoryDetailsPathProps,
  versionDetailsPathParams
} from '@ar/routes/RouteDestinations'
import TreeNodeSearchInput from '@ar/components/TreeView/TreeNodeSearchInput'
import { TreeViewContext } from '@ar/components/TreeView/TreeViewContext'
import VersionProvider from '@ar/pages/version-details/context/VersionProvider'
import ArtifactProvider from '@ar/pages/artifact-details/context/ArtifactProvider'
import type { DockerVersionDetailsQueryParams } from '@ar/pages/version-details/DockerVersion/types'
import RepositoryTreeNodeViewWidget from '@ar/frameworks/RepositoryStep/RepositoryTreeNodeViewWidget'
import VersionTreeNodeDetails from '@ar/pages/version-details/components/VersionTreeNode/VersionTreeNodeDetails'

import ArtifactTreeNodeDetails from '@ar/pages/artifact-details/components/ArtifactTreeNode/ArtifactTreeNodeDetails'
import RepositoryTreeNodeDetails from '@ar/pages/repository-details/components/RepositoryTreeNode/RepositoryTreeNodeDetails'

import { TreeViewSortingOptions } from '../../constants'
import { useArtifactRepositoriesQueryParamOptions, type ArtifactRepositoryListPageQueryParams } from '../../utils'

import css from './RepositoryListTreeView.module.scss'

export default function RepositoryListTreeView() {
  const routeDefinitions = useRoutes(true)
  const { getString } = useStrings()
  const spaceRef = useGetSpaceRef()
  const [initialised, setInitialised] = useState(false)

  const { useQueryParams, useUpdateQueryParams } = useParentHooks()
  const { digest } = useQueryParams<DockerVersionDetailsQueryParams>()
  const [activePath, setActivePath] = useState('')

  const queryParamOptions = useArtifactRepositoriesQueryParamOptions()
  const { updateQueryParams } = useUpdateQueryParams<Partial<ArtifactRepositoryListPageQueryParams>>()
  const { registrySearchTerm, compact, repositoryTypes, configType, treeSort } =
    useQueryParams<ArtifactRepositoryListPageQueryParams>(queryParamOptions)

  const [sortField, sortOrder] = treeSort?.split(',') || []

  const { data, isFetching, error, hasNextPage, fetchNextPage, refetch, isFetchingNextPage } = useInfiniteQuery<
    GetAllRegistriesOkResponse,
    Error,
    GetAllRegistriesOkResponse,
    Array<string | Record<string, unknown>>
  >({
    queryKey: [
      'registryList',
      {
        sortField,
        sortOrder,
        registrySearchTerm,
        configType,
        repositoryTypes
      }
    ],
    queryFn: ({ pageParam = 0 }) =>
      getAllRegistries({
        space_ref: spaceRef,
        queryParams: {
          page: pageParam,
          size: 20,
          sort_field: sortField,
          sort_order: sortOrder,
          package_type: repositoryTypes,
          search_term: registrySearchTerm,
          type: configType
        },
        stringifyQueryParamsOptions: {
          arrayFormat: 'repeat'
        }
      }),
    getNextPageParam: lastPage => {
      const totalPages = lastPage.content.data.pageCount ?? 0
      const lastPageNumber = lastPage.content.data.pageIndex ?? -1
      const nextPage = lastPageNumber + 1
      return nextPage < totalPages ? nextPage : undefined
    }
  })

  const handleUpdateActivePath = (values: Record<string, string>) => {
    const initialActivePath = lodashCompact([
      values.repositoryIdentifier,
      values.artifactIdentifier,
      values.versionIdentifier,
      digest
    ]).join('/')
    setActivePath(initialActivePath)
    setInitialised(true)
  }

  const { list: repositoryList, count: repositoryCount } = useMemo(() => {
    const list = data?.pages.flatMap(page => page.content.data.registries) || []
    const count = data?.pages[0].content.data.itemCount || 0
    return { list, count }
  }, [data])

  return (
    <TreeViewContext.Provider value={{ activePath, setActivePath, compact }}>
      <Layout.Horizontal className={css.treeViewPageContainer}>
        <Container className={css.treeViewContainer}>
          <TreeNodeList>
            <TreeNodeSearchInput
              className={css.searchInput}
              defaultValue={registrySearchTerm}
              onChange={val => {
                updateQueryParams({ registrySearchTerm: val })
              }}
              treeNodeProps={{
                alwaysShowAction: true,
                actionElement: (
                  <DropDown
                    icon="main-sort"
                    className={css.sortingDropDown}
                    items={TreeViewSortingOptions}
                    value={treeSort}
                    onChange={option => {
                      const selectedOption = TreeViewSortingOptions.find(each => each.label === option.label)
                      if (!selectedOption) return
                      const val = selectedOption.key
                      const dir = selectedOption.dir
                      updateQueryParams({ treeSort: [val, dir].join(',') })
                    }}
                    usePortal
                  />
                )
              }}
            />
            {!!repositoryCount && (
              <TreeNode
                disabled
                alwaysShowAction
                compact={compact}
                heading={
                  <Text font={{ variation: FontVariation.BODY, weight: 'semi-bold' }}>
                    {getString('repositoryList.registryCount', { count: repositoryCount })}
                  </Text>
                }
              />
            )}
            <TreeBody
              loading={isFetching || !initialised}
              error={error?.message}
              retryOnError={refetch}
              isEmpty={!repositoryList.length}>
              {repositoryList.map((registry, idx) => (
                <RepositoryTreeNodeViewWidget
                  key={registry.identifier}
                  data={registry}
                  packageType={registry.packageType as RepositoryPackageType}
                  isLastChild={idx === repositoryList.length - 1}
                />
              ))}
              {hasNextPage && <TreeLoadMoreNode onClick={() => fetchNextPage()} disabled={isFetchingNextPage} />}
            </TreeBody>
          </TreeNodeList>
        </Container>
        <Container className={css.treeViewPageContentContainer}>
          <Switch>
            <RouteProvider exact onLoad={handleUpdateActivePath} path={[routeDefinitions.toARRepositories()]}>
              {/* TODO: Implement a default page for this path */}
              <></>
            </RouteProvider>
            <RouteProvider
              exact
              onLoad={handleUpdateActivePath}
              path={[routeDefinitions.toARArtifactDetails({ ...artifactDetailsPathProps })]}>
              <ArtifactProvider>
                <ArtifactTreeNodeDetails />
              </ArtifactProvider>
            </RouteProvider>
            <RouteProvider
              onLoad={handleUpdateActivePath}
              path={[routeDefinitions.toARVersionDetails({ ...versionDetailsPathParams })]}>
              <VersionProvider>
                <VersionTreeNodeDetails />
              </VersionProvider>
            </RouteProvider>
            <RouteProvider
              onLoad={handleUpdateActivePath}
              path={[routeDefinitions.toARRepositoryDetails({ ...repositoryDetailsPathProps })]}>
              <RepositoryProvider>
                <RepositoryTreeNodeDetails />
              </RepositoryProvider>
            </RouteProvider>
          </Switch>
        </Container>
      </Layout.Horizontal>
    </TreeViewContext.Provider>
  )
}
