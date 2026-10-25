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
import { Link } from 'react-router-dom'
import {
  Button,
  ButtonSize,
  ButtonVariation,
  Container,
  DropDown,
  FlexExpander,
  Layout,
  PageBody,
  TableV2,
  Text
} from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useGet } from 'restful-react'
import type { CellProps, Column } from 'react-table'
import { Render } from 'react-jsx-match'
import { isEmpty, noop } from 'lodash-es'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { StringKeys, useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { StatusBadge } from 'components/StatusBadge/StatusBadge'
import { TimePopoverWithLocal } from 'utils/timePopoverLocal/TimePopoverWithLocal'
import { usePageIndex } from 'hooks/usePageIndex'
import { useQueryParams } from 'hooks/useQueryParams'
import { LIST_FETCHING_LIMIT, PageBrowserProps, getErrorMessage, voidFn } from 'utils/Utils'
import { useTaskNumbersByOriginRef } from 'utils/TaskUtils'
import {
  INSIGHTS_ISSUE_STATUSES,
  INSIGHTS_LEVELS,
  INSIGHTS_STATUS_COLORS,
  TypesInsightsIssue,
  TypesInsightsProject,
  getInsightsApiPath,
  getLevelColors
} from 'utils/InsightsUtils'
import { ErrorProjects } from './ErrorProjects'
import css from './Errors.module.scss'

export default function Errors() {
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const browserParams = useQueryParams<PageBrowserProps>()
  const pageInit = browserParams.page ? parseInt(browserParams.page) : 1
  const [page, setPage] = usePageIndex(pageInit)
  const [tab, setTab] = useState<'issues' | 'projects'>('issues')
  const [projectFilter, setProjectFilter] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [levelFilter, setLevelFilter] = useState('')
  const [releaseFilter, setReleaseFilter] = useState<string | undefined>()
  const [sort, setSort] = useState('last_seen')
  const [searchTerm, setSearchTerm] = useState<string | undefined>()
  const { repoMetadata, error, loading, refetch } = useGetRepositoryMetadata()
  const repoPath = repoMetadata?.path as string
  const insightsPath = getInsightsApiPath(repoMetadata?.path)
  const taskNumbers = useTaskNumbersByOriginRef(repoMetadata?.path, 'insights')

  const {
    data: projects,
    refetch: refetchProjects,
    error: projectsError
  } = useGet<TypesInsightsProject[]>({
    path: `${insightsPath}/projects`,
    lazy: !repoMetadata
  })

  const {
    data: issues,
    error: issuesError,
    loading: issuesLoading,
    response
  } = useGet<TypesInsightsIssue[]>({
    path: `${insightsPath}/issues`,
    queryParams: {
      page,
      limit: LIST_FETCHING_LIMIT,
      sort,
      order: 'desc',
      ...(projectFilter && { project: projectFilter }),
      ...(statusFilter && { status: statusFilter }),
      ...(levelFilter && { level: levelFilter }),
      ...(releaseFilter && { release: releaseFilter }),
      ...(searchTerm && { query: searchTerm })
    },
    debounce: 500,
    lazy: !repoMetadata || tab !== 'issues'
  })

  const columns: Column<TypesInsightsIssue>[] = useMemo(
    () => [
      {
        Header: getString('insights.issueTitle'),
        width: 'calc(100% - 880px)',
        Cell: ({ row }: CellProps<TypesInsightsIssue>) => (
          <Link
            className={css.rowLink}
            to={routes.toCODEError({ repoPath, issueId: String(row.original.id) })}
            title={row.original.title}>
            <Text color={Color.GREY_800} lineClamp={1} font={{ weight: 'semi-bold' }}>
              {row.original.title}
            </Text>
          </Link>
        )
      },
      {
        Header: getString('insights.culprit'),
        width: '180px',
        Cell: ({ row }: CellProps<TypesInsightsIssue>) => (
          <Text lineClamp={1} tooltip={row.original.culprit}>
            {row.original.culprit || '-'}
          </Text>
        )
      },
      {
        Header: getString('insights.level'),
        width: '80px',
        Cell: ({ row }: CellProps<TypesInsightsIssue>) => (
          <StatusBadge label={row.original.level} colors={getLevelColors(row.original.level)} />
        )
      },
      {
        Header: getString('insights.timesSeen'),
        width: '80px',
        Cell: ({ row }: CellProps<TypesInsightsIssue>) => <Text>{row.original.times_seen}</Text>
      },
      {
        Header: getString('insights.lastSeen'),
        width: '130px',
        Cell: ({ row }: CellProps<TypesInsightsIssue>) =>
          row.original.last_seen ? (
            <TimePopoverWithLocal time={row.original.last_seen} inline={false} />
          ) : (
            <Text>-</Text>
          )
      },
      {
        Header: getString('insights.status'),
        width: '110px',
        Cell: ({ row }: CellProps<TypesInsightsIssue>) =>
          row.original.status ? (
            <StatusBadge
              label={getString(`insights.statusValue.${row.original.status}` as StringKeys)}
              colors={INSIGHTS_STATUS_COLORS[row.original.status]}
            />
          ) : null
      },
      {
        Header: getString('insights.release'),
        width: '130px',
        Cell: ({ row }: CellProps<TypesInsightsIssue>) => (
          <Text lineClamp={1}>{row.original.last_release || row.original.first_release || '-'}</Text>
        )
      },
      {
        Header: getString('insights.task'),
        width: '90px',
        Cell: ({ row }: CellProps<TypesInsightsIssue>) => {
          const number = taskNumbers[`${row.original.project_id}:${row.original.fingerprint}`]
          if (number) {
            return <Link to={routes.toCODETask({ repoPath, taskNumber: String(number) })}>TASK-{number}</Link>
          }
          return row.original.task_id ? (
            <Link to={routes.toCODETasks({ repoPath })}>{getString('insights.taskLinked')}</Link>
          ) : (
            <Text>-</Text>
          )
        }
      }
    ],
    [getString, routes, repoPath, taskNumbers]
  )

  const allItem = (label: string) => ({ label, value: '' })
  const projectItems = [
    allItem(getString('insights.allProjects')),
    ...(projects || []).map(p => ({ label: p.identifier as string, value: String(p.id) }))
  ]
  const statusItems = [
    allItem(getString('insights.allStatuses')),
    ...INSIGHTS_ISSUE_STATUSES.map(s => ({ label: getString(`insights.statusValue.${s}` as StringKeys), value: s }))
  ]
  const levelItems = [allItem(getString('insights.allLevels')), ...INSIGHTS_LEVELS.map(l => ({ label: l, value: l }))]
  const sortItems = [
    { label: getString('insights.sortLastSeen'), value: 'last_seen' },
    { label: getString('insights.sortTimesSeen'), value: 'times_seen' }
  ]
  const hasFilter = !!(projectFilter || statusFilter || levelFilter || releaseFilter)

  const resetPage = () => setPage(1)

  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('insights.title')}
        dataTooltipId="repositoryErrors"
      />
      <PageBody
        error={getErrorMessage(error || (tab === 'issues' ? issuesError : projectsError))}
        retryOnError={voidFn(refetch)}>
        <LoadingSpinner
          visible={loading || (issuesLoading && !searchTerm && tab === 'issues')}
          withBorder={!searchTerm}
        />

        <Render when={repoMetadata}>
          <Layout.Vertical>
            <Container padding="xlarge">
              <Layout.Horizontal className={css.filters} spacing="medium">
                <Button
                  size={ButtonSize.SMALL}
                  variation={tab === 'issues' ? ButtonVariation.SECONDARY : ButtonVariation.TERTIARY}
                  text={getString('insights.issues')}
                  icon="warning-sign"
                  onClick={() => setTab('issues')}
                />
                <Button
                  size={ButtonSize.SMALL}
                  variation={tab === 'projects' ? ButtonVariation.SECONDARY : ButtonVariation.TERTIARY}
                  text={getString('insights.projects')}
                  icon="cog"
                  onClick={() => setTab('projects')}
                />
                <Render when={tab === 'issues'}>
                  <FlexExpander />
                  <SearchInputWithSpinner
                    spinnerPosition="right"
                    loading={issuesLoading && searchTerm !== undefined}
                    placeholder={getString('insights.searchPlaceholder')}
                    query={searchTerm || ''}
                    setQuery={value => {
                      setSearchTerm(value)
                      resetPage()
                    }}
                  />
                  <DropDown
                    value={projectFilter}
                    items={projectItems}
                    onChange={({ value }) => {
                      setProjectFilter(value as string)
                      resetPage()
                    }}
                  />
                  <DropDown
                    value={statusFilter}
                    items={statusItems}
                    onChange={({ value }) => {
                      setStatusFilter(value as string)
                      resetPage()
                    }}
                  />
                  <DropDown
                    value={levelFilter}
                    items={levelItems}
                    onChange={({ value }) => {
                      setLevelFilter(value as string)
                      resetPage()
                    }}
                  />
                  <SearchInputWithSpinner
                    spinnerPosition="right"
                    loading={false}
                    placeholder={getString('insights.releaseFilter')}
                    query={releaseFilter || ''}
                    setQuery={value => {
                      setReleaseFilter(value)
                      resetPage()
                    }}
                  />
                  <DropDown value={sort} items={sortItems} onChange={({ value }) => setSort(value as string)} />
                </Render>
              </Layout.Horizontal>
            </Container>

            <Container padding={{ left: 'xlarge', right: 'xlarge', bottom: 'xlarge' }}>
              {tab === 'projects' ? (
                <ErrorProjects repoPath={repoPath} projects={projects || []} onChanged={refetchProjects} />
              ) : isEmpty(issues) ? (
                <NoResultCard
                  forSearch={!!searchTerm}
                  forFilter={hasFilter}
                  emptyFilterMessage={getString('insights.noIssuesForFilter')}
                  message={getString('insights.noIssues')}
                  buttonText={isEmpty(projects) ? getString('insights.newProject') : undefined}
                  onButtonClick={isEmpty(projects) ? () => setTab('projects') : undefined}
                />
              ) : (
                <>
                  <TableV2<TypesInsightsIssue>
                    className={css.table}
                    columns={columns}
                    data={issues || []}
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
