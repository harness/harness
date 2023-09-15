import { useState, useEffect } from 'react'

const useLiveTimer = (isActive = true): number => {
  const [currentTime, setCurrentTime] = useState(Date.now())

  useEffect(() => {
    let intervalId: NodeJS.Timeout

    if (isActive) {
      intervalId = setInterval(() => {
        setCurrentTime(Date.now())
      }, 1000)
    }

    return () => {
      if (intervalId) {
        clearInterval(intervalId)
      }
    }
  }, [isActive])

  return currentTime
}

export default useLiveTimer
