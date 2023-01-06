import { useCallback, useState } from 'react'

export enum UserPreference {
  DIFF_VIEW_STYLE = 'DIFF_VIEW_STYLE',
  DIFF_LINE_BREAKS = 'DIFF_LINE_BREAKS'
}

export function useUserPreference<T = string>(key: UserPreference, defaultValue: T): [T, (val: T) => void] {
  const prefKey = `CODE_MOD_USER_PREF__${key}`
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
          console.error('useUserPreference: Failed to parse object', val)
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
        localStorage[prefKey] = Array.isArray(val) || typeof val === 'object' ? JSON.stringify(val) : val
      } catch (exception) {
        // eslint-disable-next-line no-console
        console.error('useUserPreference: Failed to stringify object', val)
      }
      setPreference(val)
    },
    [prefKey]
  )

  return [preference, savePreference]
}
