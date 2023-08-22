import React from 'react'
import { Container, PageHeader } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import css from './NewPipeline.module.scss'

const NewPipeline = () => {
  const { getString } = useStrings()
  return (
    <Container className={css.main}>
      <PageHeader title={getString('pipelines.newPipelineButton')} />
    </Container>
  )
}

export default NewPipeline
