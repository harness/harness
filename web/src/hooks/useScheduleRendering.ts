import { useCallback, useRef } from 'react'

interface UseSchedueRenderingProps<T> {
  renderer: (items: T[]) => void
  size?: number
  initialData?: T[]
}

export function useScheduleRendering<T>({ renderer, size = 50, initialData = [] }: UseSchedueRenderingProps<T>) {
  const progressRef = useRef(false)
  const rafRef = useRef(0)
  const data = useRef<T[]>(initialData)
  const startRendering = useCallback(() => {
    if (progressRef.current || !data.current.length) {
      return
    }

    cancelAnimationFrame(rafRef.current)
    progressRef.current = true

    rafRef.current = requestAnimationFrame(() => {
      try {
        renderer(data.current.splice(0, size))
        progressRef.current = false

        if (data.current.length) {
          startRendering()
        }
      } catch (error) {
        console.error('An error has occurred', error) // eslint-disable-line no-console
      }
    })
  }, [renderer, size])

  const sendDataToRender = useCallback(
    (item: T) => {
      data.current.push(item)
      startRendering()
    },
    [startRendering]
  )

  return sendDataToRender
}
