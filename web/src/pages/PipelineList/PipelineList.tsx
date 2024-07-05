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

import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { Classes, Intent, Menu, MenuItem, Popover, Position } from '@blueprintjs/core'
import {
  Avatar,
  Button,
  ButtonVariation,
  Container,
  FlexExpander,
  Layout,
  PageBody,
  StringSubstitute,
  TableV2 as Table,
  Text,
  Utils,
  useToaster
} from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import cx from 'classnames'
import type { CellProps, Column } from 'react-table'
import Keywords from 'react-keywords'
import { useHistory } from 'react-router-dom'
import { useGet, useMutate } from 'restful-react'
import { Calendar, Timer, GitFork } from 'iconoir-react'
import { String, useStrings } from 'framework/strings'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { LIST_FETCHING_LIMIT, PageBrowserProps, getErrorMessage, timeDistance, voidFn } from 'utils/Utils'
import type { TypesPipeline } from 'services/code'
import { useQueryParams } from 'hooks/useQueryParams'
import { usePageIndex } from 'hooks/usePageIndex'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { ExecutionStatus, ExecutionState } from 'components/ExecutionStatus/ExecutionStatus'
import { getStatus } from 'utils/ExecutionUtils'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import useNewPipelineModal from 'components/NewPipelineModal/NewPipelineModal'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import useSpaceSSE from 'hooks/useSpaceSSE'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { CommitActions } from 'components/CommitActions/CommitActions'
import noPipelineImage from '../RepositoriesListing/no-repo.svg?url'
import css from './PipelineList.module.scss'

const PipelineList = () => {
  const { routes } = useAppContext()
  const history = useHistory()
  const { getString } = useStrings()
  const [searchTerm, setSearchTerm] = useState<string | undefined>()
  const pageBrowser = useQueryParams<PageBrowserProps>()
  const pageInit = pageBrowser.page ? parseInt(pageBrowser.page) : 1
  const [page, setPage] = usePageIndex(pageInit)
  const space = useGetSpaceParam()

  const { repoMetadata, error, loading, refetch } = useGetRepositoryMetadata()

  const {
    data: pipelines,
    error: pipelinesError,
    response,
    refetch: pipelinesRefetch
  } = useGet<TypesPipeline[]>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/pipelines`,
    queryParams: { page, limit: LIST_FETCHING_LIMIT, query: searchTerm, latest: true },
    lazy: !repoMetadata,
    debounce: 500
  })

  const { openModal } = useNewPipelineModal()
  //TODO - do not want to show load between refetchs - remove if/when we move to event stream method
  const [isInitialLoad, setIsInitialLoad] = useState(true)

  useEffect(() => {
    if (pipelines) {
      setIsInitialLoad(false)
    }
  }, [pipelines])

  const onEvent = useCallback(
    data => {
      // should I include pipeline id here? what if a new pipeline is created? could check for ids that are higher than the lowest id on the page?
      if (repoMetadata?.id === data?.repo_id) {
        //TODO - revisit full refresh - can I use the message to update the pipeline?
        pipelinesRefetch()
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [repoMetadata?.id]
  )

  const events = useMemo(
    () => ['execution_updated', 'execution_completed', 'execution_canceled', 'execution_running'],
    []
  )

  useSpaceSSE({
    space,
    events,
    onEvent
  })

  const NewPipelineButton = (
    <Button
      text={getString('pipelines.newPipelineButton')}
      variation={ButtonVariation.PRIMARY}
      icon="plus"
      onClick={() => {
        openModal({ repoMetadata })
      }}
      disabled={loading}
    />
  )

  const columns: Column<TypesPipeline>[] = useMemo(
    () => [
      {
        Header: getString('pipelines.name'),
        width: 'calc(50% - 105px)',
        Cell: ({ row }: CellProps<TypesPipeline>) => {
          const record = row.original
          return (
            <Layout.Horizontal spacing="small" className={css.nameContainer}>
              <ExecutionStatus
                status={getStatus(record?.execution?.status || ExecutionState.PENDING)}
                iconOnly
                noBackground
                iconSize={24}
                className={css.statusIcon}
                isCi
                inExecution
              />
              <Text className={css.repoName}>
                <Keywords value={searchTerm}>{record.identifier}</Keywords>
              </Text>
            </Layout.Horizontal>
          )
        }
      },
      {
        Header: getString('pipelines.lastExecution'),
        width: 'calc(50% - 105px)',
        Cell: ({ row }: CellProps<TypesPipeline>) => {
          const record = row.original.execution

          return record ? (
            <Layout.Vertical spacing={'small'} padding={{ left: 'small', right: 'small' }}>
              <Layout.Horizontal spacing={'small'} style={{ alignItems: 'center' }}>
                <Text className={css.desc}>{`#${record.number}`}</Text>
                <PipeSeparator height={7} />
                <Text className={css.desc} lineClamp={1}>
                  {record.title || record.message}
                </Text>
              </Layout.Horizontal>
              <Layout.Horizontal spacing={'xsmall'} style={{ alignItems: 'center' }}>
                <Avatar
                  email={record.author_email}
                  name={record.author_name}
                  size="small"
                  hoverCard={false}
                  className={css.avatar}
                />
                {/* TODO need logic here for different trigger types */}
                <Text className={css.author}>{record.author_name}</Text>
                {record.target && (
                  <>
                    <PipeSeparator height={7} />
                    <GitFork height={12} width={12} color={Utils.getRealCSSColor(Color.GREY_500)} />
                    <Text className={css.author} lineClamp={1}>
                      {record.target.split('/').pop()}
                    </Text>
                  </>
                )}
                <PipeSeparator height={7} />
                <Container onClick={Utils.stopEvent}>
                  <CommitActions
                    href={routes.toCODECommit({
                      repoPath: repoMetadata?.path as string,
                      commitRef: record?.after as string
                    })}
                    sha={record?.after as string}
                    enableCopy
                  />
                </Container>
              </Layout.Horizontal>
            </Layout.Vertical>
          ) : (
            <div className={css.spacer} />
          )
        }
      },
      {
        Header: getString('pipelines.time'),
        width: '180px',
        Cell: ({ row }: CellProps<TypesPipeline>) => {
          const record = row.original.execution

          return record ? (
            <Layout.Vertical spacing={'small'}>
              {record?.started && record?.finished && (
                <Layout.Horizontal spacing={'small'} style={{ alignItems: 'center' }}>
                  <Timer color={Utils.getRealCSSColor(Color.GREY_500)} />
                  <Text inline color={Color.GREY_500} lineClamp={1} width={180} font={{ size: 'small' }}>
                    {timeDistance(record.started, record.finished, true)}
                  </Text>
                </Layout.Horizontal>
              )}
              {record?.finished && (
                <Layout.Horizontal spacing={'small'} style={{ alignItems: 'center' }}>
                  <Calendar color={Utils.getRealCSSColor(Color.GREY_500)} />
                  <Text inline color={Color.GREY_500} lineClamp={1} width={180} font={{ size: 'small' }}>
                    {timeDistance(record.finished, Date.now(), true)} ago
                  </Text>
                </Layout.Horizontal>
              )}
            </Layout.Vertical>
          ) : (
            <div className={css.spacer} />
          )
        },
        disableSortBy: true
      },
      {
        Header: ' ',
        width: '30px',
        Cell: ({ row }: CellProps<TypesPipeline>) => {
          const [menuOpen, setMenuOpen] = useState(false)
          const record = row.original
          const { identifier } = record
          const repoPath = repoMetadata?.path || ''

          const confirmDeletePipeline = useConfirmAct()
          const { showSuccess, showError } = useToaster()
          const { mutate: deletePipeline } = useMutate<TypesPipeline>({
            verb: 'DELETE',
            path: `/api/v1/repos/${repoPath}/+/pipelines/${identifier}`
          })

          return (
            <Popover
              isOpen={menuOpen}
              onInteraction={nextOpenState => {
                setMenuOpen(nextOpenState)
              }}
              className={Classes.DARK}
              position={Position.BOTTOM_RIGHT}>
              <Button
                variation={ButtonVariation.ICON}
                icon="Options"
                data-testid={`menu-${record.identifier}`}
                onClick={e => {
                  e.stopPropagation()
                  setMenuOpen(true)
                }}
              />
              <Menu>
                <MenuItem
                  icon="edit"
                  text={getString('edit')}
                  onClick={e => {
                    e.stopPropagation()
                    history.push(routes.toCODEPipelineEdit({ repoPath, pipeline: identifier as string }))
                  }}
                />
                <MenuItem
                  icon="delete"
                  text={getString('delete')}
                  onClick={e => {
                    e.stopPropagation()
                    confirmDeletePipeline({
                      title: getString('pipelines.deletePipelineButton'),
                      confirmText: getString('delete'),
                      intent: Intent.DANGER,
                      message: (
                        <String
                          useRichText
                          stringID="pipelines.deletePipelineConfirm"
                          vars={{ pipeline: row.original.identifier }}
                        />
                      ),
                      action: async () => {
                        deletePipeline({})
                          .then(() => {
                            showSuccess(
                              <StringSubstitute
                                str={getString('pipelines.deletePipelineSuccess')}
                                vars={{
                                  pipeline: row.original.identifier
                                }}
                              />,
                              5000
                            )
                            pipelinesRefetch()
                          })
                          .catch((pipelineDeleteErr: unknown) => {
                            showError(getErrorMessage(pipelineDeleteErr), 0, 'pipelines.deletePipelineError')
                          })
                      }
                    })
                  }}
                />
                <MenuItem
                  icon="settings"
                  text={getString('settings')}
                  onClick={e => {
                    e.stopPropagation()
                    history.push(
                      routes.toCODEPipelineSettings({
                        repoPath: repoMetadata?.path || '',
                        pipeline: identifier as string
                      })
                    )
                  }}
                />
              </Menu>
            </Popover>
          )
        }
      }
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [getString, history, repoMetadata?.path, routes, searchTerm]
  )

  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('pageTitle.pipelines')}
        dataTooltipId="repositoryPipelines"
      />
      <PageBody
        className={cx({ [css.withError]: !!error || !!pipelinesError })}
        error={error || pipelinesError ? getErrorMessage(error || pipelinesError) : null}
        retryOnError={voidFn(refetch)}
        noData={{
          when: () => pipelines?.length === 0 && searchTerm === undefined,
          image: noPipelineImage,
          message: getString('pipelines.noData'),
          button: NewPipelineButton
        }}>
        <LoadingSpinner visible={(loading || isInitialLoad) && !searchTerm} withBorder={!!pipelines && isInitialLoad} />

        <Container padding="xlarge">
          <Layout.Horizontal spacing="large" className={css.layout}>
            {NewPipelineButton}
            <FlexExpander />
            <SearchInputWithSpinner loading={loading} query={searchTerm} setQuery={setSearchTerm} />
          </Layout.Horizontal>

          <Container margin={{ top: 'medium' }}>
            {!!pipelines?.length && (
              <Table<TypesPipeline>
                columns={columns}
                data={pipelines || []}
                onRowClick={pipelineInfo =>
                  history.push(
                    routes.toCODEExecutions({
                      repoPath: repoMetadata?.path as string,
                      pipeline: pipelineInfo.identifier as string
                    })
                  )
                }
              />
            )}

            <NoResultCard
              showWhen={() => !!pipelines && pipelines?.length === 0 && !!searchTerm?.length}
              forSearch={true}
            />
          </Container>
          <ResourceListingPagination response={response} page={page} setPage={setPage} />
        </Container>
      </PageBody>
    </Container>
  )
}

export default PipelineList
