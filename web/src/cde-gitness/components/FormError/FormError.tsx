import { Color } from '@harnessio/design-system'
import { Text } from '@harnessio/uicore'
import React from 'react'

function FormError({ message }: { message: string }) {
  return message ? (
    <Text
      icon={'circle-cross'}
      color={Color.RED_600}
      style={{ paddingTop: '8px' }}
      iconProps={{
        color: Color.RED_600,
        size: 12
      }}>
      {message}
    </Text>
  ) : (
    <></>
  )
}

export default FormError
