/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { PropsWithChildren, ReactNode } from 'react'
import { defaultTo } from 'lodash-es'
import { Collapse, Container, Layout, Text, useToggleOpen } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'

import css from './CollapseContainer.module.scss'

interface CollapseContainerProps {
  title: string | ReactNode
  subTitle?: string | ReactNode
  className?: string
  initialState?: boolean
}
export default function CollapseContainer({
  title,
  children,
  className,
  subTitle,
  initialState
}: PropsWithChildren<CollapseContainerProps>): JSX.Element {
  const { isOpen, toggle } = useToggleOpen(defaultTo(initialState, false))

  return (
    <Container className={className}>
      <Layout.Vertical spacing="small" className={css.toggleHandler} onClick={toggle}>
        <Text
          className={css.cardHeading}
          font={{ variation: FontVariation.CARD_TITLE }}
          rightIcon={isOpen ? 'chevron-up' : 'chevron-down'}
          rightIconProps={{ size: 17, color: Color.PRIMARY_7 }}>
          {title}
        </Text>
        {subTitle && (
          <Text className={css.subHeading} font={{ variation: FontVariation.BODY }}>
            {subTitle}
          </Text>
        )}
      </Layout.Vertical>
      <Collapse
        isOpen={isOpen}
        collapseClassName={css.collapseContainer}
        className={css.collapseContainer}
        collapseHeaderClassName={css.collapseHeader}>
        <Container className={css.collapseContentContainer}>{children}</Container>
      </Collapse>
    </Container>
  )
}
