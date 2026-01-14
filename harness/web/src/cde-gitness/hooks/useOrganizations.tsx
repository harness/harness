import { useState, useEffect, useCallback } from 'react'
import { useGet } from 'restful-react'
import { useAppContext } from 'AppContext'
import { getConfig } from 'services/config'

interface Organization {
  name: string
  identifier: string
  color?: string
  description: string
}

interface OrganizationResponseItem {
  organizationResponse: {
    organization: {
      identifier: string
      name: string
      description: string
      tags: Record<string, string>
    }
    createdAt: number
    lastModifiedAt: number
    harnessManaged: boolean
  }
  projectsCount: number
  connectorsCount: number
  secretsCount: number
  delegatesCount: number
  templatesCount: number
  admins: Array<any>
  collaborators: Array<any>
}

interface OrganizationsApiResponse {
  status: string
  data: {
    content: OrganizationResponseItem[]
    totalPages: number
    totalItems: number
    pageItemCount: number
    pageSize: number
    pageIndex: number
    empty: boolean
    pageToken: string | null
  }
  metaData: any
  correlationId: string
}

interface UseOrganizationsOptions {
  searchTerm?: string
  pageSize?: number
  sortOrder?: string
}

export const useOrganizations = (options?: UseOrganizationsOptions) => {
  const { accountInfo } = useAppContext()
  const accountId = accountInfo?.identifier || ''
  const [organizations, setOrganizations] = useState<Organization[]>([])
  const [pageIndex, setPageIndex] = useState<number>(0)
  const [hasMore, setHasMore] = useState<boolean>(true)
  const [searchTerm, setSearchTerm] = useState<string | undefined>(options?.searchTerm)
  const [sortOrder, setSortOrder] = useState<string | undefined>(options?.sortOrder)

  const PAGE_SIZE = options?.pageSize || 200

  const { data, loading, error, refetch } = useGet<OrganizationsApiResponse>({
    path: '/api/aggregate/organizations',
    base: getConfig('ng'),
    queryParams: {
      routingId: accountId,
      accountIdentifier: accountId,
      pageIndex,
      pageSize: PAGE_SIZE,
      searchTerm,
      sortOrders: sortOrder || 'lastModifiedAt,DESC'
    },
    lazy: !accountId
  })

  const extractOrganizations = useCallback(
    (apiResponse: OrganizationsApiResponse | undefined | null): Organization[] => {
      if (!apiResponse?.data?.content) {
        return []
      }

      return apiResponse.data.content.map(item => ({
        identifier: item.organizationResponse.organization.identifier,
        name: item.organizationResponse.organization.name,
        description: item.organizationResponse.organization.description
      }))
    },
    []
  )

  useEffect(() => {
    if (data) {
      const extractedOrganizations = extractOrganizations(data)

      if (pageIndex === 0) {
        setOrganizations(extractedOrganizations)
      } else {
        setOrganizations(prevOrgs => [...prevOrgs, ...extractedOrganizations])
      }

      setHasMore(pageIndex < data.data.totalPages - 1)
    }
  }, [data, extractOrganizations, pageIndex])

  useEffect(() => {
    setPageIndex(0)
    setHasMore(true)
    setOrganizations([])
  }, [searchTerm, sortOrder])

  useEffect(() => {
    if (options?.searchTerm !== undefined && options.searchTerm !== searchTerm) {
      setSearchTerm(options.searchTerm)
    }
  }, [options?.searchTerm, searchTerm])

  useEffect(() => {
    if (options?.sortOrder !== undefined && options.sortOrder !== sortOrder) {
      setSortOrder(options.sortOrder)
    }
  }, [options?.sortOrder, sortOrder])

  const loadMore = useCallback(() => {
    if (!loading && hasMore) {
      setPageIndex(prev => prev + 1)
    }
  }, [loading, hasMore])

  const search = useCallback((term: string | undefined) => {
    setSearchTerm(term)
    setPageIndex(0)
    setOrganizations([])
    setHasMore(true)
  }, [])

  return {
    organizations: organizations || [],
    totalPages: data?.data?.totalPages || 0,
    totalItems: data?.data?.totalItems || 0,
    loading,
    error,
    refetch,
    hasMore,
    loadMore,
    searchTerm,
    search,
    sortOrder
  }
}
