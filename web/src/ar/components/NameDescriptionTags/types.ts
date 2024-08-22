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

import type { ITagInputProps } from '@blueprintjs/core'

export interface DescriptionProps {
  placeholder?: string
  isOptional?: boolean
  disabled?: boolean
}
export interface DescriptionComponentProps {
  descriptionProps?: DescriptionProps
  hasValue?: boolean
  disabled?: boolean
  dataTooltipId?: string
}

export interface TagsProps {
  className?: string
}

export interface TagsComponentProps {
  tagsProps?: Partial<ITagInputProps>
  hasValue?: boolean
  isOptional?: boolean
  dataTooltipId?: string
  disabled: boolean
  name: string
}
