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

import React, { CSSProperties } from 'react'
import { Container, Popover, StringSubstitute, Text } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import css from './CommitDivergence.module.scss'

interface CommitDivergenceProps {
  behind: number
  ahead: number
  defaultBranch: string
}

export function CommitDivergence({ behind, ahead, defaultBranch }: CommitDivergenceProps) {
  const { getString } = useStrings()
  const message =
    behind === 0 ? (
      ahead === 0 ? (
        getString('branchUpToDate', { defaultBranch })
      ) : (
        <StringSubstitute
          str={getString('branchDivergenceAhead')}
          vars={{
            ahead,
            aheadCommits: ahead,
            defaultBranch
          }}
        />
      )
    ) : ahead === 0 ? (
      <StringSubstitute
        str={getString('branchDivergenceBehind')}
        vars={{
          behind,
          behindCommits: behind,
          defaultBranch
        }}
      />
    ) : (
      <StringSubstitute
        str={getString('branchDivergenceAheadBehind')}
        vars={{
          ahead,
          aheadCommits: ahead,
          behind,
          behindCommits: behind,
          defaultBranch
        }}
      />
    )

  return (
    <Popover content={<Text padding="small">{message}</Text>} interactionKind="hover">
      <Container className={css.container}>
        <Container className={css.main}>
          <Text className={css.behind} style={{ '--bar-size': `${behind}%` } as CSSProperties}>
            <span>{behind}</span>
          </Text>
          <span className={css.pipe} />
          <Text className={css.ahead} style={{ '--bar-size': `${ahead}%` } as CSSProperties}>
            <span>{ahead}</span>
          </Text>
        </Container>
      </Container>
    </Popover>
  )
}

// TODO: --bar-size is not calculated precisely. Need some more work.
