import React from 'react'
import { Container, PageHeader } from '@harness/uicore'
import { useParams } from 'react-router-dom'
import { useGet } from 'restful-react'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import type { CODEProps } from 'RouteDefinitions'
import type { TypesExecution } from 'services/code'
import css from './Execution.module.scss'

const Execution = () => {
  const space = useGetSpaceParam()
  const { pipeline, execution: executionNum } = useParams<CODEProps>()

  const {
    data: execution
    // error,
    // loading,
    // refetch
    // response
  } = useGet<TypesExecution>({
    path: `/api/v1/pipelines/${space}/${pipeline}/+/executions/${executionNum}`
  })

  return (
    <Container className={css.main}>
      <PageHeader title={`EXECUTION STATUS = ${execution?.status}`} />
    </Container>
  )
}

export default Execution
