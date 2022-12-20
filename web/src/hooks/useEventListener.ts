import { useEffect } from 'react'

export function useEventListener<K extends keyof HTMLElementEventMap>(
  type: K,
  listener: (this: HTMLElement, ev: HTMLElementEventMap[K]) => Unknown,
  element: HTMLElement = window as unknown as HTMLElement,
  options?: boolean | AddEventListenerOptions
) {
  useEffect(() => {
    element?.addEventListener(type, listener, options)
    return () => {
      element?.removeEventListener(type, listener)
    }
  }, [element, type, listener, options])
}
