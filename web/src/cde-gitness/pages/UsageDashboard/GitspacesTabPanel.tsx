import React, { useEffect, useState, useCallback } from 'react'
import { Container, Layout, Page, ExpandingSearchInput, Text, Button } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { ButtonVariation } from '@harnessio/uicore'
import debounce from 'lodash/debounce'
import noSpace from 'cde-gitness/assests/no-gitspace.svg?url'
import { GitspaceOwnerType, GitspaceStatus } from 'cde-gitness/constants'
import type { EnumGitspaceSort } from 'services/cde'
import { useAppContext } from 'AppContext'
import StatusDropdown from 'cde-gitness/components/StatusDropdown/StatusDropdown'
//import GitspaceOwnerDropdown from 'cde-gitness/components/GitspaceOwnerDropdown/GitspaceOwnerDropdown'
import SortByDropdown from 'cde-gitness/components/SortByDropdown/SortByDropdown'
import OrganizationDropdown from 'cde-gitness/components/OrganizationDropdown/OrganizationDropdown'
import ProjectDropdown from 'cde-gitness/components/ProjectDropdown/ProjectDropdown'
import { ListGitspaces } from 'cde-gitness/components/GitspaceListing/ListGitspaces'
import { useGetCDEAPIParams } from 'cde-gitness/hooks/useGetCDEAPIParams'
import { useFindGitspaceSettings } from 'services/cde'
import { useStrings } from 'framework/strings'
import { useQueryParams } from 'hooks/useQueryParams'
import { useUpdateQueryParams } from 'hooks/useUpdateQueryParams'
import { LIST_FETCHING_LIMIT } from 'utils/Utils'
import { useUsageDashboardGitspaces } from 'cde-gitness/hooks/useUsageDashboardGitspaces'
import css from './UsageDashboardPage.module.scss'

interface DashboardQueryParams {
  page?: string
  limit?: string
  gitspace_states?: string
  gitspace_owner?: GitspaceOwnerType
  sort?: EnumGitspaceSort
  order?: 'asc' | 'desc'
  query?: string
  org_identifiers?: string
  project_identifiers?: string
}

interface FilterProps {
  gitspace_states: GitspaceStatus[]
  gitspace_owner: GitspaceOwnerType
  query: string
  org_identifiers?: string[]
  project_identifiers?: string[]
}

interface SortProps {
  sort: EnumGitspaceSort
  order: 'asc' | 'desc'
}

interface PageConfigProps {
  page: number
  limit: number
}

const GitspacesTabPanel: React.FC = () => {
  const { getString } = useStrings()
  const { updateQueryParams } = useUpdateQueryParams()
  const queryParams = useQueryParams<DashboardQueryParams>()
  const { standalone } = useAppContext()
  const { accountIdentifier = '' } = useGetCDEAPIParams()
  const [filter, setFilter] = useState<FilterProps>({
    gitspace_states: [],
    gitspace_owner: GitspaceOwnerType.ALL,
    query: queryParams.query || '',
    org_identifiers: [],
    project_identifiers: []
  })

  const [apiFilter, setApiFilter] = useState<FilterProps>(filter)

  const [sortConfig, setSortConfig] = useState<SortProps>({
    sort: (queryParams.sort as EnumGitspaceSort) || ('last_activated' as EnumGitspaceSort),
    order: queryParams.order || 'desc'
  })

  const [pageConfig, setPageConfig] = useState<PageConfigProps>({
    page: parseInt(queryParams.page || '1'),
    limit: parseInt(queryParams.limit || LIST_FETCHING_LIMIT.toString())
  })

  const {
    data: gitspaceSettings,
    loading: settingsLoading,
    error: settingsError
  } = useFindGitspaceSettings({
    accountIdentifier: accountIdentifier || '',
    lazy: !accountIdentifier
  })

  useEffect(() => {
    if (queryParams.gitspace_states) {
      const states = queryParams.gitspace_states?.split(',') as GitspaceStatus[]
      setFilter(prev => ({ ...prev, gitspace_states: states }))
    }
  }, [queryParams.gitspace_states])

  useEffect(() => {
    if (queryParams.gitspace_owner) {
      setFilter(prev => ({ ...prev, gitspace_owner: queryParams.gitspace_owner as GitspaceOwnerType }))
    }
  }, [queryParams.gitspace_owner])

  useEffect(() => {
    if (queryParams.org_identifiers) {
      const orgs = queryParams.org_identifiers.split(',')
      setFilter(prev => ({ ...prev, org_identifiers: orgs }))
    }
  }, [queryParams.org_identifiers])

  useEffect(() => {
    if (queryParams.project_identifiers) {
      const projects = queryParams.project_identifiers.split(',')
      setFilter(prev => ({ ...prev, project_identifiers: projects }))
    }
  }, [queryParams.project_identifiers])

  const debouncedFilterUpdate = useCallback(
    debounce((newFilter: FilterProps) => {
      setApiFilter(newFilter)
    }, 500),
    []
  )

  useEffect(() => {
    debouncedFilterUpdate(filter)

    return () => {
      debouncedFilterUpdate.cancel()
    }
  }, [filter, debouncedFilterUpdate])

  const { data, loading, error, refetch, pagination } = useUsageDashboardGitspaces({
    page: pageConfig.page,
    limit: pageConfig.limit,
    filter: apiFilter,
    sortConfig
  })

  const errorMessage =
    error || settingsError ? error?.message || settingsError?.message || 'An error occurred' : undefined

  const totalCount = pagination?.totalItems || 0

  const handleFilterChange = (key: string, value: any) => {
    const updatedFilter = { ...filter, [key]: value }
    setFilter(updatedFilter)
    setPageConfig(prev => ({ ...prev, page: 1 }))

    const queryParamsToUpdate: Partial<DashboardQueryParams> = { page: '1' }

    if (key === 'gitspace_states') {
      queryParamsToUpdate.gitspace_states = value.join(',')
    } else if (key === 'gitspace_owner') {
      queryParamsToUpdate.gitspace_owner = value
    } else if (key === 'query') {
      queryParamsToUpdate.query = value
    } else if (key === 'org_identifiers') {
      queryParamsToUpdate.org_identifiers = value.join(',')
    } else if (key === 'project_identifiers') {
      queryParamsToUpdate.project_identifiers = value.join(',')
    }

    updateQueryParams(queryParamsToUpdate)
  }

  const handleSort = (key: string, value: any) => {
    const updatedSort = { ...sortConfig, [key]: value }
    setSortConfig(updatedSort)
    updateQueryParams({ [key]: value })
  }

  const handlePagination = (key: string, value: any) => {
    const payload = { ...pageConfig, [key]: parseInt(value) }
    if (key === 'limit') {
      payload['page'] = 1
    }
    updateQueryParams({ page: payload?.page?.toString(), limit: payload?.limit?.toString() })
    setPageConfig(payload)
  }

  return (
    <>
      <Page.SubHeader>
        <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
          {!standalone && (
            <>
              <StatusDropdown
                value={filter.gitspace_states}
                onChange={value => handleFilterChange('gitspace_states', value)}
              />
              {/*<GitspaceOwnerDropdown*/}
              {/*  value={filter.gitspace_owner}*/}
              {/*  onChange={value => handleFilterChange('gitspace_owner', value)}*/}
              {/*/>*/}
            </>
          )}
          <OrganizationDropdown
            value={filter.org_identifiers || []}
            onChange={value => handleFilterChange('org_identifiers', value)}
          />
          <ProjectDropdown
            value={filter.project_identifiers || []}
            onChange={value => handleFilterChange('project_identifiers', value)}
            orgIdentifiers={filter.org_identifiers}
          />
        </Layout.Horizontal>

        <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
          <SortByDropdown
            value={sortConfig.sort}
            onChange={(val: EnumGitspaceSort) => handleSort('sort', val)}
            sortOrder={sortConfig.order}
            onOrderChange={val => handleSort('order', val)}
          />
          <ExpandingSearchInput
            autoFocus={false}
            alwaysExpanded={true}
            placeholder={getString('cde.usageDashboard.searchPlaceholder')}
            defaultValue={filter.query || ''}
            onChange={value => handleFilterChange('query', value)}
          />
        </Layout.Horizontal>
      </Page.SubHeader>
      <Page.Body
        loading={loading || settingsLoading}
        error={
          errorMessage ? (
            <Layout.Vertical spacing={'large'}>
              <Text font={{ variation: FontVariation.FORM_MESSAGE_DANGER }}>{errorMessage}</Text>
              <Button onClick={() => refetch()} variation={ButtonVariation.PRIMARY} text={getString('cde.retry')} />
            </Layout.Vertical>
          ) : undefined
        }
        noData={{
          when: () => !!(data && data.length === 0),
          image: noSpace,
          message: getString('cde.noGitspaces')
        }}>
        <Container className={standalone ? css.standaloneContainer : css.main}>
          <Layout.Vertical spacing={'normal'}>
            {data && data.length > 0 && (
              <>
                <Text className={css.totalItems}>
                  {getString('cde.total')}: {totalCount}
                </Text>

                <ListGitspaces
                  data={data || []}
                  refreshList={refetch}
                  hasFilter={!!filter.query || filter.gitspace_states.length > 0}
                  gotoPage={page => handlePagination('page', page + 1)}
                  onPageSizeChange={size => handlePagination('limit', size)}
                  pageConfig={{
                    page: pageConfig.page,
                    pageSize: pageConfig.limit,
                    totalItems: pagination?.totalItems || 0,
                    totalPages: pagination?.totalPages || 0
                  }}
                  gitspaceSettings={gitspaceSettings}
                  isFromDashboard={true}
                />
              </>
            )}
          </Layout.Vertical>
        </Container>
      </Page.Body>
    </>
  )
}
export default GitspacesTabPanel
