import { useEffect, useRef } from 'react'

type UseSpaceSSEProps = {
  space: string
  events: string[]
  onEvent: (type: string, data: any) => void
  onError?: (event: Event) => void
  shouldRun?: boolean
}

const useSpaceSSE = ({ space, events, onEvent, onError, shouldRun = true }: UseSpaceSSEProps) => {
  //TODO - this is not working right - need to get to the bottom of too many streams being opened and closed... can miss events!
  const eventSourceRef = useRef<EventSource | null>(null)

  useEffect(() => {
    // Conditionally establish the event stream - don't want to open on a finished execution
    if (shouldRun && events.length > 0) {
      if (!eventSourceRef.current) {
        eventSourceRef.current = new EventSource(`/api/v1/spaces/${space}/+/events`)

        const handleMessage = (event: MessageEvent) => {
          const data = JSON.parse(event.data)
          onEvent(event.type, data)
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
  }, [space, events, shouldRun, onEvent, onError])
}

export default useSpaceSSE
