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

import React, { useEffect, useRef } from 'react'
import cx from 'classnames'
import { Icon } from '@harnessio/icons'
import { Container, Layout, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import moment from 'moment'
import { ThreadSection } from 'components/ThreadSection/ThreadSection'
import { useStrings } from 'framework/strings'

import type { OpenapiGetGitspaceEventResponse } from 'services/cde'
import css from './EventsTimeline.module.scss'

const formatTimestamp = (timestamp: number) => {
  const inputDate = moment(timestamp)
  const currentDate = moment()

  if (inputDate.isSame(currentDate, 'day')) {
    // If the date is the same, show only the time
    return inputDate.format('HH:mm:ss')
  } else {
    // If the date is different, show both date and time
    return inputDate.format('YYYY-MM-DD HH:mm:ss')
  }
}

const EventsTimeline = ({ events }: { events: OpenapiGetGitspaceEventResponse[] | null }) => {
  const { getString } = useStrings()
  const localRef = useRef<HTMLDivElement | null>(null)
  const scrollContainerRef = useRef<HTMLDivElement | null>(null)
  const features = events

  useEffect(() => {
    if (scrollContainerRef.current) {
      const scrollParent = scrollContainerRef.current

      // Check if the user is already at the bottom or top of the scroll
      const autoScroll = scrollParent.scrollTop === scrollParent.scrollHeight - scrollParent.clientHeight

      if (autoScroll || scrollParent.scrollTop === 0) {
        // Scroll to the bottom
        scrollParent.scrollTop = scrollParent.scrollHeight
      }
    }
  }, [events])

  return (
    <Container padding={'medium'} className={css.featureContainer} ref={scrollContainerRef}>
      <Container ref={localRef}>
        <Text className={css.featureText} color={Color.GREY_400} padding={{ top: 'large', bottom: 'small' }}>
          {getString('cde.eventTimeline')}
        </Text>
        {features?.map((feature: OpenapiGetGitspaceEventResponse) => (
          <ThreadSection
            key={`${feature.entity_uid}`}
            title={
              <Layout.Horizontal className={css.releasedContainer}>
                <Icon className={css.iconContainer} name="dot" size={16} />
                <Container padding="xsmall">
                  <Container className={css.tagBackground} padding="xsmall">
                    <Text className={cx(css.tagText)}>{feature.message}</Text>
                  </Container>
                </Container>
              </Layout.Horizontal>
            }>
            <Container>
              <Layout.Vertical>
                <Text className={css.featureTitle}>{formatTimestamp(feature.timestamp || 0)}</Text>
              </Layout.Vertical>
            </Container>
          </ThreadSection>
        ))}
      </Container>
    </Container>
  )
}

export default EventsTimeline
