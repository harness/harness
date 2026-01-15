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

import { useStrings } from '@ar/frameworks/strings'

import Badge from './Badge'
import css from './Badge.module.scss'

export enum AvailablityBadgeType {
  ARCHIVED = 'ARCHIVED',
  AVAILABLE = 'AVAILABLE'
}

interface AvailablityBadgeProps {
  type: AvailablityBadgeType
}

function ArchivedBadge() {
  const { getString } = useStrings()
  return <Badge className={css.archivedBadge}>{getString('status.archived')}</Badge>
}

export default function AvailablityBadge(props: AvailablityBadgeProps): JSX.Element {
  const { type } = props
  switch (type) {
    case AvailablityBadgeType.ARCHIVED:
      return <ArchivedBadge />
    case AvailablityBadgeType.AVAILABLE:
    default:
      return <></>
  }
}
