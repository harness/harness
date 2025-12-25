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

import React from 'react'
import classNames from 'classnames'
import { Text, TextProps } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import css from './RepoTypeLabel.module.scss'

export const RepoTypeLabel: React.FC<{ isPublic?: boolean; isArchived?: boolean; margin?: TextProps['margin'] }> = ({
  isPublic,
  isArchived,
  margin
}) => {
  const { getString } = useStrings()

  const visibility = getString(isPublic ? 'public' : 'private')
  const archiveStatus = isArchived ? ` ${getString('archived')}` : ''

  return (
    <>
      <Text inline className={css.label} margin={margin}>
        {visibility}
      </Text>

      {isArchived && (
        <Text inline className={classNames(css.label, css.labelArchived)} margin={margin}>
          {archiveStatus}
        </Text>
      )}
    </>
  )
}
