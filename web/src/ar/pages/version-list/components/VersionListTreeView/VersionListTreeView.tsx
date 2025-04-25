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
  getAllArtifactVersions,
  type GetAllArtifactVersionsOkResponse,
  type RegistryArtifactMetadata,
  type RegistryMetadata
} from '@harnessio/react-har-service-client'

import { DEFAULT_PAGE_SIZE } from '@ar/constants'
import { useStrings } from '@ar/frameworks/strings'
import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import TreeBody from '@ar/components/TreeView/TreeBody'
import { useGetSpaceRef, useParentHooks } from '@ar/hooks'
import type { RepositoryPackageType } from '@ar/common/types'
import TreeNodeList from '@ar/components/TreeView/TreeNodeList'
import type { NodeSpec } from '@ar/components/TreeView/TreeViewContext'
import TreeLoadMoreNode from '@ar/components/TreeView/TreeLoadMoreNode'
import TreeNodeSearchInput from '@ar/components/TreeView/TreeNodeSearchInput'
import VersionTreeNodeViewWidget from '@ar/frameworks/Version/VersionTreeNodeViewWidget'
import { useVersionListQueryParamOptions, VersionListPageQueryParams } from '../../utils'

interface VersiontListTreeViewProps {
  registryIdentifier: string
  artifactIdentifier: string
  parentNodeLevels: Array<NodeSpec<RegistryMetadata | RegistryArtifactMetadata>>
}

export default function VersionListTreeView(props: VersiontListTreeViewProps) {
  const { registryIdentifier, artifactIdentifier, parentNodeLevels } = props

  const { useQueryParams, useUpdateQueryParams } = useParentHooks()
  const { updateQueryParams } = useUpdateQueryParams<Partial<VersionListPageQueryParams>>()
  const queryParams = useQueryParams<VersionListPageQueryParams>(useVersionListQueryParamOptions())
  const { versionSearchTerm } = queryParams

  const { getString } = useStrings()
  const registryRef = useGetSpaceRef(registryIdentifier)

  const { data, isLoading, error, hasNextPage, fetchNextPage, refetch, isFetchingNextPage } = useInfiniteQuery<
    GetAllArtifactVersionsOkResponse,
    Error,
    GetAllArtifactVersionsOkResponse,
    Array<string | undefined>
  >({
    queryKey: ['versionList', registryIdentifier, artifactIdentifier, versionSearchTerm],
    queryFn: ({ pageParam = 0 }) =>
      getAllArtifactVersions({
        registry_ref: registryRef,
        artifact: encodeRef(artifactIdentifier),
        queryParams: {
          page: pageParam,
          size: DEFAULT_PAGE_SIZE,
          search_term: versionSearchTerm
        },
        stringifyQueryParamsOptions: {
          arrayFormat: 'repeat'
        }
      }),
    getNextPageParam: lastPage => {
      const totalPages = lastPage.content.data.pageCount ?? 0
      const lastPageNumber = lastPage.content.data.pageIndex ?? -1
      return lastPageNumber < totalPages - 1 ? lastPageNumber + 1 : undefined
    }
  })

  const { list: versionList, count: totalVersions } = useMemo(() => {
    const list = data?.pages.flatMap(page => page.content.data.artifactVersions ?? []) || []
    const count = data?.pages[0].content.data.itemCount || 0
    return { list, count }
  }, [data])

  return (
    <TreeNodeList>
      {totalVersions > DEFAULT_PAGE_SIZE && (
        <TreeNodeSearchInput
          level={2}
          defaultValue={versionSearchTerm}
          onChange={val => {
            updateQueryParams({ versionSearchTerm: val })
          }}
        />
      )}
      <TreeBody
        loading={isLoading}
        error={error?.message}
        retryOnError={refetch}
        isEmpty={!versionList.length}
        emptyDataMessage={getString('versionList.table.noVersionsTitle')}>
        {versionList.map((each, idx) => (
          <VersionTreeNodeViewWidget
            key={each.name}
            packageType={each.packageType as RepositoryPackageType}
            artifactIdentifier={artifactIdentifier}
            data={{
              ...each,
              registryIdentifier
            }}
            parentNodeLevels={parentNodeLevels}
            isLastChild={idx === versionList.length - 1}
          />
        ))}
        {hasNextPage && <TreeLoadMoreNode level={2} onClick={() => fetchNextPage()} disabled={isFetchingNextPage} />}
      </TreeBody>
    </TreeNodeList>
  )
}
