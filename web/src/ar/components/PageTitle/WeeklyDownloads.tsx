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
import { FontVariation } from '@harnessio/design-system'
import { Container, Layout, Text } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'

import css from './PageTitle.module.scss'

interface WeeklyDownloadsProps {
  downloads: number | undefined
  label: string | React.ReactNode
}

function WeeklyDownloads(props: WeeklyDownloadsProps): JSX.Element {
  return (
    <Container flex={{ alignItems: 'flex-end' }}>
      <Icon name="bar-chart" size={40} />
      <Layout.Vertical className={css.weeklyDownloadContent}>
        <Text font={{ variation: FontVariation.H5 }}>{props.downloads || 0}</Text>
        <Text font={{ variation: FontVariation.SMALL }}>{props.label}</Text>
      </Layout.Vertical>
    </Container>
  )
}

export default WeeklyDownloads
