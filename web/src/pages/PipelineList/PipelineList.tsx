import React from 'react'
import { Container, PageHeader } from '@harness/uicore'
import { useStrings } from 'framework/strings'
import css from './PipelineList.module.scss'

const PipelineList = () => {
  const { getString } = useStrings()

  return (
    <Container className={css.main}>
      <PageHeader title={getString('pageTitle.pipelines')} />
    </Container>
  )
}

export default PipelineList
