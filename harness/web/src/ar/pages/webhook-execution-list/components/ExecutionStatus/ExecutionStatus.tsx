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
import { Icon } from '@harnessio/icons'
import { Layout, Tag, Text } from '@harnessio/uicore'
import type { WebhookExecResult } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'

import { StatusConfigMap, UnknownStatusConfig } from './constants'

import css from './ExecutionStatus.module.scss'

interface ExecutionStatusProps {
  status: WebhookExecResult
  message?: string
}

export default function ExecutionStatus(props: ExecutionStatusProps) {
  const { status, message } = props
  const { getString } = useStrings()
  const config = StatusConfigMap[status] || UnknownStatusConfig
  return (
    <Tag className={classNames(css.status, css[config.className as keyof typeof css])}>
      <Layout.Horizontal flex={{ alignItems: 'center' }} spacing="small">
        <Icon name={config.iconName} />
        <Text tooltip={message} alwaysShowTooltip font={{ weight: 'bold' }}>
          {getString(config.label)}
        </Text>
      </Layout.Horizontal>
    </Tag>
  )
}
