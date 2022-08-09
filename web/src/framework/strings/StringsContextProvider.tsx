import React from 'react'
import { languageLoader } from './languageLoader'

import { StringsContext, StringsContextValue } from './StringsContext'

export interface StringsContextProviderProps extends Pick<StringsContextValue, 'getString'> {
  children: React.ReactNode
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  initialStrings?: Record<string, any> // temp prop for backward compatability
}

export function StringsContextProvider(props: StringsContextProviderProps): React.ReactElement {
  // const [strings, setStrings] = React.useState<StringsMap>(props.initialStrings || {})

  // useGlobalEventListener('LOAD_STRINGS_CHUNK', (e: CustomEvent<HarnessModules[]>) => {
  //   const mods = e.detail

  //   const promises = mods.map(mod => languageLoader('en', mod).then<[HarnessModules, StringsMap]>(data => [mod, data]))

  //   Promise.all(promises).then(data => {
  //     const newData = data.reduce<StringsMap>((acc, [mod, values]) => ({ ...acc, [mod]: values.default }), {})

  //     setStrings(oldData => ({
  //       ...oldData,
  //       ...newData
  //     }))
  //   })
  // })

  return (
    <StringsContext.Provider
      value={{
        data: {
          ...props.initialStrings,
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          ...(languageLoader() as any)
        },
        getString: props.getString
      }}>
      {props.children}
    </StringsContext.Provider>
  )
}
