import React from 'react'

import type { StringsMap } from './stringTypes'

export type StringKeys = keyof StringsMap

export type { StringsMap }

export interface StringsContextValue {
  data: StringsMap
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  getString?(key: StringKeys, vars?: Record<string, any>): string
}

export const StringsContext = React.createContext<StringsContextValue>({} as StringsContextValue)

export function useStringsContext(): StringsContextValue {
  return React.useContext(StringsContext)
}
