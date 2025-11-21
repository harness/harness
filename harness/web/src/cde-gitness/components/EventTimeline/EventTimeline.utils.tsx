/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react'
import moment from 'moment'
import { Text, Layout } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'

export const convertToMilliSecs = (timestamp: number) => {
  if (timestamp > 1e12) {
    timestamp = Math.floor(timestamp / 1e6)
  }
  return timestamp
}

export const formatTimestamp = (timestamp: number) => {
  const convertedTimeStamp = convertToMilliSecs(timestamp)

  const inputDate = moment(convertedTimeStamp)
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
