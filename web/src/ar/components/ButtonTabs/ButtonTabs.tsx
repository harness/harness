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

import React from 'react'
import classNames from 'classnames'
import { Button, Container, ButtonSize, ButtonGroup, ButtonVariation } from '@harnessio/uicore'
import type { IconName, IconProps } from '@harnessio/icons'

import css from './ButtonTabs.module.scss'

interface ButtonTabProps<T> {
  id: T
  title: React.ReactElement | string
  panel: React.ReactElement
  icon?: IconName
  iconProps?: Omit<IconProps, 'name'>
}

export function ButtonTab<T>(props: ButtonTabProps<T>): JSX.Element {
  return props.panel
}

interface ButtonTabsProps<T> {
  id?: T
  selectedTabId: T
  onChange: (newTab: T) => void
  children: React.ReactElement<ButtonTabProps<T>, typeof ButtonTab>[]
  small?: boolean
  bold?: boolean
  className?: string
}

export function ButtonTabs<T>(props: ButtonTabsProps<T>): JSX.Element {
  const { children: tabs, small, id, selectedTabId, onChange, bold, className } = props
  const selectedTabPannel = tabs.find(each => each.props.id === selectedTabId)
  return (
    <Container className={className} data-testid={id}>
      <ButtonGroup className={css.btnGroup}>
        {tabs.map(each => (
          <Button
            key={each.props.id as string}
            size={ButtonSize.SMALL}
            icon={each.props.icon}
            iconProps={each.props.iconProps}
            variation={ButtonVariation.SECONDARY}
            className={classNames({
              [css.small]: small,
              [css.bold]: bold,
              [css.selected]: each.props.id === props.selectedTabId
            })}
            data-testid={each.props.id}
            onClick={() => {
              onChange(each.props.id)
            }}
            text={each.props.title}
          />
        ))}
      </ButtonGroup>
      {selectedTabPannel && <Container className={css.content}>{selectedTabPannel}</Container>}
    </Container>
  )
}
