import React, { useEffect } from 'react'
import { useAppContext } from 'AppContext'

const LogStreaming: React.FC<any> = ({ logKeyList, onMessageStreaming, onError }: any) => {
  const { hooks } = useAppContext()
  const { subscribe, closeStream } = hooks?.useLogsStreaming(logKeyList, onMessageStreaming, onError)

  useEffect(() => {
    subscribe()

    return () => closeStream()
  }, [])
  return <></>
}

export default LogStreaming
