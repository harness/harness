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
import { Container, Text, Layout } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import { convertToMilliSecs } from '../EventTimeline/EventTimeline.utils'

const EventTimelineSummary = ({ timestamp, message }: { timestamp?: number; message?: string }) => {
  const { getString } = useStrings()
  const convertedTimeStamp = convertToMilliSecs(timestamp || 0)
  return (
    <Container width="100%" flex={{ alignItems: 'center', justifyContent: 'space-between' }}>
      <Text font={{ variation: FontVariation.CARD_TITLE }} margin={{ left: 'large' }}>
        {getString('cde.details.gitspaceActivity')}
      </Text>
      {Boolean(message) && (
        <Layout.Horizontal spacing="large" flex={{ alignItems: 'center' }}>
          <Text iconProps={{ color: Color.GREEN_450 }} icon="tick-circle">
            {message}
          </Text>
          <Text margin={{ left: 'large' }} font={{ size: 'small' }}>
            {moment(convertedTimeStamp).format('DD MMM, YYYY hh:mma')}
          </Text>
        </Layout.Horizontal>
      )}
    </Container>
  )
}

export default EventTimelineSummary
