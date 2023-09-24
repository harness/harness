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
import { Link } from 'react-router-dom'
import { Layout } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { CopyButton } from 'components/CopyButton/CopyButton'
import { useStrings } from 'framework/strings'
import { CodeIcon } from 'utils/GitUtils'
import css from './GitRefLink.module.scss'

export const GitRefLink: React.FC<{ text: string; url: string; showCopy: boolean }> = ({
  text,
  url,
  showCopy = true
}) => {
  const { getString } = useStrings()

  return (
    <Layout.Horizontal className={css.link} inline>
      <Link className={css.linkText} to={url}>
        {text}
      </Link>
      {showCopy ? (
        <CopyButton
          className={css.copyContainer}
          content={text}
          tooltip={getString('copyBranch')}
          icon={CodeIcon.Copy}
          color={Color.PRIMARY_7}
          iconProps={{ size: 14, color: Color.PRIMARY_7 }}
          background={Color.PRIMARY_1}
        />
      ) : null}
    </Layout.Horizontal>
  )
}
