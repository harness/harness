import { Container, PageHeader, PageBody } from '@harnessio/uicore'
import React, { useState } from 'react'
import cx from 'classnames'
import { useParams } from 'react-router-dom'
import { useGet } from 'restful-react'
import SplitPane from 'react-split-pane'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import type { CODEProps } from 'RouteDefinitions'
import type { TypesExecution } from 'services/code'
import ExecutionStageList from 'components/ExecutionStageList/ExecutionStageList'
import Console from 'components/Console/Console'
import { getErrorMessage, voidFn } from 'utils/Utils'
import { useStrings } from 'framework/strings'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import noExecutionImage from '../RepositoriesListing/no-repo.svg'
import css from './Execution.module.scss'

const Execution = () => {
  const space = useGetSpaceParam()
  const { pipeline, execution: executionNum } = useParams<CODEProps>()
  const { getString } = useStrings()

  const {
    data: execution,
    error,
    loading,
    refetch
  } = useGet<TypesExecution>({
    path: `/api/v1/pipelines/${space}/${pipeline}/+/executions/${executionNum}`
  })

  const [selectedStage, setSelectedStage] = useState<number | null>(null)

  return (
    <Container className={css.main}>
      <PageHeader title={execution?.title} />
      <PageBody
        className={cx({ [css.withError]: !!error })}
        error={error ? getErrorMessage(error) : null}
        retryOnError={voidFn(refetch)}
        noData={{
          when: () => !execution,
          image: noExecutionImage,
          message: getString('executions.noData')
          // button: NewExecutionButton
        }}>
        <LoadingSpinner visible={loading} />
        <SplitPane split="vertical" size={300} minSize={200} maxSize={400}>
          <ExecutionStageList
            stages={execution?.stages || []}
            setSelectedStage={setSelectedStage}
            selectedStage={selectedStage}
          />
          {selectedStage && <Console stage={execution?.stages?.[selectedStage - 1]} />}
        </SplitPane>
      </PageBody>
    </Container>
  )
}

export default Execution
