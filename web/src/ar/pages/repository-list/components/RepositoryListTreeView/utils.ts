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
import { useAppStore, useRoutes } from '@ar/hooks'
import { RepositoryDetailsTab } from '@ar/pages/repository-details/constants'
import { VersionDetailsTab } from '@ar/pages/version-details/components/VersionDetailsTabs/constants'
import { ITreeNode, NodeTypeEnum, INode } from '@ar/components/TreeView/types'

import type { APIQueryParams } from './types'
import type { TreeViewRepositoryQueryParams } from '../../utils'

const getLoadMoreNodeConfig = (parentNode: ITreeNode, metadata: APIQueryParams): INode => {
  const { page } = metadata
  const pathId = parentNode.id
  return {
    id: `${pathId}/loadMore/${page}`,
    label: '',
    value: '',
    type: NodeTypeEnum.LoadMore,
    metadata
  }
}

export function useRepositoryTreeViewUtils(queryParams: TreeViewRepositoryQueryParams) {
  const routes = useRoutes()
  const history = useHistory()
  const { getString } = useStrings()
  const { scope } = useAppStore()
  const { space } = scope
  const { searchTerm, packageTypes, configType, sort, page } = queryParams

  const getTotalCountNodeConfig = (node: ITreeNode, totalCount: number): INode => {
    const { id: pathId } = node
    return {
      id: `${pathId}/totalCount`,
      label: getString('repositoryList.registryCount', { count: totalCount }),
      value: `${pathId}/totalCount`,
      type: NodeTypeEnum.Header,
      metadata: {
        totalCount: totalCount
      }
    }
  }

  const getErrorNodeNodeConfig = (node: ITreeNode, error: string): INode => {
    const { id: pathId } = node
    return {
      id: `${pathId}/error`,
      label: error,
      value: `${pathId}/error`,
      type: NodeTypeEnum.Error,
      metadata: error
    }
  }

  const fetchDockerDigestList = async (node: ITreeNode, filters?: APIQueryParams): Promise<INode[]> => {
    const { id: pathId, value, metadata } = node
    const { repositoryIdentifier, artifactIdentifier } = metadata
    const registryRef = getSpaceRef(filters?.space ?? space, filters?.repositoryIdentifier ?? repositoryIdentifier)
    try {
      const response = await getDockerArtifactManifests({
        registry_ref: registryRef,
        artifact: encodeRef(artifactIdentifier),
        version: filters?.versionIdentifier ?? value
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
      return data
    } catch (e: any) {
      throw getErrorInfoFromErrorObject(e, true)
    }
  }

  const fetchArtifactVersionList = async (node: ITreeNode, filters?: APIQueryParams): Promise<INode[]> => {
    const { id: pathId, value, metadata } = node
    const { repositoryIdentifier } = metadata
    const registryRef = getSpaceRef(filters?.space ?? space, filters?.repositoryIdentifier ?? repositoryIdentifier)
    try {
      const response = await getAllArtifactVersions({
        registry_ref: registryRef,
        artifact: encodeRef(filters?.artifactIdentifier ?? value),
        queryParams: {
          size: DEFAULT_PAGE_SIZE,
          page: filters?.page ?? page ?? 0,
          search_term: filters?.searchTerm ?? undefined
        }
      })
      const versionList = response.content.data.artifactVersions || []
      const { pageIndex = 0, pageCount = 1 } = response.content.data
      const data: INode[] = versionList.map(each => ({
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
      if (pageCount > pageIndex + 1) {
        data.push(
          getLoadMoreNodeConfig(node, {
            ...filters,
            page: pageIndex + 1,
            size: DEFAULT_PAGE_SIZE,
            repositoryIdentifier,
            artifactIdentifier: filters?.artifactIdentifier ?? value
          })
        )
      }
      return data
    } catch (e: any) {
      throw getErrorInfoFromErrorObject(e, true)
    }
  }

  const fetchArtifactList = async (node: ITreeNode, filters?: APIQueryParams): Promise<INode[]> => {
    const { id: pathId, value } = node
    const registryRef = getSpaceRef(filters?.space ?? space, filters?.repositoryIdentifier ?? value)
    try {
      const response = await getAllArtifactsByRegistry({
        registry_ref: registryRef,
        queryParams: {
          size: DEFAULT_PAGE_SIZE,
          page: filters?.page ?? page ?? 0,
          search_term: filters?.searchTerm ?? undefined
        }
      })
      const artifactList = response.content.data.artifacts || []
      const { pageIndex = 0, pageCount = 1 } = response.content.data
      const data: INode[] = artifactList.map(each => ({
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

      if (pageCount > pageIndex + 1) {
        data.push(
          getLoadMoreNodeConfig(node, {
            ...filters,
            page: pageIndex + 1,
            size: DEFAULT_PAGE_SIZE,
            repositoryIdentifier: filters?.repositoryIdentifier ?? value
          })
        )
      }
      return data
    } catch (e: any) {
      throw getErrorInfoFromErrorObject(e, true)
    }
  }

  const fetchRegistryList = async (node: ITreeNode, filters?: APIQueryParams): Promise<INode[]> => {
    const { id: pathId } = node
    const registryRef = getSpaceRef(filters?.space ?? space)
    const [sortField, sortOrder] = filters?.sort ?? sort?.split(',') ?? []
    const pageNumber = filters?.page ?? page ?? 0
    try {
      const response = await getAllRegistries({
        space_ref: registryRef,
        queryParams: {
          size: DEFAULT_PAGE_SIZE,
          page: pageNumber,
          search_term: filters?.searchTerm ?? searchTerm ?? undefined,
          sort_field: sortField ?? undefined,
          sort_order: sortOrder ?? undefined,
          type: configType ?? undefined,
          package_type: filters?.packageTypes ?? packageTypes?.split(',') ?? undefined
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
      if (pageNumber === 0) {
        data.unshift(getTotalCountNodeConfig(node, itemCount))
      }
      if (pageCount > pageIndex + 1) {
        data.push(
          getLoadMoreNodeConfig(node, {
            ...filters,
            page: pageIndex + 1,
            size: DEFAULT_PAGE_SIZE
          })
        )
      }

      return data
    } catch (e: any) {
      throw getErrorInfoFromErrorObject(e, true)
    }
  }

  const handleNavigateToRepositoryDetials = (node: ITreeNode) => {
    const { value } = node
    const newRoute = routes.toARRepositoryDetailsTab({
      repositoryIdentifier: value,
      tab: RepositoryDetailsTab.CONFIGURATION
    })
    history.push(newRoute)
  }

  const handleNavigateToArtifactDetials = (node: ITreeNode) => {
    const { value, metadata } = node
    const { repositoryIdentifier } = metadata
    const newRoute = routes.toARArtifactDetails({
      repositoryIdentifier,
      artifactIdentifier: value
    })
    history.push(newRoute)
  }

  const handleNavigateToVersionDetails = (node: ITreeNode) => {
    const { value, metadata } = node
    const { artifactIdentifier, repositoryIdentifier } = metadata
    const newRoute = routes.toARVersionDetailsTab({
      repositoryIdentifier,
      artifactIdentifier,
      versionIdentifier: value,
      versionTab: VersionDetailsTab.OVERVIEW
    })
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
        queryParams: { digest: value }
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
    handleNavigateToDigestDetails,
    getErrorNodeNodeConfig
  }
}
