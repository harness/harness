/*
 * Copyright 2023 Harness, Inc.
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

export const getConfig = (str: string): string => {
  // 'code/api/v1' -> 'api/v1'       (standalone)
  //               -> 'code/api/v1'  (embedded inside Harness platform)
  if (window.STRIP_CODE_PREFIX) {
    str = str.replace(/^code\//, '')
  }

  return window.apiUrl ? `${window.apiUrl}/${str}` : `${window.harnessNameSpace || ''}/${str}`
}
