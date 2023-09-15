import { useState, useEffect } from 'react'

/**
 * useLiveTimer returns the current time, updated every second.
 *
 * @param isActive - If true, the timer is active and updates every second.
 */
export function useLiveTimer(isActive: boolean): number {
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
