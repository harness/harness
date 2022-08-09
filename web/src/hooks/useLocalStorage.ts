import { useState } from 'react'
import type { Dispatch, SetStateAction } from 'react'

// eslint-disable-next-line @typescript-eslint/ban-types
export function isFunction(arg: unknown): arg is Function {
  return typeof arg === 'function'
}

export function useLocalStorage<T>(key: string, initalValue: T): [T, Dispatch<SetStateAction<T>>] {
  const [state, setState] = useState(() => {
    try {
      const item = window.localStorage.getItem(key)

      return item && item !== 'undefined' ? JSON.parse(item) : initalValue
    } catch (e) {
      // eslint-disable-next-line no-console
      console.log(e)
      return initalValue
    }
  })

  function setItem(value: SetStateAction<T>): void {
    try {
      const valueToSet = isFunction(value) ? value(state) : value

      setState(valueToSet)
      window.localStorage.setItem(key, JSON.stringify(valueToSet))
    } catch (e) {
      // eslint-disable-next-line no-console
      console.log(e)
    }
  }

  return [state, setItem]
}
