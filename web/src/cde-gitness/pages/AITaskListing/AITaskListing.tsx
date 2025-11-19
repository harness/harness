import React, { useMemo, useState } from 'react'
import {
  Breadcrumbs,
  Button,
  ButtonVariation,
  Container,
  ExpandingSearchInput,
  Layout,
  Page,
  Text
} from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { useHistory } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useAppContext } from 'AppContext'
import { useGetCDEAPIParams } from 'cde-gitness/hooks/useGetCDEAPIParams'
import { useListAITasks, type TypesAITask, type EnumAITaskState, type EnumAIAgent } from 'services/cde'
import { getErrorMessage, LIST_FETCHING_LIMIT } from 'utils/Utils'
import { useQueryParams } from 'hooks/useQueryParams'
import { AIAgentEnum, TaskStatus } from 'cde-gitness/constants/index'
import { useUpdateQueryParams } from 'hooks/useUpdateQueryParams'
import { ListAITasks } from 'cde-gitness/components/AITaskListing/ListAITasks'
import MultiSelectDropdownList from 'cde-gitness/components/MultiDropdownSelect/MultiDropdownSelect'
import css from './AITaskListing.module.scss'

interface PageBrowser {
  page?: string
  limit?: string
  query?: string
  aitask_states?: string
  aitask_agents?: string
}

interface PageConfig {
  page: number
  limit: number
}

interface FilterProps {
  query: string
  aitask_states: EnumAITaskState[]
  aitask_agents: EnumAIAgent[]
}

const AITaskListing = () => {
  const { getString } = useStrings()
  const space = useGetSpaceParam()
  const { standalone, accountInfo, routes } = useAppContext()
  const history = useHistory()
  const { orgIdentifier, projectIdentifier, accountIdentifier } = useGetCDEAPIParams()
  const pageBrowser = useQueryParams<PageBrowser>()
  const { updateQueryParams } = useUpdateQueryParams()

  const getBreadcrumbLinks = () => {
    if (standalone) {
      return [{ url: routes.toCDEAITasks({ space }), label: 'Tasks' }]
    }

    return [
      {
        url: `/account/${accountIdentifier}/module/cde`,
        label: `Account: ${accountInfo?.name || accountIdentifier}`
      },
      {
        url: `/account/${accountIdentifier}/module/cde/orgs/${orgIdentifier}`,
        label: `Organization: ${orgIdentifier}`
      },
      {
        url: `/account/${accountIdentifier}/module/cde/orgs/${orgIdentifier}/projects/${projectIdentifier}`,
        label: `Project: ${projectIdentifier}`
      },
      {
        url: routes.toCDEAITasks({ space }),
        label: getString('cde.aiTasks.tasks')
      }
    ]
  }
  const filterInit: FilterProps = {
    query: pageBrowser.query ?? '',
    aitask_states: pageBrowser.aitask_states
      ? pageBrowser.aitask_states.split(',').map(s => s.trim() as EnumAITaskState)
      : [],
    aitask_agents: pageBrowser.aitask_agents
      ? pageBrowser.aitask_agents.split(',').map(s => s.trim() as EnumAIAgent)
      : []
  }
  const [filter, setFilter] = useState<FilterProps>(filterInit)
  const pageInit: PageConfig = {
    page: pageBrowser.page ? parseInt(pageBrowser.page) : 1,
    limit: pageBrowser.limit ? parseInt(pageBrowser.limit) : LIST_FETCHING_LIMIT
  }
  const [pageConfig, setPageConfig] = useState<PageConfig>(pageInit)

  const handlePagination = (key: keyof PageConfig, value: number) => {
    const payload: PageConfig = {
      ...pageConfig,
      [key]: value
    }
    if (key === 'limit') {
      payload.page = 1
    }
    updateQueryParams({ page: payload.page.toString(), limit: payload.limit.toString() })
    setPageConfig(payload)
  }
  const handleFilterChange = (key: keyof FilterProps, value: any) => {
    const payload = { ...filter, [key]: value }
    setPageConfig(prevState => ({ page: 1, limit: prevState.limit }))
    setFilter(payload)

    const params =
      typeof value === 'string'
        ? { [key]: value, page: 1 }
        : { [key]: Array.isArray(value) ? (value as any[]).toString() : String(value), page: 1 }

    updateQueryParams(params as any)
  }
  const { data, loading, error, refetch, response } = useListAITasks({
    accountIdentifier: accountIdentifier || '',
    orgIdentifier: orgIdentifier || '',
    projectIdentifier: projectIdentifier || '',
    queryParams: {
      page: pageConfig.page,
      limit: pageConfig.limit,
      query: filter.query || undefined,
      aitask_states: filter.aitask_states?.length ? filter.aitask_states : undefined,
      aitask_agents: filter.aitask_agents?.length ? filter.aitask_agents : undefined
    },
    queryParamStringifyOptions: {
      arrayFormat: 'repeat'
    }
  })
  const { totalItems, totalPages, tasksExist } = useMemo(() => {
    const total = parseInt(response?.headers?.get('x-total') || '0')
    const pages = parseInt(response?.headers?.get('x-total-pages') || '0')
    const exist = !!parseInt(response?.headers?.get('x-total-no-filter') || '0')
    return { totalItems: total, totalPages: pages, tasksExist: exist }
  }, [response])
  const tasks: TypesAITask[] = Array.isArray(data) ? data : []
  const agentItems = [{ label: 'Claude Code', value: AIAgentEnum.CLAUDE_CODE as EnumAIAgent }] as {
    label: string
    value: EnumAIAgent
  }[]

  const AIAgentDropdown = () => {
    return (
      <MultiSelectDropdownList<EnumAIAgent>
        width={160}
        buttonTestId="ai-agent-select"
        items={agentItems}
        value={filter.aitask_agents}
        onSelect={val => handleFilterChange('aitask_agents', val)}
        placeholder="AI Agents"
        allowSearch={true}
        expandingSearchInputProps={{ autoFocus: false }}
      />
    )
  }
  const allStatuses: EnumAITaskState[] = [
    TaskStatus.UNINITIALIZED as EnumAITaskState,
    TaskStatus.RUNNING as EnumAITaskState,
    TaskStatus.COMPLETED as EnumAITaskState,
    TaskStatus.ERROR as EnumAITaskState
  ]
  const statusItems = allStatuses.map(s => ({
    label: s.charAt(0).toUpperCase() + s.slice(1),
    value: s
  })) as { label: string; value: EnumAITaskState }[]

  const AITaskStatusDropdown = () => {
    return (
      <MultiSelectDropdownList<EnumAITaskState>
        width={120}
        buttonTestId="aitask-status-select"
        items={statusItems}
        value={filter.aitask_states}
        onSelect={val => handleFilterChange('aitask_states', val)}
        placeholder={getString('status')}
      />
    )
  }

  return (
    <>
      <Page.Header title="Tasks" breadcrumbs={<Breadcrumbs links={getBreadcrumbLinks()} />} />
      <Page.SubHeader>
        <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
          <Button
            icon={'plus'}
            onClick={() => history.push(routes.toCDEAITaskCreate({ space }))}
            variation={ButtonVariation.PRIMARY}>
            {getString('cde.aiTasks.listing.newTask')}
          </Button>
          <AIAgentDropdown />
          <AITaskStatusDropdown />
        </Layout.Horizontal>

        <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
          <ExpandingSearchInput
            autoFocus={false}
            alwaysExpanded
            placeholder={getString('search')}
            onChange={text => {
              handleFilterChange('query', text)
            }}
            defaultValue={filter?.query ?? ''}
            width={240}
          />
        </Layout.Horizontal>
      </Page.SubHeader>

      <Container className={css.main}>
        <Page.Body
          loading={loading}
          error={
            error ? (
              <Layout.Vertical spacing="large">
                <Text font={{ variation: FontVariation.FORM_MESSAGE_DANGER }}>{getErrorMessage(error)}</Text>
                <Button onClick={() => refetch?.()} variation={ButtonVariation.PRIMARY} text={getString('cde.retry')} />
              </Layout.Vertical>
            ) : null
          }
          noData={{
            when: () => tasks?.length === 0 && !tasksExist,
            message: getString('cde.aiTasks.listing.noTasksFound')
          }}>
          {(tasks?.length || tasksExist) && (
            <>
              <Text className={css.totalItems}>{`${getString('cde.total') || 'Total'}: ${totalItems}`}</Text>
              <ListAITasks
                data={tasks}
                hasFilter={tasksExist}
                gotoPage={(pageNumber: number) => handlePagination('page', pageNumber + 1)}
                onPageSizeChange={(newSize: number) => handlePagination('limit', newSize)}
                pageConfig={{
                  page: pageConfig.page,
                  pageSize: pageConfig.limit,
                  totalItems,
                  totalPages
                }}
              />
            </>
          )}
        </Page.Body>
      </Container>
    </>
  )
}

export default AITaskListing
