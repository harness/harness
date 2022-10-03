import React from 'react'
import { Text, Color } from '@harness/uicore'
import css from './Repos.module.scss'

export function PinnedRibbon(): JSX.Element {
  return (
    <Text
      className={css.pinned}
      inline
      icon="pin"
      color={Color.PRIMARY_7}
      background={Color.PRIMARY_1}
      iconProps={{ size: 13 }}>
      PINNED
    </Text>
  )
}
