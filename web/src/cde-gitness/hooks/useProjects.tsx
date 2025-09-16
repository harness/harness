import { useState, useEffect, useCallback } from 'react'
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
}

interface UseProjectsOptions {
  searchTerm?: string
  pageSize?: number
  orgIdentifiers?: string[]
}

export const useProjects = (options?: UseProjectsOptions) => {
  const { accountInfo } = useAppContext()
  const accountId = accountInfo?.identifier || ''
  const [projects, setProjects] = useState<TransformedProject[]>([])
  const [pageIndex, setPageIndex] = useState<number>(0)
  const [hasMore, setHasMore] = useState<boolean>(true)
  const [searchTerm, setSearchTerm] = useState<string | undefined>(options?.searchTerm)
  const orgIdentifiers = options?.orgIdentifiers || []

  const PAGE_SIZE = options?.pageSize || 500

  const { data, loading, error, refetch } = useGet<ApiResponse>({
    path: '/api/aggregate/projects',
    base: getConfig('ng'),
    queryParams: {
      routingId: accountId,
      accountIdentifier: accountId,
      pageIndex,
      pageSize: PAGE_SIZE,
      searchTerm,
      sortOrders: 'lastModifiedAt,DESC',
      onlyFavorites: false,
      ...(orgIdentifiers.length > 0 ? { orgIdentifier: orgIdentifiers } : {})
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
        name: project.name || project.identifier
      }
    })
  }, [])

  useEffect(() => {
    if (data) {
      const transformedProjects = transformProjects(data)

      if (pageIndex === 0) {
        setProjects(transformedProjects)
      } else {
        setProjects(prevProjects => [...prevProjects, ...transformedProjects])
      }

      // Check if we have more pages to load
      setHasMore(pageIndex < data.data.totalPages - 1)
    }
  }, [data, transformProjects, pageIndex])

  // Reset pagination when search term changes
  useEffect(() => {
    setPageIndex(0)
    setHasMore(true)
    setProjects([])
  }, [searchTerm])

  // Update search term when options change
  useEffect(() => {
    if (options?.searchTerm !== undefined && options.searchTerm !== searchTerm) {
      setSearchTerm(options.searchTerm)
    }
  }, [options?.searchTerm])

  useEffect(() => {
    setPageIndex(0)
    setHasMore(true)
    setProjects([])
  }, [options?.orgIdentifiers])

  const loadMore = useCallback(() => {
    if (!loading && hasMore) {
      setPageIndex(prev => prev + 1)
    }
  }, [loading, hasMore])

  const search = useCallback((term: string | undefined) => {
    setSearchTerm(term)
    setPageIndex(0)
    setProjects([])
    setHasMore(true)
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
