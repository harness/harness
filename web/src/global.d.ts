/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/* eslint-disable @typescript-eslint/no-explicit-any */
declare const __DEV__: boolean

declare module '*.png' {
  const value: string
  export default value
}

declare module '*.jpg' {
  const value: string
  export default value
}

declare module '*.svg' {
  import React from 'react'
  const SVG: React.VFC<React.SVGProps<SVGSVGElement>>
  export default SVG
}

declare module '*.svg?url' {
  const value: string
  export default value
}

declare module '*.gif' {
  const value: string
  export default value
}

declare module '*.mp4' {
  const value: string
  export default value
}

declare module '*.yaml' {
  const value: Record<string, any>
  export default value
}

declare module '*.yml' {
  const value: Record<string, any>
  export default value
}

declare module '*.gql' {
  const query: string
  export default query
}

declare interface Window {
  apiUrl: string
  harnessNameSpace: string
  bugsnagClient?: any
  STRIP_CDE_PREFIX?: boolean
  STRIP_CODE_PREFIX?: boolean
  Sanitizer: any
  publicAccessOnGitness: boolean
}

declare const __ENABLE_CDN__: boolean
declare let __webpack_public_path__: string

declare module '*.scss'

type Unknown = any

type RequiredField<T, K extends keyof T> = T & Required<Pick<T, K>>

type Optional<Type, Key extends keyof Type> = Omit<Type, Key> & Partial<Pick<Type, Key>>

declare module 'lang-map' {
  const languages: { languages: (name: string) => string[] }
  export default languages
}

declare type Nullable<T> = T | undefined | null

declare module 'monaco-editor/esm/vs/editor/common/services/languageFeatures.js' {
  export const ILanguageFeaturesService: { documentSymbolProvider: unknown }
}

declare module 'monaco-editor/esm/vs/editor/contrib/documentSymbols/browser/outlineModel.js' {
  import type { editor, languages } from 'monaco-editor'

  export abstract class OutlineModel {
    static create(registry: unknown, model: editor.ITextModel): Promise<OutlineModel>

    asListOfDocumentSymbols(): languages.DocumentSymbol[]
  }
}

declare module 'monaco-editor/esm/vs/editor/standalone/browser/standaloneServices.js' {
  export const StandaloneServices: {
    get: (id: unknown) => { documentSymbolProvider: unknown }
  }
}
