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

import React, { FC } from 'react'
import cx from 'classnames'
import { Container, Layout, SelectOption, Text } from '@harnessio/uicore'
import css from './ToggleTabsBtn.module.scss'

interface ToggleTabsBtnProps {
  currentTab: string
  tabsList: SelectOption[]
  onTabChange: (newTab: string) => void
  wrapperClassName?: string
}

const ToggleTabsBtn: FC<ToggleTabsBtnProps> = ({ currentTab, tabsList, onTabChange, wrapperClassName }) => {
  return (
    <Layout.Horizontal className={cx(css.toggleTabs, wrapperClassName)}>
      <Container className={css.stateToggle}>
        {tabsList.map(tab => (
          <Text
            key={tab.value as string}
            className={cx(css.stateCtn, { [css.isSelected]: currentTab === tab.value })}
            onClick={() => onTabChange(tab.value as string)}>
            {tab.label}
          </Text>
        ))}
      </Container>
    </Layout.Horizontal>
  )
}

export default ToggleTabsBtn
