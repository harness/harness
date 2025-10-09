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

export interface TreeViewSortingOption {
  value: string
  label: string
  dir?: string
}

export const TreeViewSortingOptions = [
  { value: 'lastModified,DESC', label: 'Newest', key: 'lastModified', dir: 'DESC' },
  { value: 'lastModified,ASC', label: 'Oldest', key: 'lastModified', dir: 'ASC' },
  { value: 'identifier,ASC', label: 'Name (A->Z, 0->9)', key: 'identifier', dir: 'ASC' },
  { value: 'identifier,DESC', label: 'Name (Z->A, 9->0)', key: 'identifier', dir: 'DESC' }
]
