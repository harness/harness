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
import { Position } from '@blueprintjs/core'
import { Container, Text } from '@harnessio/uicore'
import type { ArtifactVersionSummary } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { DEFAULT_DATE_TIME_FORMAT } from '@ar/constants'
import { getReadableDateTime } from '@ar/common/dateUtils'

import Badge from './Badge'
import css from './Badge.module.scss'

function ScanBadgeTooltip({ value }: { value: string }) {
  const { getString } = useStrings()
  return (
    <Container padding="medium">
      <Text>
        {getString('violationsList.evaluatedAt', {
          value: getReadableDateTime(Number(value), DEFAULT_DATE_TIME_FORMAT)
        })}
      </Text>
    </Container>
  )
}

interface ScanBadgeProps {
  scanId?: string
  evaluatedAt?: string
  status?: ArtifactVersionSummary['scanStatus']
  onClick?: () => void
}

export default function ScanBadge(props: ScanBadgeProps): JSX.Element {
  const { status, onClick, scanId } = props
  const { getString } = useStrings()
  if (!scanId) return <Text>{getString('versionList.table.notScanned')}</Text>
  switch (status) {
    case 'BLOCKED':
      return (
        <Badge
          className={css.blockedStatus}
          tooltip={<ScanBadgeTooltip value={props.evaluatedAt || ''} />}
          tooltipProps={{
            position: Position.TOP,
            disabled: !props.evaluatedAt
          }}
          icon="warning-sign"
          iconProps={{ size: 12 }}
          onClick={onClick}>
          {getString('status.blocked')}
        </Badge>
      )
    case 'WARN':
      return (
        <Badge
          className={css.warningStatus}
          tooltip={<ScanBadgeTooltip value={props.evaluatedAt || ''} />}
          tooltipProps={{
            position: Position.TOP,
            disabled: !props.evaluatedAt
          }}
          icon="warning-icon"
          iconProps={{ size: 12 }}
          onClick={onClick}>
          {getString('status.warning')}
        </Badge>
      )
    default:
      return (
        <Badge
          className={css.passedStatus}
          tooltip={<ScanBadgeTooltip value={props.evaluatedAt || ''} />}
          tooltipProps={{
            position: Position.TOP,
            disabled: !props.evaluatedAt
          }}
          icon="tick-circle"
          iconProps={{ size: 12 }}>
          {getString('status.passed')}
        </Badge>
      )
  }
}
