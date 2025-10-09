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
import { Color } from '@harnessio/design-system'

import { EntityScope } from '@ar/common/types'
import type { StringsMap } from '@ar/strings/types'
import { useStrings } from '@ar/frameworks/strings/String'

import Badge from './Badge'

interface BadgeConfig {
  icon: IconName
  label: keyof StringsMap
  color: Color
}

const EntityScopeToBadgetMapping: Record<EntityScope, BadgeConfig> = {
  [EntityScope.ACCOUNT]: { icon: 'Account', label: 'badges.accountScope', color: Color.PRIMARY_7 },
  [EntityScope.ORG]: { icon: 'nav-organization', label: 'badges.orgScope', color: Color.PRIMARY_7 },
  [EntityScope.PROJECT]: { icon: 'nav-project', label: 'badges.projectScope', color: Color.PRIMARY_7 }
}

const UnknownScopeBadgeConfig: BadgeConfig = {
  icon: 'nav-project',
  label: 'badges.projectScope',
  color: Color.PRIMARY_7
}

interface ScopeBadgeProps {
  scope: EntityScope
  helperText?: string
}

function ScopeBadge({ scope, helperText }: ScopeBadgeProps): JSX.Element {
  const { getString } = useStrings()
  const badgeConfig = EntityScopeToBadgetMapping[scope] || UnknownScopeBadgeConfig
  return (
    <Badge tooltip={helperText} icon={badgeConfig.icon} iconProps={{ size: 16 }} color={badgeConfig.color}>
      {getString(badgeConfig.label)}
    </Badge>
  )
}

export default ScopeBadge
