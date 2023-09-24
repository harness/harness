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

import { Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import React from 'react'

export const PipeSeparator: React.FC<{ height?: number }> = ({ height }) => (
  <Text inline style={{ fontSize: height ? `${height}px` : undefined, alignSelf: 'center' }} color={Color.GREY_200}>
    |
  </Text>
)
