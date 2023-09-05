import { Container, PageBody } from '@harnessio/uicore'
import React, { useState } from 'react'
import cx from 'classnames'
import { useParams } from 'react-router-dom'
import { useGet } from 'restful-react'
import SplitPane from 'react-split-pane'
import { routes, type CODEProps } from 'RouteDefinitions'
import type { TypesExecution } from 'services/code'
import ExecutionStageList from 'components/ExecutionStageList/ExecutionStageList'
import Console from 'components/Console/Console'
import { getErrorMessage, voidFn } from 'utils/Utils'
import { useStrings } from 'framework/strings'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { ExecutionPageHeader } from 'components/ExecutionPageHeader/ExecutionPageHeader'
import noExecutionImage from '../RepositoriesListing/no-repo.svg'
import css from './Execution.module.scss'

const Execution = () => {
  const { pipeline, execution: executionNum } = useParams<CODEProps>()
  const { getString } = useStrings()

  const { repoMetadata, error, loading, refetch } = useGetRepositoryMetadata()

  const {
    data: execution,
    error: executionError,
    loading: executionLoading
  } = useGet<TypesExecution>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/pipelines/${pipeline}/executions/${executionNum}`,
    lazy: !repoMetadata
  })

  const [selectedStage, setSelectedStage] = useState<number | null>(1)

  return (
    <Container className={css.main}>
      <ExecutionPageHeader
        repoMetadata={repoMetadata}
        title={execution?.title as string}
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
          message: execution?.message as string,
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
        className={cx({ [css.withError]: !!error })}
        error={error ? getErrorMessage(error || executionError) : null}
        retryOnError={voidFn(refetch)}
        noData={{
          when: () => !execution && !loading && !executionLoading,
          image: noExecutionImage,
          message: getString('executions.noData')
          // button: NewExecutionButton
        }}>
        <LoadingSpinner visible={loading || executionLoading} withBorder={!!execution && executionLoading} />
        {execution && (
          <SplitPane split="vertical" size={300} minSize={200} maxSize={400}>
            <ExecutionStageList
              stages={execution?.stages || []}
              setSelectedStage={setSelectedStage}
              selectedStage={selectedStage}
            />
            {selectedStage && (
              <Console stage={execution?.stages?.[selectedStage - 1]} repoPath={repoMetadata?.path as string} />
            )}
          </SplitPane>
        )}
      </PageBody>
    </Container>
  )
}

export default Execution
