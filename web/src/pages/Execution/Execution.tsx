import { Container, PageBody } from '@harnessio/uicore'
import React, { useEffect, useState } from 'react'
import cx from 'classnames'
import { useParams } from 'react-router-dom'
import { useGet } from 'restful-react'
import { routes, type CODEProps } from 'RouteDefinitions'
import type { TypesExecution } from 'services/code'
import ExecutionStageList from 'components/ExecutionStageList/ExecutionStageList'
import Console from 'components/Console/Console'
import { getErrorMessage, voidFn } from 'utils/Utils'
import { useStrings } from 'framework/strings'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { Split } from 'components/Split/Split'
import { ExecutionPageHeader } from 'components/ExecutionPageHeader/ExecutionPageHeader'
import useSpaceSSE from 'hooks/useSpaceSSE'
import { ExecutionState } from 'components/ExecutionStatus/ExecutionStatus'
import { getStatus } from 'utils/ExecutionUtils'
import noExecutionImage from '../RepositoriesListing/no-repo.svg'
import css from './Execution.module.scss'

const Execution = () => {
  const { pipeline, execution: executionNum } = useParams<CODEProps>()
  const { getString } = useStrings()

  const { repoMetadata, error, loading, refetch, space } = useGetRepositoryMetadata()

  const {
    data: execution,
    error: executionError,
    loading: executionLoading,
    refetch: executionRefetch
  } = useGet<TypesExecution>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/pipelines/${pipeline}/executions/${executionNum}`,
    lazy: !repoMetadata
  })

  //TODO remove null type here?
  const [selectedStage, setSelectedStage] = useState<number | null>(1)
  //TODO - do not want to show load between refetchs - remove if/when we move to event stream method
  const [isInitialLoad, setIsInitialLoad] = useState(true)

  useEffect(() => {
    if (execution) {
      setIsInitialLoad(false)
    }
  }, [execution])

  useSpaceSSE({
    space,
    events: ['execution_updated', 'execution_completed', 'execution_canceled', 'execution_running'],
    onEvent: data => {
      if (
        data?.repo_id === execution?.repo_id &&
        data?.pipeline_id === execution?.pipeline_id &&
        data?.number === execution?.number
      ) {
        //TODO - revisit full refresh - can I use the message to update the execution?
        executionRefetch()
      }
    },
    shouldRun: [ExecutionState.RUNNING, ExecutionState.PENDING].includes(getStatus(execution?.status))
  })

  return (
    <Container className={css.main}>
      <ExecutionPageHeader
        repoMetadata={repoMetadata}
        title={pipeline as string}
        dataTooltipId="repositoryExecution"
        extraBreadcrumbLinks={
          repoMetadata && [
            {
              label: getString('pageTitle.pipelines'),
              url: routes.toCODEPipelines({ repoPath: repoMetadata.path as string })
            },
            {
              label: getString('pageTitle.executions'),
              url: routes.toCODEExecutions({ repoPath: repoMetadata.path as string, pipeline: pipeline as string })
            }
          ]
        }
        executionInfo={{
          message: (execution?.message || execution?.title) as string,
          authorName: execution?.author_name as string,
          authorEmail: execution?.author_email as string,
          source: execution?.source as string,
          hash: execution?.after as string,
          status: execution?.status as string,
          started: execution?.started as number,
          finished: execution?.finished as number
        }}
      />
      <PageBody
        className={cx(css.pageBody, { [css.withError]: !!error })}
        error={error ? getErrorMessage(error || executionError) : null}
        retryOnError={voidFn(refetch)}
        noData={{
          when: () => !execution && !loading && !executionLoading,
          image: noExecutionImage,
          message: getString('executions.noData')
          // button: NewExecutionButton
        }}>
        <LoadingSpinner visible={loading || isInitialLoad} withBorder={!!execution && isInitialLoad} />
        {execution && (
          <Split split="vertical" size={300} minSize={200} maxSize={400}>
            <ExecutionStageList
              stages={execution?.stages || []}
              setSelectedStage={setSelectedStage}
              selectedStage={selectedStage}
            />
            {selectedStage && (
              <Console stage={execution?.stages?.[selectedStage - 1]} repoPath={repoMetadata?.path as string} />
            )}
          </Split>
        )}
      </PageBody>
    </Container>
  )
}

export default Execution
