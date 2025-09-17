import { useState, useEffect, useCallback, useMemo } from 'react'
import { useGet } from 'restful-react'
import { useAppContext } from 'AppContext'
import { getConfig } from 'services/config'

interface ProjectData {
  orgIdentifier: string
  identifier: string
  name: string
  color: string
  description: string
  modules: string[]
}

interface ProjectResponse {
  projectResponse: {
    project: ProjectData
    createdAt: number
    lastModifiedAt: number
    isFavorite: boolean
  }
  organization: {
    identifier: string
    name: string
    description: string
    tags: Record<string, string>
  }
  harnessManagedOrg: boolean
  admins: Array<Record<string, any>>
  collaborators: Array<Record<string, any>>
}

interface ApiResponse {
  status: string
  data: {
    totalPages: number
    totalItems: number
    pageItemCount: number
    pageSize: number
    content: ProjectResponse[]
    pageIndex: number
    empty: boolean
    pageToken: string | null
  }
  metaData: any
  correlationId: string
}

interface TransformedProject {
  identifier: string
  name: string
  orgIdentifier: string
  color: string
  description: string
  modules: string[]
  fullIdentifier: string
}

interface UseProjectsOptions {
  searchTerm?: string
  pageSize?: number
}

export const useProjects = (options?: UseProjectsOptions) => {
  const { accountInfo } = useAppContext()
  const accountId = accountInfo?.identifier || ''
  const [pageIndex, setPageIndex] = useState<number>(0)
  const [hasMore, setHasMore] = useState<boolean>(true)
  const [searchTerm, setSearchTerm] = useState<string | undefined>(options?.searchTerm)

  const [currentParams, setCurrentParams] = useState({
    pageIndex: 0,
    searchTerm: searchTerm
  })

  const PAGE_SIZE = options?.pageSize || 200

  const { data, loading, error, refetch } = useGet<ApiResponse>({
    path: '/api/aggregate/projects',
    base: getConfig('ng'),
    queryParams: {
      routingId: accountId,
      accountIdentifier: accountId,
      pageIndex: currentParams.pageIndex,
      pageSize: PAGE_SIZE,
      searchTerm: currentParams.searchTerm,
      sortOrders: 'lastModifiedAt,DESC',
      onlyFavorites: false
    },
    queryParamStringifyOptions: { arrayFormat: 'repeat' },
    lazy: !accountId
  })

  const transformProjects = useCallback((apiData: ApiResponse | undefined): TransformedProject[] => {
    if (!apiData?.data?.content) return []

    return apiData.data.content.map(item => {
      const project = item.projectResponse.project
      return {
        ...project,
        identifier: project.identifier,
        name: project.name || project.identifier,
        fullIdentifier: `${project.orgIdentifier}/${project.identifier}`
      }
    })
  }, [])

  const projects = useMemo(() => {
    if (!data) return []
    return transformProjects(data)
  }, [data, transformProjects])

  useEffect(() => {
    if (options?.searchTerm !== undefined && options.searchTerm !== currentParams.searchTerm) {
      setSearchTerm(options.searchTerm)
      setCurrentParams(prev => ({
        ...prev,
        searchTerm: options.searchTerm,
        pageIndex: 0
      }))
    }
  }, [options?.searchTerm])

  useEffect(() => {
    if (pageIndex !== currentParams.pageIndex) {
      setCurrentParams(prev => ({
        ...prev,
        pageIndex
      }))
    }
  }, [pageIndex])

  useEffect(() => {
    if (data) {
      setHasMore(currentParams.pageIndex < data.data.totalPages - 1)
    }
  }, [data, currentParams.pageIndex])

  const loadMore = useCallback(() => {
    if (!loading && hasMore) {
      setPageIndex(prev => prev + 1)
    }
  }, [loading, hasMore])

  const search = useCallback((term: string | undefined) => {
    setSearchTerm(term)
    setPageIndex(0)
    setHasMore(true)

    setCurrentParams(prev => ({
      ...prev,
      searchTerm: term,
      pageIndex: 0
    }))
  }, [])

  return {
    projects,
    totalPages: data?.data?.totalPages || 0,
    totalItems: data?.data?.totalItems || 0,
    loading,
    error,
    refetch,
    hasMore,
    loadMore,
    searchTerm,
    search
  }
}
