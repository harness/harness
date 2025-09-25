import { useState, useEffect, useCallback, useRef } from 'react'
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
  organization?: {
    identifier: string
    name: string
    description: string
    tags: Record<string, string>
  }
}

interface UseProjectsOptions {
  searchTerm?: string
  pageSize?: number
  orgIdentifier?: string
  sortOrder?: string
}

export const useProjects = (options?: UseProjectsOptions) => {
  const { accountInfo } = useAppContext()
  const accountId = accountInfo?.identifier || ''
  const [projects, setProjects] = useState<TransformedProject[]>([])
  const [pageIndex, setPageIndex] = useState(0)
  const [hasMore, setHasMore] = useState(true)
  const [searchTerm, setSearchTerm] = useState(options?.searchTerm)
  const [orgIdentifier, setOrgIdentifier] = useState(options?.orgIdentifier)
  const [sortOrder, setSortOrder] = useState(options?.sortOrder)
  const [totalPages, setTotalPages] = useState(0)
  const [totalItems, setTotalItems] = useState(0)
  const debounceTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const isInitialLoadRef = useRef(true)
  const [shouldFetch, setShouldFetch] = useState(false)
  const [isLoadMore, setIsLoadMore] = useState(false)

  const PAGE_SIZE = options?.pageSize || 200

  const { data, loading, error } = useGet<ApiResponse>({
    path: '/api/aggregate/projects',
    base: getConfig('ng'),
    queryParamStringifyOptions: { arrayFormat: 'repeat' },
    queryParams: {
      routingId: accountId,
      accountIdentifier: accountId,
      pageIndex,
      pageSize: PAGE_SIZE,
      searchTerm,
      orgIdentifier,
      sortOrders: sortOrder || 'lastModifiedAt,DESC',
      onlyFavorites: false
    },
    lazy: !accountId || !shouldFetch
  })

  const transformProjects = useCallback((apiData: ApiResponse | undefined): TransformedProject[] => {
    if (!apiData?.data?.content) return []

    return apiData.data.content.map(item => {
      const project = item.projectResponse.project
      return {
        ...project,
        identifier: project.identifier,
        name: project.name || project.identifier,
        fullIdentifier: `${project.orgIdentifier}/${project.identifier}`,
        organization: item.organization
      }
    })
  }, [])

  useEffect(() => {
    if (data) {
      const transformedProjects = transformProjects(data)

      if (isLoadMore) {
        setProjects(prevProjects => [...prevProjects, ...transformedProjects])
        setIsLoadMore(false)
      } else {
        setProjects(transformedProjects)
      }

      setTotalPages(data.data.totalPages)
      setTotalItems(data.data.totalItems)
      setHasMore(pageIndex < data.data.totalPages - 1)
    }
  }, [data, transformProjects, pageIndex, isLoadMore])

  const makeApiCall = useCallback(
    (params: {
      pageIndex: number
      searchTerm?: string
      orgIdentifier?: string
      sortOrder?: string
      isLoadMore?: boolean
    }) => {
      if (!accountId) return

      setIsLoadMore(params.isLoadMore || false)

      setPageIndex(prevPageIndex => (params.pageIndex !== prevPageIndex ? params.pageIndex : prevPageIndex))

      setSearchTerm(prevSearchTerm => (params.searchTerm !== prevSearchTerm ? params.searchTerm : prevSearchTerm))

      setOrgIdentifier(prevOrgIdentifier =>
        params.orgIdentifier !== prevOrgIdentifier ? params.orgIdentifier : prevOrgIdentifier
      )

      setSortOrder(prevSortOrder => (params.sortOrder !== prevSortOrder ? params.sortOrder : prevSortOrder))

      setShouldFetch(true)
    },
    [accountId]
  )

  const debouncedApiCall = useCallback(
    (
      params: {
        pageIndex: number
        searchTerm?: string
        orgIdentifier?: string
        sortOrder?: string
        isLoadMore?: boolean
      },
      delay = 300
    ) => {
      if (debounceTimeoutRef.current) {
        clearTimeout(debounceTimeoutRef.current)
      }

      debounceTimeoutRef.current = setTimeout(() => {
        makeApiCall(params)
      }, delay)
    },
    [makeApiCall]
  )

  useEffect(() => {
    if (accountId && isInitialLoadRef.current) {
      isInitialLoadRef.current = false

      setSearchTerm(options?.searchTerm)
      setOrgIdentifier(options?.orgIdentifier)
      setSortOrder(options?.sortOrder)
      setPageIndex(0)
      setShouldFetch(true)
    }
  }, [accountId, options?.searchTerm, options?.orgIdentifier, options?.sortOrder])

  useEffect(() => {
    if (isInitialLoadRef.current) return

    const searchChanged = options?.searchTerm !== searchTerm
    const orgChanged = options?.orgIdentifier !== orgIdentifier
    const sortChanged = options?.sortOrder !== sortOrder

    if (searchChanged || orgChanged || sortChanged) {
      setPageIndex(0)
      setHasMore(true)

      const delay = searchChanged ? 500 : 100

      debouncedApiCall(
        {
          pageIndex: 0,
          searchTerm: options?.searchTerm,
          orgIdentifier: options?.orgIdentifier,
          sortOrder: options?.sortOrder
        },
        delay
      )
    }
  }, [
    options?.searchTerm,
    options?.orgIdentifier,
    options?.sortOrder,
    searchTerm,
    orgIdentifier,
    sortOrder,
    debouncedApiCall
  ])

  // Cleanup timeout on unmount
  useEffect(() => {
    return () => {
      if (debounceTimeoutRef.current) {
        clearTimeout(debounceTimeoutRef.current)
      }
    }
  }, [])

  const loadMore = useCallback(() => {
    if (!loading && hasMore) {
      const nextPageIndex = pageIndex + 1

      makeApiCall({
        pageIndex: nextPageIndex,
        searchTerm: searchTerm,
        orgIdentifier: orgIdentifier,
        sortOrder: sortOrder,
        isLoadMore: true
      })
    }
  }, [loading, hasMore, pageIndex, searchTerm, orgIdentifier, sortOrder, makeApiCall])

  const search = useCallback(
    (term: string | undefined) => {
      setPageIndex(0)
      setHasMore(true)

      debouncedApiCall(
        {
          pageIndex: 0,
          searchTerm: term,
          orgIdentifier: orgIdentifier,
          sortOrder: sortOrder
        },
        500
      )
    },
    [orgIdentifier, sortOrder, debouncedApiCall]
  )

  const handleRefetch = useCallback(() => {
    makeApiCall({
      pageIndex: pageIndex,
      searchTerm: searchTerm,
      orgIdentifier: orgIdentifier,
      sortOrder: sortOrder
    })
  }, [pageIndex, searchTerm, orgIdentifier, sortOrder, makeApiCall])

  return {
    projects,
    totalPages,
    totalItems,
    loading,
    error,
    refetch: handleRefetch,
    hasMore,
    loadMore,
    searchTerm,
    search,
    sortOrder
  }
}
