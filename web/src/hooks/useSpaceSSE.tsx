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

import { useEffect, useRef, useState } from 'react'
import { isEqual } from 'lodash-es'
import { EventSourcePolyfill } from 'event-source-polyfill'
import { useAppContext } from 'AppContext'
import { getConfig } from 'services/config'

type UseSpaceSSEProps = {
  space: string
  events: string[]
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  onEvent: (data: any, type: string) => void
  onError?: (event: Event) => void
  shouldRun?: boolean
}

const useSpaceSSE = ({ space, events: _events, onEvent, onError, shouldRun = true }: UseSpaceSSEProps) => {
  const { standalone, routingId, hooks } = useAppContext()
  const [events, setEvents] = useState(_events)
  const eventSourceRef = useRef<EventSource | null>(null)
  const bearerToken = hooks?.useGetToken?.() || ''

  useEffect(() => {
    if (!isEqual(events, _events)) {
      setEvents(_events)
    }
  }, [_events, setEvents, events])

  useEffect(() => {
    // Conditionally establish the event stream - don't want to open on a finished execution
    if (shouldRun && events.length > 0) {
      if (!eventSourceRef.current) {
        const pathAndQuery = getConfig(
          `code/api/v1/spaces/${space}/+/events${standalone ? '' : `?routingId=${routingId}`}`
        )
        const options: { heartbeatTimeout: number; headers?: { Authorization?: string } } = {
          heartbeatTimeout: 999999999
        }

        if (!standalone) {
          options.headers = { Authorization: `Bearer ${bearerToken}` }
        }

        eventSourceRef.current = new EventSourcePolyfill(pathAndQuery, options)
        const handleMessage = (event: MessageEvent) => {
          const data = JSON.parse(event.data)
          onEvent(data, event.type)
        }

        const handleError = (event: Event) => {
          if (onError) onError(event)
          eventSourceRef?.current?.close()
        }

        // always register error
        eventSourceRef.current.addEventListener('error', handleError)

        // register requested events
        for (const i in events) {
          const eventType = events[i]
          eventSourceRef.current.addEventListener(eventType, handleMessage)
        }

        return () => {
          eventSourceRef.current?.removeEventListener('error', handleError)
          for (const i in events) {
            const eventType = events[i]
            eventSourceRef.current?.removeEventListener(eventType, handleMessage)
          }
          eventSourceRef.current?.close()
          eventSourceRef.current = null
        }
      }
    } else {
      // If shouldRun is false, close and cleanup any existing stream
      if (eventSourceRef.current) {
        eventSourceRef.current.close()
        eventSourceRef.current = null
      }
    }
  }, [space, events, shouldRun, onEvent, onError, routingId, standalone, bearerToken])
}

export enum SSEEvents {
  PULLREQ_UPDATED = 'pullreq_updated'
}

export default useSpaceSSE
