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
import { useInfiniteQuery } from '@tanstack/react-query'
import {
  type Error,
  getAllArtifactsByRegistry,
  type GetAllArtifactsByRegistryOkResponse,
  type RegistryMetadata
} from '@harnessio/react-har-service-client'

import { DEFAULT_PAGE_SIZE } from '@ar/constants'
import { useStrings } from '@ar/frameworks/strings'
import TreeBody from '@ar/components/TreeView/TreeBody'
import { useGetSpaceRef, useParentHooks } from '@ar/hooks'
import type { RepositoryPackageType } from '@ar/common/types'
import TreeNodeList from '@ar/components/TreeView/TreeNodeList'
import TreeLoadMoreNode from '@ar/components/TreeView/TreeLoadMoreNode'
import type { NodeSpec } from '@ar/components/TreeView/TreeViewContext'
import TreeNodeSearchInput from '@ar/components/TreeView/TreeNodeSearchInput'
import ArtifactTreeNodeViewWidget from '@ar/frameworks/Version/ArtifactTreeNodeViewWidget'

import {
  type RegistryArtifactListPageQueryParams,
  useRegistryArtifactListQueryParamOptions
} from '../RegistryArtifactListTable/utils'

interface ArtifactListTreeViewProps {
  registryIdentifier: string
  parentNodeLevels: Array<NodeSpec<RegistryMetadata>>
}

export default function ArtifactListTreeView(props: ArtifactListTreeViewProps) {
  const { registryIdentifier } = props
  const { useQueryParams, useUpdateQueryParams } = useParentHooks()
  const { updateQueryParams } = useUpdateQueryParams<Partial<RegistryArtifactListPageQueryParams>>()
  const queryParams = useQueryParams<RegistryArtifactListPageQueryParams>(useRegistryArtifactListQueryParamOptions())
  const { artifactSearchTerm } = queryParams
  const { getString } = useStrings()

  const registryRef = useGetSpaceRef(registryIdentifier)
  const { data, isLoading, error, hasNextPage, fetchNextPage, refetch, isFetchingNextPage } = useInfiniteQuery<
    GetAllArtifactsByRegistryOkResponse,
    Error,
    GetAllArtifactsByRegistryOkResponse,
    Array<string | undefined>
  >({
    queryKey: ['artifactList', registryIdentifier, artifactSearchTerm],
    queryFn: ({ pageParam = 0 }) =>
      getAllArtifactsByRegistry({
        registry_ref: registryRef,
        queryParams: {
          page: pageParam,
          size: DEFAULT_PAGE_SIZE,
          search_term: artifactSearchTerm
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

  const { list: artifactList, count: totalArtifacts } = useMemo(() => {
    const list = data?.pages.flatMap(page => page.content.data.artifacts) || []
    const count = data?.pages[0].content.data.itemCount || 0
    return { list, count }
  }, [data])

  return (
    <TreeNodeList>
      {totalArtifacts > DEFAULT_PAGE_SIZE && (
        <TreeNodeSearchInput
          level={1}
          defaultValue={artifactSearchTerm}
          onChange={val => {
            updateQueryParams({ artifactSearchTerm: val })
          }}
        />
      )}
      <TreeBody
        loading={isLoading}
        error={error?.message}
        retryOnError={refetch}
        isEmpty={!artifactList.length}
        emptyDataMessage={getString('artifactList.table.noArtifactsTitle')}>
        {artifactList.map((each, indx) => (
          <ArtifactTreeNodeViewWidget
            key={each.name}
            packageType={each.packageType as RepositoryPackageType}
            data={each}
            isLastChild={indx === artifactList.length - 1}
            parentNodeLevels={props.parentNodeLevels}
          />
        ))}
        {hasNextPage && <TreeLoadMoreNode level={1} onClick={() => fetchNextPage()} disabled={isFetchingNextPage} />}
      </TreeBody>
    </TreeNodeList>
  )
}
