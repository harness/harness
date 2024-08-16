import { useEffect, useRef, useState, useLayoutEffect } from 'react'

const DEFAULT_POLLING_INTERVAL_IN_MS = 30_000

interface UsePollingOptions {
  // In milliseconds, default is 30s
  pollingInterval?: number
  // Start polling based on a condition Ex: poll only on first page
  startCondition?: boolean
  // Stop polling based on a condition Ex: stop polling when data is fully loaded
  stopCondition?: boolean
}

/**
 *
 * @param callback a promise returning function that will be called in every pollingInterval, ex: refetch
 * @param options: UsePollingOptions
 *
 * remembers last call and re-poll only after its resolved
 * @returns boolean
 */
export function usePolling(
  callback: () => Promise<void> | undefined,
  { startCondition = false, stopCondition = false, pollingInterval = DEFAULT_POLLING_INTERVAL_IN_MS }: UsePollingOptions
): boolean {
  const savedCallback = useRef(callback)
  const [isPolling, setIsPolling] = useState(false)
  const interval = pollingInterval

  // Remember the latest callback if it changes.
  useLayoutEffect(() => {
    savedCallback.current = callback
  }, [callback])

  useEffect(() => {
    // Poll only if start condition from component is met
    if (!startCondition || stopCondition) return

    // Poll only when the current request is resolved
    if (!isPolling) {
      const timerId = setTimeout(async () => {
        setIsPolling(true)
        try {
          await savedCallback.current?.()
        } finally {
          setIsPolling(false)
        }
      }, interval)

      return () => clearTimeout(timerId)
    }
  }, [interval, isPolling, startCondition, stopCondition])

  return isPolling
}
