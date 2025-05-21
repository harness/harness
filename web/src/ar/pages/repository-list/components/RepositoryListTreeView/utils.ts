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

import { useHistory } from 'react-router-dom'
import { omit } from 'lodash-es'
import {
  getAllArtifactsByRegistry,
  getAllArtifactVersions,
  getAllRegistries,
  getDockerArtifactManifests
} from '@harnessio/react-har-service-client'
import { getErrorInfoFromErrorObject } from '@harnessio/uicore'

import { DEFAULT_PAGE_SIZE } from '@ar/constants'
import { useStrings } from '@ar/frameworks/strings'
import { RepositoryPackageType } from '@ar/common/types'
import { getShortDigest } from '@ar/pages/digest-list/utils'
import { encodeRef, getSpaceRef } from '@ar/hooks/useGetSpaceRef'
import { useAppStore, useParentHooks, useRoutes } from '@ar/hooks'
import { RepositoryDetailsTab } from '@ar/pages/repository-details/constants'
import { VersionDetailsTab } from '@ar/pages/version-details/components/VersionDetailsTabs/constants'
import { INodeConfig, ITreeNode, NodeTypeEnum, IFetchDataResult, INode } from '@ar/components/TreeView/types'

import type { IGlobalFilters } from './types'
import type { ArtifactRepositoryListPageQueryParams } from '../../utils'

export function useRepositoryTreeViewUtils() {
  const { scope } = useAppStore()
  const { space = '' } = scope
  const routes = useRoutes()
  const history = useHistory()
  const { getString } = useStrings()
  const { useQueryParams, useUpdateQueryParams } = useParentHooks()
  const queryParams = useQueryParams<Record<string, string>>()

  const { updateQueryParams } = useUpdateQueryParams<Partial<ArtifactRepositoryListPageQueryParams>>()
  const fetchDockerDigestList = async (
    node: ITreeNode,
    _filters?: INodeConfig<IGlobalFilters>
  ): Promise<IFetchDataResult> => {
    const { id: pathId, value, metadata } = node
    const { repositoryIdentifier, artifactIdentifier } = metadata
    const registryRef = getSpaceRef(space, repositoryIdentifier)
    try {
      const response = await getDockerArtifactManifests({
        registry_ref: registryRef,
        artifact: encodeRef(artifactIdentifier),
        version: value
      })
      const digestList = response.content.data.manifests || []
      const data = digestList.map(each => ({
        id: `${pathId}/${each.digest}`,
        label: getShortDigest(each.digest),
        value: each.digest,
        type: NodeTypeEnum.File,
        metadata: {
          ...each,
          repositoryIdentifier,
          artifactIdentifier,
          versionIdentifier: value,
          digestIdentifier: each.digest
        }
      }))
      return {
        data
      }
    } catch (e: any) {
      throw getErrorInfoFromErrorObject(e, true)
    }
  }

  const fetchArtifactVersionList = async (
    node: ITreeNode,
    filters?: INodeConfig<IGlobalFilters>
  ): Promise<IFetchDataResult> => {
    const { id: pathId, value, metadata } = node
    const { page, searchTerm } = filters || {}
    const { repositoryIdentifier } = metadata
    const registryRef = getSpaceRef(space, repositoryIdentifier)
    try {
      const response = await getAllArtifactVersions({
        registry_ref: registryRef,
        artifact: encodeRef(value),
        queryParams: {
          size: DEFAULT_PAGE_SIZE,
          page: page ?? 0,
          search_term: searchTerm ?? undefined
        }
      })
      const versionList = response.content.data.artifactVersions || []
      const data = versionList.map(each => ({
        id: `${pathId}/${each.name}`,
        label: each.name,
        value: each.name,
        type: each.packageType === RepositoryPackageType.DOCKER ? NodeTypeEnum.Folder : NodeTypeEnum.File,
        metadata: {
          ...each,
          repositoryIdentifier,
          artifactIdentifier: value,
          versionIdentifier: each.name
        }
      }))
      const { pageIndex = 0, pageCount = 1 } = response.content.data
      return {
        data,
        pagination: {
          page: pageIndex,
          hasMore: pageIndex + 1 < pageCount
        }
      }
    } catch (e: any) {
      throw getErrorInfoFromErrorObject(e, true)
    }
  }

  const fetchArtifactList = async (
    node: ITreeNode,
    filters?: INodeConfig<IGlobalFilters>
  ): Promise<IFetchDataResult> => {
    const { id: pathId, value } = node
    const { page, searchTerm } = filters || {}
    const registryRef = getSpaceRef(space, value)
    try {
      const response = await getAllArtifactsByRegistry({
        registry_ref: registryRef,
        queryParams: {
          size: DEFAULT_PAGE_SIZE,
          page: page ?? 0,
          search_term: searchTerm ?? undefined
        }
      })
      const artifactList = response.content.data.artifacts || []
      const data = artifactList.map(each => ({
        id: `${pathId}/${encodeURIComponent(each.name)}`,
        label: each.name,
        value: each.name,
        type: NodeTypeEnum.Folder,
        metadata: {
          ...each,
          repositoryIdentifier: value,
          artifactIdentifier: each.name
        }
      }))
      const { pageIndex = 0, pageCount = 1 } = response.content.data
      return {
        data,
        pagination: {
          page: pageIndex,
          hasMore: pageIndex + 1 < pageCount
        }
      }
    } catch (e: any) {
      throw getErrorInfoFromErrorObject(e, true)
    }
  }

  const fetchRegistryList = async (
    node: ITreeNode,
    filters?: INodeConfig<IGlobalFilters>
  ): Promise<IFetchDataResult> => {
    const { id: pathId } = node
    const { page, searchTerm, sort, repositoryTypes, configType } = filters || {}
    const registryRef = getSpaceRef(space)
    const [sortField, sortOrder] = sort?.split(',') || []
    try {
      updateQueryParams({ repositoryTypes, configType, registrySearchTerm: searchTerm, treeSort: sort })
      const response = await getAllRegistries({
        space_ref: registryRef,
        queryParams: {
          size: DEFAULT_PAGE_SIZE,
          page: page ?? 0,
          search_term: searchTerm ?? undefined,
          sort_field: sortField ?? undefined,
          sort_order: sortOrder ?? undefined,
          type: configType ?? undefined,
          package_type: repositoryTypes ?? undefined
        },
        stringifyQueryParamsOptions: {
          arrayFormat: 'repeat'
        }
      })
      const registryList = response.content.data.registries || []
      const { pageIndex = 0, pageCount = 1, itemCount = 0 } = response.content.data
      const data: Array<INode> = registryList.map(each => ({
        id: `${pathId}/${each.identifier}`,
        label: each.identifier,
        value: each.identifier,
        type: NodeTypeEnum.Folder,
        metadata: {
          ...each,
          repositoryIdentifier: each.identifier
        }
      }))
      if (itemCount) {
        data.unshift({
          id: `${pathId}/totalCount`,
          label: getString('repositoryList.registryCount', { count: itemCount }),
          value: `${pathId}/totalCount`,
          type: NodeTypeEnum.Header,
          metadata: {
            totalCount: itemCount
          }
        })
      }

      return {
        data,
        pagination: {
          page: pageIndex,
          hasMore: pageCount > pageIndex + 1
        }
      }
    } catch (e: any) {
      throw getErrorInfoFromErrorObject(e, true)
    }
  }

  const handleNavigateToRepositoryDetials = (node: ITreeNode) => {
    const { value } = node
    const newRoute = routes.toARRepositoryDetailsTab(
      {
        repositoryIdentifier: value,
        tab: RepositoryDetailsTab.CONFIGURATION
      },
      { queryParams: omit(queryParams, 'digest') }
    )
    history.push(newRoute)
  }

  const handleNavigateToArtifactDetials = (node: ITreeNode) => {
    const { value, metadata } = node
    const { repositoryIdentifier } = metadata
    const newRoute = routes.toARArtifactDetails(
      {
        repositoryIdentifier,
        artifactIdentifier: value
      },
      { queryParams: omit(queryParams, 'digest') }
    )
    history.push(newRoute)
  }

  const handleNavigateToVersionDetails = (node: ITreeNode) => {
    const { value, metadata } = node
    const { artifactIdentifier, repositoryIdentifier } = metadata
    const newRoute = routes.toARVersionDetailsTab(
      {
        repositoryIdentifier,
        artifactIdentifier,
        versionIdentifier: value,
        versionTab: VersionDetailsTab.OVERVIEW
      },
      { queryParams: omit(queryParams, 'digest') }
    )
    history.push(newRoute)
  }

  const handleNavigateToDigestDetails = (node: ITreeNode) => {
    const { value, metadata } = node
    const { artifactIdentifier, repositoryIdentifier, versionIdentifier } = metadata
    const newRoute = routes.toARVersionDetailsTab(
      {
        repositoryIdentifier,
        artifactIdentifier,
        versionIdentifier,
        versionTab: VersionDetailsTab.OVERVIEW
      },
      {
        queryParams: {
          ...omit(queryParams, 'digest'),
          digest: value
        }
      }
    )
    history.push(newRoute)
  }

  return {
    fetchDockerDigestList,
    fetchArtifactVersionList,
    fetchArtifactList,
    fetchRegistryList,
    handleNavigateToRepositoryDetials,
    handleNavigateToArtifactDetials,
    handleNavigateToVersionDetails,
    handleNavigateToDigestDetails
  }
}
