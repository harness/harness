import { Color, Text } from '@harness/uicore'
import React from 'react'

export const PipeSeparator: React.FC<{ height?: number }> = ({ height }) => (
  <Text inline style={{ fontSize: height ? `${height}px` : undefined, alignSelf: 'center' }} color={Color.GREY_200}>
    |
  </Text>
)
