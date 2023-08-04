import React from 'react'
import { Container, PageHeader } from '@harness/uicore'
import css from './Execution.module.scss'

const Execution = () => {
  return (
    <Container className={css.main}>
      <PageHeader title={'THIS IS AN EXECUTION'} />
    </Container>
  )
}

export default Execution
