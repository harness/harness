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
import type { IconName } from '@harnessio/icons'
import type { FirewallExceptionStatusV3 } from '@harnessio/react-har-service-client'

import type { StringsMap } from '@ar/strings/types'
import { useStrings } from '@ar/frameworks/strings/String'

import Badge from './Badge'
import css from './Badge.module.scss'

interface ExemptionStatusBadgeConfig {
  icon: IconName
  label: keyof StringsMap
  className?: string
}

const ExemptionStatusBadgeConfigMapping: Record<string, ExemptionStatusBadgeConfig> = {
  APPROVED: { icon: 'tick-circle', label: 'exemptionBadges.approved', className: css.passedStatus },
  REJECTED: { icon: 'circle-cross', label: 'exemptionBadges.rejected', className: css.blockedStatus },
  PENDING: { icon: 'status-pending', label: 'exemptionBadges.pending', className: css.warningStatus },
  EXPIRED: { icon: 'expired', label: 'exemptionBadges.expired', className: css.visibilityBadge }
}

const UnknownScopeBadgeConfig: ExemptionStatusBadgeConfig = {
  icon: 'status-pending',
  label: 'exemptionBadges.pending',
  className: css.warningStatus
}

interface ExemptionStatusBadgeProps {
  status: FirewallExceptionStatusV3
  helperText?: string
}

function ExemptionStatusBadge({ status, helperText }: ExemptionStatusBadgeProps): JSX.Element {
  const { getString } = useStrings()
  const badgeConfig = ExemptionStatusBadgeConfigMapping[status] || UnknownScopeBadgeConfig
  return (
    <Badge tooltip={helperText} icon={badgeConfig.icon} iconProps={{ size: 16 }} className={badgeConfig.className}>
      {getString(badgeConfig.label)}
    </Badge>
  )
}

export default ExemptionStatusBadge
