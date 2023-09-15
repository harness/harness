import React, { useEffect, useMemo, useState } from 'react'
import {
  Button,
  ButtonVariation,
  Container,
  FlexExpander,
  Layout,
  PageBody,
  TableV2 as Table,
  Text,
  Utils
} from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import cx from 'classnames'
import type { CellProps, Column } from 'react-table'
import { useHistory, useParams } from 'react-router-dom'
import { useGet } from 'restful-react'
import { Timer, Calendar } from 'iconoir-react'
import { useStrings } from 'framework/strings'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { useAppContext } from 'AppContext'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { LIST_FETCHING_LIMIT, PageBrowserProps, getErrorMessage, timeDistance, voidFn } from 'utils/Utils'
import type { CODEProps } from 'RouteDefinitions'
import type { EnumTriggerAction, TypesExecution } from 'services/code'
import { useQueryParams } from 'hooks/useQueryParams'
import { usePageIndex } from 'hooks/usePageIndex'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { ExecutionState, ExecutionStatus } from 'components/ExecutionStatus/ExecutionStatus'
import { getStatus } from 'utils/ExecutionUtils'
import useSpaceSSE from 'hooks/useSpaceSSE'
import { ExecutionText, ExecutionTrigger } from 'components/ExecutionText/ExecutionText'
import useRunPipelineModal from 'components/RunPipelineModal/RunPipelineModal'
import useLiveTimer from 'hooks/useLiveTimeHook'
import noExecutionImage from '../RepositoriesListing/no-repo.svg'
import css from './ExecutionList.module.scss'

const ExecutionList = () => {
  const { routes } = useAppContext()
  const { pipeline } = useParams<CODEProps>()
  const history = useHistory()
  const { getString } = useStrings()
  const pageBrowser = useQueryParams<PageBrowserProps>()
  const pageInit = pageBrowser.page ? parseInt(pageBrowser.page) : 1
  const [page, setPage] = usePageIndex(pageInit)
  const currentTime = useLiveTimer()

  const { repoMetadata, error, loading, refetch, space } = useGetRepositoryMetadata()

  const {
    data: executions,
    error: executionsError,
    response,
    refetch: executionsRefetch
  } = useGet<TypesExecution[]>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/pipelines/${pipeline}/executions`,
    queryParams: { page, limit: LIST_FETCHING_LIMIT },
    lazy: !repoMetadata
  })

  //TODO - do not want to show load between refetchs - remove if/when we move to event stream method
  const [isInitialLoad, setIsInitialLoad] = useState(true)

  useEffect(() => {
    if (executions) {
      setIsInitialLoad(false)
    }
  }, [executions])

  useSpaceSSE({
    space,
    events: ['execution_updated', 'execution_completed', 'execution_canceled', 'execution_running'],
    onEvent: data => {
      // ideally this would include number - so we only check for executions on the page - but what if new executions are kicked off? - could check for ids that are higher than the lowest id on the page?
      if (
        executions?.some(
          execution => execution.repo_id === data?.repo_id && execution.pipeline_id === data?.pipeline_id
        )
      ) {
        //TODO - revisit full refresh - can I use the message to update the execution?
        executionsRefetch()
      }
    }
  })

  const { openModal: openRunPipelineModal } = useRunPipelineModal()

  const handleClick = async () => {
    if (repoMetadata && pipeline) {
      openRunPipelineModal({ repoMetadata, pipeline })
    }
  }

  const NewExecutionButton = (
    <Button
      text={getString('run')}
      variation={ButtonVariation.PRIMARY}
      icon="play-outline"
      onClick={handleClick}></Button>
  )

  const columns: Column<TypesExecution>[] = useMemo(
    () => [
      {
        Header: getString('executions.description'),
        width: 'calc(100% - 180px)',
        Cell: ({ row }: CellProps<TypesExecution>) => {
          const record = row.original
          return (
            <Layout.Vertical className={css.nameContainer}>
              <Layout.Horizontal spacing={'small'} style={{ alignItems: 'center' }}>
                <ExecutionStatus status={getStatus(record.status)} iconOnly noBackground iconSize={20} isCi />
                <Text className={css.number}>{`#${record.number}.`}</Text>
                <Text className={css.desc}>{record.message || record.title}</Text>
              </Layout.Horizontal>
              <ExecutionText
                authorEmail={record.author_email as string}
                authorName={record.author_name as string}
                repoPath={repoMetadata?.path as string}
                commitRef={record.after as string}
                event={record.event as ExecutionTrigger}
                action={record.action as EnumTriggerAction}
                target={record.target as string}
                beforeRef={record.before as string}
                source={record.source as string}
              />
            </Layout.Vertical>
          )
        }
      },
      {
        Header: getString('executions.time'),
        width: '180px',
        Cell: ({ row }: CellProps<TypesExecution>) => {
          const record = row.original

          // Determine if the execution is active.
          const isActive = record.status === ExecutionState.RUNNING

          return (
            <Layout.Vertical spacing={'small'}>
              {record?.started && (isActive || record?.finished) && (
                <Layout.Horizontal spacing={'small'} style={{ alignItems: 'center' }}>
                  <Timer color={Utils.getRealCSSColor(Color.GREY_500)} />
                  <Text inline color={Color.GREY_500} lineClamp={1} width={180} font={{ size: 'small' }}>
                    {/* Use live time when running, static time when finished */}
                    {timeDistance(record.started, isActive ? currentTime : record.finished, true)}
                  </Text>
                </Layout.Horizontal>
              )}
              {record?.finished && (
                <Layout.Horizontal spacing={'small'} style={{ alignItems: 'center' }}>
                  <Calendar color={Utils.getRealCSSColor(Color.GREY_500)} />
                  <Text inline color={Color.GREY_500} lineClamp={1} width={180} font={{ size: 'small' }}>
                    {timeDistance(record.finished, currentTime, true)} ago
                  </Text>
                </Layout.Horizontal>
              )}
            </Layout.Vertical>
          )
        },
        disableSortBy: true
      }
    ],
    [currentTime, getString, repoMetadata?.path]
  )

  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('pageTitle.executions')}
        dataTooltipId="repositoryExecutions"
        extraBreadcrumbLinks={
          repoMetadata && [
            {
              label: getString('pageTitle.pipelines'),
              url: routes.toCODEPipelines({ repoPath: repoMetadata.path as string })
            }
          ]
        }
      />
      <PageBody
        className={cx({ [css.withError]: !!error })}
        error={error ? getErrorMessage(error || executionsError) : null}
        retryOnError={voidFn(refetch)}
        noData={{
          when: () => executions?.length === 0,
          image: noExecutionImage,
          message: getString('executions.noData'),
          button: NewExecutionButton
        }}>
        <LoadingSpinner visible={loading || isInitialLoad} withBorder={!!executions && isInitialLoad} />

        <Container padding="xlarge">
          <Layout.Horizontal spacing="large" className={css.layout}>
            <Layout.Horizontal spacing="medium">
              {NewExecutionButton}
              <Button
                variation={ButtonVariation.SECONDARY}
                text={getString('edit')}
                onClick={e => {
                  e.stopPropagation()
                  if (repoMetadata?.path && pipeline) {
                    history.push(routes.toCODEPipelineEdit({ repoPath: repoMetadata.path, pipeline }))
                  }
                }}
              />
            </Layout.Horizontal>
            <FlexExpander />
            <Button
              variation={ButtonVariation.TERTIARY}
              text={getString('pipelines.settings')}
              onClick={e => {
                e.stopPropagation()
                if (repoMetadata?.path && pipeline) {
                  history.push(routes.toCODEPipelineSettings({ repoPath: repoMetadata.path, pipeline }))
                }
              }}
            />
          </Layout.Horizontal>

          <Container margin={{ top: 'medium' }}>
            {!!executions?.length && (
              <Table<TypesExecution>
                columns={columns}
                data={executions || []}
                onRowClick={executionInfo =>
                  history.push(
                    routes.toCODEExecution({
                      repoPath: repoMetadata?.path as string,
                      pipeline: pipeline as string,
                      execution: String(executionInfo.number)
                    })
                  )
                }
              />
            )}

            <NoResultCard showWhen={() => !!executions && executions.length === 0} forSearch={true} />
          </Container>
          <ResourceListingPagination response={response} page={page} setPage={setPage} />
        </Container>
      </PageBody>
    </Container>
  )
}

export default ExecutionList
