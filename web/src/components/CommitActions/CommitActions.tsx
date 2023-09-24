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

import React, { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import cx from 'classnames'
import { Container, Layout, Button, ButtonVariation, Utils, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import { CodeIcon } from 'utils/GitUtils'
import css from './CommitActions.module.scss'

interface CommitActionButtonProps {
  sha: string
  href: string
  enableCopy?: boolean
}

export function CommitActions({ sha, href, enableCopy }: CommitActionButtonProps) {
  const { getString } = useStrings()
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    let timeoutId: number
    if (copied) {
      timeoutId = window.setTimeout(() => setCopied(false), 2500)
    }
    return () => {
      clearTimeout(timeoutId)
    }
  }, [copied])

  return (
    <Container className={css.container}>
      <Layout.Horizontal className={cx(css.layout, !enableCopy ? css.noCopy : '')}>
        <Link to={href}>
          <Text tooltip={getString('viewCommitDetails')} inline>
            {sha.substring(0, 6)}
          </Text>
        </Link>
        {enableCopy && (
          <Button
            id={css.commitCopyButton}
            variation={ButtonVariation.ICON}
            icon={copied ? 'tick' : CodeIcon.Copy}
            iconProps={{ size: 14, color: copied ? Color.GREEN_500 : undefined }}
            onClick={() => {
              setCopied(true)
              Utils.copy(sha)
            }}
            tooltip={getString('copyCommitSHA')}
          />
        )}
      </Layout.Horizontal>
    </Container>
  )
}
