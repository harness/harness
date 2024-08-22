/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

export const DEFAULT_PAGE_INDEX = 0
export const DEFAULT_PAGE_SIZE = 50
export const PAGE_SIZE_OPTIONS = [10, 20, 50, 100]
export const DEFAULT_PIPELINE_LIST_TABLE_SORT = ['updatedAt', 'DESC']
export const DEFAULT_UPSTREAM_PROXY_LIST_TABLE_SORT = ['updatedAt', 'DESC']

export const DEFAULT_DATE_FORMAT = 'MMM DD, YYYY'
export const DEFAULT_TIME_FORMAT = 'hh:mm a'
export const DEFAULT_DATE_TIME_FORMAT = `${DEFAULT_DATE_FORMAT}  ${DEFAULT_TIME_FORMAT}`

export const REPO_KEY_REGEX = /^[a-z0-9]+(?:[._-][a-z0-9]+)*$/
export const URL_REGEX =
  /((https?):\/\/)?(www.)?[a-z0-9]+(\.[a-z]{2,}){1,3}(#?\/?[a-zA-Z0-9#]+)*\/?(\?[a-zA-Z0-9-_]+=[a-zA-Z0-9-%]+&?)?$/

export enum PreferenceScope {
  USER = 'USER',
  MACHINE = 'MACHINE' // or workstation. This will act as default PreferenceScope
}
