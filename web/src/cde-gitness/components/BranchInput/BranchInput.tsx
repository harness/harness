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
import cx from 'classnames'
import { FormInput } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import css from './BranchInput.module.scss'

export const BranchInput = ({ disabled }: { disabled?: boolean }) => {
  const { getString } = useStrings()
  return (
    <FormInput.Text
      name="branch"
      disabled={disabled}
      inputGroup={{ leftIcon: 'git-branch', color: Color.GREY_500 }}
      placeholder={getString('cde.branchPlaceholder')}
      className={cx(css.branchDropdown, css.branchInput)}
    />
  )
}
