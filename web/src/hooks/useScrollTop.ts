import { useEffect } from 'react'

/**
 * Scroll window to top.
 */
export function useScrollTop() {
  useEffect(() => {
    window.scroll({ top: 0 })
  }, [])
}
