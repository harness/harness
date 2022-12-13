import { useCallback, useState } from 'react'

export enum UserPreference {
  DIFF_VIEW_STYLE = 'DIFF_VIEW_STYLE'
}

export function useUserPreference(key: UserPreference, defaultValue: string) {
  const prefKey = `CODE_MODULE_USER_PREFERENCE_${key}`
  const [preference, setPreference] = useState(localStorage[prefKey] || defaultValue)
  const savePreference = useCallback(
    val => {
      localStorage[prefKey] = val
      setPreference(val)
    },
    [prefKey]
  )

  return [preference, savePreference]
}
