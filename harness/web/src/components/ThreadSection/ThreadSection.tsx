/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react'
import { Container, Layout } from '@harnessio/uicore'
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
