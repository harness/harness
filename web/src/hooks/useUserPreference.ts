import { useCallback, useState } from 'react'

export enum UserPreference {
  DIFF_VIEW_STYLE = 'DIFF_VIEW_STYLE',
  DIFF_LINE_BREAKS = 'DIFF_LINE_BREAKS'
}

export function useUserPreference<T = string>(key: UserPreference, defaultValue: T): [T, (val: T) => void] {
  const prefKey = `CODE_MODULE_USER_PREFERENCE_${key}`
  const convert = useCallback(
    val => {
      if (val === undefined || val === null) {
        return val
      }

      if (typeof defaultValue === 'boolean') {
        return val === 'true'
      }

      if (typeof defaultValue === 'number') {
        return Number(val)
      }

      if (Array.isArray(defaultValue) || typeof defaultValue === 'object') {
        try {
          return JSON.parse(val)
        } catch (exception) {
          // eslint-disable-next-line no-console
          console.error('Failed to parse object', val)
        }
      }
      return val
    },
    [defaultValue]
  )
  const [preference, setPreference] = useState<T>(convert(localStorage[prefKey]) || (defaultValue as T))
  const savePreference = useCallback(
    (val: T) => {
      try {
        localStorage[prefKey] = JSON.stringify(val)
      } catch (exception) {
        // eslint-disable-next-line no-console
        console.error('Failed to stringify object', val)
      }
      setPreference(val)
    },
    [prefKey]
  )

  return [preference, savePreference]
}
