import React from 'react'
import moment from 'moment'
import { Text, Layout } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'

export const formatTimestamp = (timestamp: number) => {
  const inputDate = moment(timestamp)
  const currentDate = moment()

  if (inputDate.isSame(currentDate, 'day')) {
    return (
      <Text width="70%" color={Color.GREY_500}>
        {inputDate.format('HH:mm:ss')}
      </Text>
    )
  } else {
    return (
      <Layout.Vertical width="70%" spacing="small">
        <Text font={{ size: 'small' }} color={Color.GREY_500}>
          {inputDate.format('YYYY-MM-DD')}
        </Text>
        <Text font={{ size: 'small' }} color={Color.GREY_500}>
          {inputDate.format('HH:mm:ss')}
        </Text>
      </Layout.Vertical>
    )
  }
}
