import React from 'react'
import { Container } from '@harness/uicore'
import css from './Welcome.module.scss'

export const Welcome: React.FC = () => {
  return <Container className={css.main}>Welcome to Harness SCM (from MFE)!</Container>
}

export default Welcome
