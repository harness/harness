import React from 'react'
import { Container, PageHeader } from '@harness/uicore'
import { useStrings } from 'framework/strings'
import css from './SecretList.module.scss'

const PipelineList = () => {
  const { getString } = useStrings()

  return (
    <Container className={css.main}>
      <PageHeader title={getString('pageTitle.secrets')} />
    </Container>
  )
}

export default PipelineList
