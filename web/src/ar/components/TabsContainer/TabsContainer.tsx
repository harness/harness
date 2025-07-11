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
import { Container } from '@harnessio/uicore'
import type { ContainerProps } from '@harnessio/uicore/dist/components/Container/Container'

import css from './TabsContainer.module.scss'

function TabsContainer(props: React.PropsWithChildren<ContainerProps>) {
  return <Container {...props} className={classNames(props.className, css.tabsContainer)} />
}

export default TabsContainer
