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
import type { PolicyFailureDetailCategoryV3 } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'

import Badge from './Badge'
import css from './Badge.module.scss'

interface ScanBadgeProps {
  category: PolicyFailureDetailCategoryV3
}

export default function ScanCategoryBadge(props: ScanBadgeProps): JSX.Element {
  const { category } = props
  const { getString } = useStrings()
  switch (category) {
    case 'License':
      return <Badge className={css.yellowStatus}>{getString('scanCategory.license')}</Badge>
    case 'PackageAge':
      return <Badge className={css.warningStatus}>{getString('scanCategory.packageAge')}</Badge>
    case 'Security':
      return <Badge className={css.blockedStatus}>{getString('scanCategory.security')}</Badge>
    case 'OssRiskLevel':
      return <Badge className={css.visibilityBadge}>{getString('scanCategory.ossRiskLevel')}</Badge>
    default:
      return <Badge className={css.visibilityBadge}>{getString('scanCategory.unknown')}</Badge>
  }
}
