/*
 * Copyright 2026 Harness, Inc.
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

import React, { useMemo } from 'react'
import { Link } from 'react-router-dom'
import { Card, Container, Layout, Page, PageBody, TableV2, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import type { CellProps, Column } from 'react-table'
import { Render } from 'react-jsx-match'
import cx from 'classnames'
import { useAppContext } from 'AppContext'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useStrings } from 'framework/strings'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { TaskStateBadge } from 'components/TaskStateBadge/TaskStateBadge'
import { TimePopoverWithLocal } from 'utils/timePopoverLocal/TimePopoverWithLocal'
import { getErrorMessage, voidFn } from 'utils/Utils'
import {
  AIAgentRun,
  AIAgentTask,
  formatSuccessRate,
  getQuotaPercent,
  useGetAIAgentPanel
} from 'utils/AIAgentUtils'
import css from './AIAgent.module.scss'

interface ModelKindRow {
  model: string
  kind: string
  rate: number
  samples: number
}

export default function AIAgent() {
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const space = useGetSpaceParam()
  const { data, error, loading, refetch } = useGetAIAgentPanel(space)

  const taskColumns: Column<AIAgentTask>[] = useMemo(
    () => [
      {
        Header: getString('aiAgent.number'),
        width: '60px',
        Cell: ({ row }: CellProps<AIAgentTask>) => <Text>{row.original.number}</Text>
      },
      {
        Header: getString('aiAgent.repository'),
        width: '25%',
        Cell: ({ row }: CellProps<AIAgentTask>) => (
          <Link to={routes.toCODETasks({ repoPath: row.original.repo_path })}>
            <Text color={Color.PRIMARY_7} lineClamp={1}>
              {row.original.repo_path}
            </Text>
          </Link>
        )
      },
      {
        Header: getString('aiAgent.taskTitle'),
        width: '*',
        Cell: ({ row }: CellProps<AIAgentTask>) => <Text lineClamp={1}>{row.original.title}</Text>
      },
      {
        Header: getString('aiAgent.state'),
        width: '130px',
        Cell: ({ row }: CellProps<AIAgentTask>) => <TaskStateBadge state={row.original.state} />
      },
      {
        Header: getString('aiAgent.model'),
        width: '25%',
        Cell: ({ row }: CellProps<AIAgentTask>) => (
          <Text lineClamp={1}>{row.original.model || getString('aiAgent.unknownModel')}</Text>
        )
      }
    ],
    [getString, routes]
  )

  const waitingColumns: Column<AIAgentTask>[] = useMemo(() => taskColumns.slice(0, 4), [taskColumns])

  const runColumns: Column<AIAgentRun>[] = useMemo(
    () => [
      {
        Header: getString('aiAgent.number'),
        width: '60px',
        Cell: ({ row }: CellProps<AIAgentRun>) => <Text>{row.original.task_number}</Text>
      },
      {
        Header: getString('aiAgent.repository'),
        width: '25%',
        Cell: ({ row }: CellProps<AIAgentRun>) => <Text lineClamp={1}>{row.original.repo_path}</Text>
      },
      {
        Header: getString('aiAgent.taskTitle'),
        width: '*',
        Cell: ({ row }: CellProps<AIAgentRun>) => <Text lineClamp={1}>{row.original.title}</Text>
      },
      {
        Header: getString('aiAgent.model'),
        width: '22%',
        Cell: ({ row }: CellProps<AIAgentRun>) => <Text lineClamp={1}>{row.original.model}</Text>
      },
      {
        Header: getString('aiAgent.outcome'),
        width: '120px',
        Cell: ({ row }: CellProps<AIAgentRun>) => <Text>{row.original.outcome}</Text>
      },
      {
        Header: getString('aiAgent.requests'),
        width: '90px',
        Cell: ({ row }: CellProps<AIAgentRun>) => <Text>{row.original.requests_used}</Text>
      },
      {
        Header: getString('aiAgent.when'),
        width: '130px',
        Cell: ({ row }: CellProps<AIAgentRun>) => <TimePopoverWithLocal time={row.original.created} inline={false} />
      }
    ],
    [getString]
  )

  const modelColumns: Column<ModelKindRow>[] = useMemo(
    () => [
      {
        Header: getString('aiAgent.model'),
        width: '*',
        Cell: ({ row }: CellProps<ModelKindRow>) => <Text lineClamp={1}>{row.original.model}</Text>
      },
      {
        Header: getString('aiAgent.kind'),
        width: '25%',
        Cell: ({ row }: CellProps<ModelKindRow>) => <Text>{row.original.kind}</Text>
      },
      {
        Header: getString('aiAgent.successRate'),
        width: '20%',
        Cell: ({ row }: CellProps<ModelKindRow>) => <Text>{formatSuccessRate(row.original.rate)}</Text>
      },
      {
        Header: getString('aiAgent.samples'),
        width: '15%',
        Cell: ({ row }: CellProps<ModelKindRow>) => <Text>{row.original.samples}</Text>
      }
    ],
    [getString]
  )

  const modelRows: ModelKindRow[] = useMemo(
    () =>
      (data?.model_success || []).flatMap(m =>
        m.kinds.map(k => ({ model: m.model, kind: k.kind, rate: k.rate, samples: k.samples }))
      ),
    [data]
  )

  const quota = data?.quota
  const percent = getQuotaPercent(quota)
  const classes = Object.keys(quota?.used_by_class || {}).map(name => ({ name }))

  const section = (title: string, body: React.ReactNode) => (
    <Card className={css.card}>
      <Layout.Vertical spacing="medium">
        <Text font={{ variation: FontVariation.H5 }}>{title}</Text>
        {body}
      </Layout.Vertical>
    </Card>
  )

  const empty = (message: string) => <Text color={Color.GREY_500}>{message}</Text>

  return (
    <Container className={css.main}>
      <Page.Header title={getString('aiAgent.title')} />
      <PageBody error={getErrorMessage(error)} retryOnError={voidFn(refetch)}>
        <LoadingSpinner visible={loading} withBorder />

        <Render when={data}>
          <Layout.Vertical spacing="large" padding="xlarge">
            <Render when={!data?.enabled}>
              <Text icon="warning-sign" color={Color.ORANGE_700}>
                {getString('aiAgent.disabled')}
              </Text>
            </Render>

            {section(
              getString('aiAgent.quota'),
              <Layout.Vertical spacing="small">
                <Render when={data?.enabled}>
                  <div className={css.progress}>
                    <div
                      className={cx(css.progressBar, { [css.progressBarFull]: percent >= 1 })}
                      style={{ width: `${percent * 100}%` }}
                    />
                  </div>
                  <Text>{getString('aiAgent.quotaUsed', { used: quota?.used, limit: quota?.limit })}</Text>
                  <TableV2<{ name: string }>
                    className={css.table}
                    columns={[
                      {
                        Header: getString('aiAgent.quotaClass'),
                        width: '40%',
                        Cell: ({ row }: CellProps<{ name: string }>) => <Text>{row.original.name}</Text>
                      },
                      {
                        Header: getString('aiAgent.quotaClassUsed'),
                        width: '30%',
                        Cell: ({ row }: CellProps<{ name: string }>) => <Text>{quota?.used_by_class[row.original.name]}</Text>
                      },
                      {
                        Header: getString('aiAgent.quotaClassRemaining'),
                        width: '30%',
                        Cell: ({ row }: CellProps<{ name: string }>) => <Text>{quota?.remaining_by_class[row.original.name]}</Text>
                      }
                    ]}
                    data={classes}
                    getRowClassName={() => css.row}
                  />
                </Render>
                <Text>{getString('aiAgent.quotaSpace', { used: quota?.space_used_today ?? 0 })}</Text>
              </Layout.Vertical>
            )}

            {section(
              getString('aiAgent.activeTasks'),
              data?.active_tasks?.length ? (
                <TableV2<AIAgentTask>
                  className={css.table}
                  columns={taskColumns}
                  data={data.active_tasks}
                  getRowClassName={() => css.row}
                />
              ) : (
                empty(getString('aiAgent.noActiveTasks'))
              )
            )}

            {section(
              getString('aiAgent.waitingQuota'),
              data?.waiting_quota?.count ? (
                <Layout.Vertical spacing="small">
                  <Text>{getString('aiAgent.waitingQuotaCount', { count: data.waiting_quota.count })}</Text>
                  <TableV2<AIAgentTask>
                    className={css.table}
                    columns={waitingColumns}
                    data={data.waiting_quota.tasks}
                    getRowClassName={() => css.row}
                  />
                </Layout.Vertical>
              ) : (
                empty(getString('aiAgent.noWaitingQuota'))
              )
            )}

            {section(
              getString('aiAgent.modelSuccess'),
              modelRows.length ? (
                <TableV2<ModelKindRow>
                  className={css.table}
                  columns={modelColumns}
                  data={modelRows}
                  getRowClassName={() => css.row}
                />
              ) : (
                empty(getString('aiAgent.noModelSuccess'))
              )
            )}

            {section(
              getString('aiAgent.latestRuns'),
              data?.latest_runs?.length ? (
                <TableV2<AIAgentRun>
                  className={css.table}
                  columns={runColumns}
                  data={data.latest_runs}
                  getRowClassName={() => css.row}
                />
              ) : (
                empty(getString('aiAgent.noRuns'))
              )
            )}
          </Layout.Vertical>
        </Render>
      </PageBody>
    </Container>
  )
}
