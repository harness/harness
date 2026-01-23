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
import type { ArtifactScan } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'

import Badge from './Badge'
import css from './Badge.module.scss'

interface ScanBadgeProps {
  status: ArtifactScan['scanStatus']
}

export default function ScanBadge(props: ScanBadgeProps): JSX.Element {
  const { status } = props
  const { getString } = useStrings()
  switch (status) {
    case 'BLOCKED':
      return (
        <Badge className={css.blockedStatus} icon="warning-sign" iconProps={{ size: 12 }}>
          {getString('status.blocked')}
        </Badge>
      )
    case 'WARN':
      return (
        <Badge className={css.warningStatus} icon="warning-icon" iconProps={{ size: 12 }}>
          {getString('status.warning')}
        </Badge>
      )
    default:
      return <></>
  }
}
