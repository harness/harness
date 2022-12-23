import { useCallback, useState } from 'react'

export enum UserPreference {
  DIFF_VIEW_STYLE = 'DIFF_VIEW_STYLE',
  DIFF_LINE_BREAKS = 'DIFF_LINE_BREAKS'
}

export function useUserPreference<T = string>(key: UserPreference, defaultValue: T): [T, (val: T) => void] {
  const prefKey = `CODE_MODULE_USER_PREFERENCE_${key}`
  const [preference, setPreference] = useState<T>(localStorage[prefKey] || (defaultValue as T))
  const savePreference = useCallback(
    (val: T) => {
      localStorage[prefKey] = val
      setPreference(val)
    },
    [prefKey]
  )

  return [preference, savePreference]
}
