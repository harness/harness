import React from 'react'
import { Container } from '@harnessio/uicore'
import type { DelegateSelectorsV2Props } from 'utils/types'

export const defaultDelegateSelectorsV2 = (props: DelegateSelectorsV2Props): React.ReactElement => {
  return <Container {...(props as any)} />
}
