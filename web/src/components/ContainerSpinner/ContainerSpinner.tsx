import React from 'react'
import cx from 'classnames'
import { Container, PageSpinner } from '@harness/uicore'
import css from './ContainerSpinner.module.scss'

export const ContainerSpinner: React.FC<React.ComponentProps<typeof Container>> = ({ className, ...props }) => {
  return (
    <Container className={cx(css.spinner, className)} {...props}>
      <PageSpinner />
    </Container>
  )
}
