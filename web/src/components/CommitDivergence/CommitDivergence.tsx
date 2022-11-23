import React, { CSSProperties } from 'react'
import { Container, Popover, Text } from '@harness/uicore'
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
    behind === 0
      ? ahead === 0
        ? getString('branchUpToDate', { defaultBranch })
        : getString('branchDivergenceAhead', { defaultBranch, ahead })
      : ahead === 0
      ? getString('branchDivergenceBehind', { defaultBranch, behind })
      : getString('branchDivergenceAheadBehind', { defaultBranch, ahead, behind })

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
