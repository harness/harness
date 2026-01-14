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

import cdeStringRecords from 'cde-gitness/strings/strings.en.yaml'
import stringsRecordsEN from '../../i18n/strings.en.yaml'
import stringsRecordsES from '../../i18n/strings.es.yaml'

export type LangLocale = 'es' | 'en' | 'en-IN' | 'en-US' | 'en-UK'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type LanguageRecord = Record<string, Record<string, any>>

export function languageLoader(langId: LangLocale = 'en'): LanguageRecord {
  switch (langId) {
    case 'es':
      return stringsRecordsES
    case 'en':
    case 'en-US':
    case 'en-IN':
    case 'en-UK':
    default:
      return { stringsRecordsEN, cdeStringRecords }
  }
}
