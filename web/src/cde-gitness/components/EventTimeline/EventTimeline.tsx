import React, { useEffect, useRef, useState } from 'react'
import { Color } from '@harnessio/design-system'
import { Container, Text, Layout } from '@harnessio/uicore'
import { isArray, isEqual } from 'lodash-es'
import type { TypesGitspaceEventResponse } from 'cde-gitness/services'
import { useStrings } from 'framework/strings'
import { formatTimestamp } from './EventTimeline.utils'
import css from './EventTimeline.module.scss'

const EventTimeline = ({ data }: { data?: TypesGitspaceEventResponse[] | null }) => {
  const localRef = useRef<HTMLDivElement | null>(null)
  const scrollContainerRef = useRef<HTMLDivElement | null>(null)

  const { getString } = useStrings()
  const [cache, setCache] = useState(data)

  useEffect(() => {
    if (!isEqual(data, cache)) {
      setCache(data)
    }
  }, [data])

  useEffect(() => {
    if (scrollContainerRef.current) {
      const scrollParent = scrollContainerRef.current

      const autoScroll = scrollParent.scrollTop <= scrollParent.scrollHeight - scrollParent.clientHeight

      if (autoScroll || scrollParent.scrollTop === 0) {
        scrollParent.scrollTop = scrollParent.scrollHeight
      }
    }
  }, [cache])

  return (
    <Container className={css.main} ref={scrollContainerRef}>
      {!data?.length && isArray(data) && (
        <Container
          width={'100%'}
          padding={{ left: 'large' }}
          flex={{ alignItems: 'center' }}
          height={'64px'}
          border={{ left: true, color: Color.GREY_200 }}
          background={Color.PRIMARY_2}>
          <Text iconProps={{ color: Color.GREEN_450 }} icon="no-deployments">
            {getString('cde.details.fetchingDetails')}
          </Text>
        </Container>
      )}
      <Container ref={localRef}>
        {data?.map((item, index) => {
          return (
            <Layout.Horizontal background={Color.GREY_50} key={`${item.query_key}_${item.timestamp}`}>
              <Container
                background={Color.GREY_50}
                width={'8%'}
                flex={{ alignItems: 'center', justifyContent: 'center' }}>
                {formatTimestamp(item.timestamp || 0)}
              </Container>
              <Text className={css.marker} />
              <Container
                width={'92%'}
                padding={{ left: 'large' }}
                flex={{ alignItems: 'center' }}
                height={'64px'}
                border={{ left: true, color: Color.GREY_200 }}
                className={index % 2 ? css.lightBackground : css.darkBackground}>
                <Text iconProps={{ color: Color.GREEN_450 }} icon="tick-circle">
                  {`${item.message}`}
                </Text>
              </Container>
            </Layout.Horizontal>
          )
        })}
      </Container>
    </Container>
  )
}

export default EventTimeline
