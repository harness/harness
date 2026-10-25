/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useMemo, useState } from 'react'
import { Link, useHistory } from 'react-router-dom'
import {
  Button,
  ButtonSize,
  ButtonVariation,
  Checkbox,
  Container,
  DropDown,
  FlexExpander,
  Layout,
  PageBody,
  PageHeader,
  TableV2,
  Text,
  useToaster
} from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useGet, useMutate } from 'restful-react'
import type { CellProps, Column } from 'react-table'
import { Render } from 'react-jsx-match'
import { isEmpty, noop } from 'lodash-es'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { StringKeys, useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { TaskStateBadge } from 'components/TaskStateBadge/TaskStateBadge'
import { TimePopoverWithLocal } from 'utils/timePopoverLocal/TimePopoverWithLocal'
import { usePageIndex } from 'hooks/usePageIndex'
import { useQueryParams } from 'hooks/useQueryParams'
import { LIST_FETCHING_LIMIT, PageBrowserProps, getErrorMessage, voidFn } from 'utils/Utils'
import {
  TASK_ORIGINS,
  TASK_PRIORITIES,
  TASK_STATES,
  TaskState,
  TaskStateInput,
  TypesTask,
  getAssigneeLabel,
  getTaskApiPath
} from 'utils/TaskUtils'
import { useNewTaskModal } from './useNewTaskModal'
import { TaskBoard } from './TaskBoard'
import css from './Tasks.module.scss'

const BOARD_LIMIT = 100

interface TasksProps {
  // Lists the tasks of all the repositories of the space (recursively), with a repository column.
  spaceMode?: boolean
}

export default function Tasks({ spaceMode = false }: TasksProps) {
  const { getString } = useStrings()
  const history = useHistory()
  const { routes } = useAppContext()
  const { showError, showSuccess } = useToaster()
  const browserParams = useQueryParams<PageBrowserProps>()
  const pageInit = browserParams.page ? parseInt(browserParams.page) : 1
  const [page, setPage] = usePageIndex(pageInit)
  const [view, setView] = useState<'list' | 'board'>('list')
  const [stateFilter, setStateFilter] = useState('')
  const [priorityFilter, setPriorityFilter] = useState('')
  const [originFilter, setOriginFilter] = useState('')
  const [agentOnly, setAgentOnly] = useState(false)
  const [searchTerm, setSearchTerm] = useState<string | undefined>()
  const space = useGetSpaceParam()
  const { repoMetadata, error, loading, refetch } = useGetRepositoryMetadata()
  const repoPath = repoMetadata?.path as string
  const isBoard = !spaceMode && view === 'board'
  const tasksPath = spaceMode ? `/api/v1/spaces/${space}/+/tasks` : getTaskApiPath(repoMetadata?.path)

  const {
    data,
    error: tasksError,
    loading: tasksLoading,
    refetch: refetchTasks,
    response
  } = useGet<TypesTask[]>({
    path: tasksPath,
    queryParams: {
      ...(spaceMode && { recursive: true }),
      page: isBoard ? 1 : page,
      limit: isBoard ? BOARD_LIMIT : LIST_FETCHING_LIMIT,
      ...(!isBoard && stateFilter && { state: stateFilter }),
      ...(priorityFilter && { priority: priorityFilter }),
      ...(originFilter && { origin: originFilter }),
      ...(agentOnly && { is_agent: true }),
      ...(searchTerm && { query: searchTerm })
    },
    debounce: 500,
    lazy: !spaceMode && !repoMetadata
  })

  const { mutate: changeState } = useMutate<TypesTask, unknown, unknown, TaskStateInput, { number: number }>({
    verb: 'POST',
    path: ({ number }) => `${getTaskApiPath(repoMetadata?.path)}/${number}/state`
  })

  const onMove = (task: TypesTask, state: TaskState) => {
    changeState({ state }, { pathParams: { number: task.number as number } })
      .then(() => {
        showSuccess(getString('tasks.stateChanged', { state: getString(`tasks.stateValue.${state}` as StringKeys) }))
      })
      .catch(err => showError(getErrorMessage(err) || getString('tasks.moveFailed')))
      .finally(() => refetchTasks())
  }

  const { openModal } = useNewTaskModal({
    repoPath: repoMetadata?.path,
    onCreated: task =>
      history.push(routes.toCODETask({ repoPath: repoMetadata?.path as string, taskNumber: String(task.number) }))
  })

  const columns: Column<TypesTask>[] = useMemo(
    () => [
      {
        Header: getString('tasks.number'),
        width: '60px',
        Cell: ({ row }: CellProps<TypesTask>) => <Text color={Color.GREY_500}>#{row.original.number}</Text>
      },
      {
        Header: getString('tasks.taskTitle'),
        width: spaceMode ? 'calc(100% - 700px)' : 'calc(100% - 520px)',
        Cell: ({ row }: CellProps<TypesTask>) => (
          <Link
            className={css.rowLink}
            to={routes.toCODETask({
              repoPath: (spaceMode ? row.original.repo_path : repoPath) as string,
              taskNumber: String(row.original.number)
            })}>
            <Text color={Color.GREY_800} lineClamp={1} font={{ weight: 'semi-bold' }}>
              {row.original.title}
            </Text>
          </Link>
        )
      },
      ...(spaceMode
        ? [
            {
              Header: getString('tasks.repository'),
              width: '180px',
              Cell: ({ row }: CellProps<TypesTask>) => (
                <Link className={css.rowLink} to={routes.toCODETasks({ repoPath: row.original.repo_path as string })}>
                  <Text color={Color.GREY_700} lineClamp={1}>
                    {row.original.repo_path}
                  </Text>
                </Link>
              )
            }
          ]
        : []),
      {
        Header: getString('tasks.state'),
        width: '130px',
        Cell: ({ row }: CellProps<TypesTask>) => <TaskStateBadge state={row.original.state} />
      },
      {
        Header: getString('tasks.priority'),
        width: '80px',
        Cell: ({ row }: CellProps<TypesTask>) => <Text>{row.original.priority?.toUpperCase()}</Text>
      },
      {
        Header: getString('tasks.origin'),
        width: '90px',
        Cell: ({ row }: CellProps<TypesTask>) => (
          <Text>{getString(`tasks.originValue.${row.original.origin}` as StringKeys)}</Text>
        )
      },
      {
        Header: getString('tasks.assignee'),
        width: '110px',
        Cell: ({ row }: CellProps<TypesTask>) => <Text>{getAssigneeLabel(row.original, getString)}</Text>
      },
      {
        Header: getString('tasks.due'),
        width: '110px',
        Cell: ({ row }: CellProps<TypesTask>) =>
          row.original.due ? <TimePopoverWithLocal time={row.original.due} inline={false} /> : <Text>-</Text>
      }
    ],
    [getString, routes, repoPath, spaceMode]
  )

  const allItem = (label: string) => ({ label, value: '' })
  const stateItems = [
    allItem(getString('tasks.allStates')),
    ...TASK_STATES.map(s => ({ label: getString(`tasks.stateValue.${s}` as StringKeys), value: s }))
  ]
  const priorityItems = [
    allItem(getString('tasks.allPriorities')),
    ...TASK_PRIORITIES.map(p => ({ label: p.toUpperCase(), value: p }))
  ]
  const originItems = [
    allItem(getString('tasks.allOrigins')),
    ...TASK_ORIGINS.map(o => ({ label: getString(`tasks.originValue.${o}` as StringKeys), value: o }))
  ]
  const hasFilter = !!(stateFilter || priorityFilter || originFilter || agentOnly)

  return (
    <Container className={css.main}>
      {spaceMode ? (
        <PageHeader title={getString('tasks.title')} />
      ) : (
        <RepositoryPageHeader
          repoMetadata={repoMetadata}
          title={getString('tasks.title')}
          dataTooltipId="repositoryTasks"
        />
      )}
      <PageBody error={getErrorMessage(error || tasksError)} retryOnError={voidFn(refetch)}>
        <LoadingSpinner visible={loading || (tasksLoading && !searchTerm)} withBorder={!searchTerm} />

        <Render when={repoMetadata || spaceMode}>
          <Layout.Vertical>
            <Container padding="xlarge">
              <Layout.Horizontal className={css.filters} spacing="medium">
                <SearchInputWithSpinner
                  spinnerPosition="right"
                  loading={tasksLoading && searchTerm !== undefined}
                  placeholder={getString('tasks.searchPlaceholder')}
                  query={searchTerm || ''}
                  setQuery={value => {
                    setSearchTerm(value)
                    setPage(1)
                  }}
                />
                <Render when={!isBoard}>
                  <DropDown
                    value={stateFilter}
                    items={stateItems}
                    onChange={({ value }) => {
                      setStateFilter(value as string)
                      setPage(1)
                    }}
                  />
                </Render>
                <DropDown
                  value={priorityFilter}
                  items={priorityItems}
                  onChange={({ value }) => {
                    setPriorityFilter(value as string)
                    setPage(1)
                  }}
                />
                <DropDown
                  value={originFilter}
                  items={originItems}
                  onChange={({ value }) => {
                    setOriginFilter(value as string)
                    setPage(1)
                  }}
                />
                <Checkbox
                  label={getString('tasks.agentOnly')}
                  checked={agentOnly}
                  onChange={e => {
                    setAgentOnly(e.currentTarget.checked)
                    setPage(1)
                  }}
                />
                <FlexExpander />
                <Render when={!spaceMode}>
                  <Button
                    size={ButtonSize.SMALL}
                    variation={isBoard ? ButtonVariation.TERTIARY : ButtonVariation.SECONDARY}
                    text={getString('tasks.listView')}
                    icon="list"
                    onClick={() => setView('list')}
                  />
                  <Button
                    size={ButtonSize.SMALL}
                    variation={isBoard ? ButtonVariation.SECONDARY : ButtonVariation.TERTIARY}
                    text={getString('tasks.boardView')}
                    icon="grid-view"
                    onClick={() => setView('board')}
                  />
                  <Button
                    variation={ButtonVariation.PRIMARY}
                    text={getString('tasks.newTask')}
                    icon="plus"
                    onClick={openModal}
                  />
                </Render>
              </Layout.Horizontal>
            </Container>

            <Container padding={{ left: 'xlarge', right: 'xlarge', bottom: 'xlarge' }}>
              {isEmpty(data) ? (
                <NoResultCard
                  forSearch={!!searchTerm}
                  forFilter={hasFilter}
                  emptyFilterMessage={getString('tasks.noTasksForFilter')}
                  message={getString('tasks.noTasks')}
                  buttonText={spaceMode ? undefined : getString('tasks.newTask')}
                  onButtonClick={spaceMode ? undefined : openModal}
                />
              ) : isBoard ? (
                <TaskBoard repoPath={repoPath} tasks={data || []} onMove={onMove} />
              ) : (
                <>
                  <TableV2<TypesTask>
                    className={css.table}
                    columns={columns}
                    data={data || []}
                    getRowClassName={() => css.row}
                    onRowClick={noop}
                  />
                  <ResourceListingPagination response={response} page={page} setPage={setPage} />
                </>
              )}
            </Container>
          </Layout.Vertical>
        </Render>
      </PageBody>
    </Container>
  )
}
