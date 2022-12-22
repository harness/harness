import React from 'react'
import { Container, Layout } from '@harness/uicore'
import cx from 'classnames'
import css from './ThreadSection.module.scss'

interface ThreadSectionProps {
  title: JSX.Element
  className?: string
  contentClassName?: string
  hideGutter?: boolean
}

export const ThreadSection: React.FC<ThreadSectionProps> = ({
  title,
  children,
  className,
  contentClassName,
  hideGutter
}) => {
  return (
    <Container className={cx(css.thread, className)}>
      <Layout.Vertical spacing="medium">
        {title}
        <Container className={cx(css.content, contentClassName, hideGutter ? css.hideGutter : '')}>
          {children}
        </Container>
      </Layout.Vertical>
    </Container>
  )
}
