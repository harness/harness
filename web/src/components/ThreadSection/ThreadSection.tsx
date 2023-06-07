import React from 'react'
import { Container, Layout } from '@harness/uicore'
import cx from 'classnames'
import css from './ThreadSection.module.scss'

interface ThreadSectionProps {
  title: JSX.Element
  className?: string
  contentClassName?: string
  hideGutter?: boolean
  hideTitleGutter?: boolean
  onlyTitle?: boolean
  inCommentBox?: boolean
  lastItem?: boolean
}

export const ThreadSection: React.FC<ThreadSectionProps> = ({
  title,
  children,
  className,
  contentClassName,
  hideGutter,
  hideTitleGutter,
  onlyTitle,
  inCommentBox = false,
  lastItem
}) => {
  return (
    <Container
      className={cx(
        inCommentBox ? css.thread : css.threadLessSpace,
        hideTitleGutter ? css.hideTitleGutter : '',
        className,
        {
          [css.titleContent]: onlyTitle && !inCommentBox && !lastItem,
          [css.inCommentBox]: inCommentBox && !lastItem
        }
      )}>
      <Layout.Vertical spacing={'medium'}>
        {title}
        <Container className={cx(css.content, contentClassName, hideGutter ? css.hideGutter : '')}>
          {children}
        </Container>
      </Layout.Vertical>
    </Container>
  )
}
