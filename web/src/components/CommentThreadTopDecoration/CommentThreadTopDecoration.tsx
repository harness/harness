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

import React from 'react'
import { Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'

export const CommentThreadTopDecoration: React.FC<{ startLine: number; endLine: number }> = ({
  startLine,
  endLine
}) => {
  const { getString } = useStrings()

  return startLine !== endLine ? (
    <Text
      color={Color.GREY_500}
      padding={{ bottom: 'small' }}
      font={{ variation: FontVariation.BODY }}
      data-start-line={startLine}
      data-end-line={endLine}>
      {getString('pr.commentLineNumbers', { start: startLine, end: endLine })}
    </Text>
  ) : null
}
