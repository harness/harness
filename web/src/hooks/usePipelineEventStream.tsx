import { useEffect, useRef } from 'react'

type UsePipelineEventStreamProps = {
  space: string
  onEvent: (data: any) => void
  onError?: (event: Event) => void
  shouldRun?: boolean
}

const usePipelineEventStream = ({ space, onEvent, onError, shouldRun = true }: UsePipelineEventStreamProps) => {
  //TODO - this is not working right - need to get to the bottom of too many streams being opened and closed... can miss events!
  const eventSourceRef = useRef<EventSource | null>(null)

  useEffect(() => {
    // Conditionally establish the event stream - don't want to open on a finished execution
    if (shouldRun) {
      if (!eventSourceRef.current) {
        eventSourceRef.current = new EventSource(`/api/v1/spaces/${space}/stream`)

        const handleMessage = (event: MessageEvent) => {
          const data = JSON.parse(event.data)
          onEvent(data)
        }

        const handleError = (event: Event) => {
          if (onError) onError(event)
          eventSourceRef?.current?.close()
        }

        eventSourceRef.current.addEventListener('message', handleMessage)
        eventSourceRef.current.addEventListener('error', handleError)

        return () => {
          eventSourceRef.current?.removeEventListener('message', handleMessage)
          eventSourceRef.current?.removeEventListener('error', handleError)
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
  }, [space, shouldRun, onEvent, onError])
}

export default usePipelineEventStream
