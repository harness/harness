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

import React from 'react'
import { FontVariation } from '@harnessio/design-system'
import { Container, Text } from '@harnessio/uicore'

import { DEFAULT_DATE_TIME_FORMAT } from '@ar/constants'
import { getReadableDateTime } from '@ar/common/dateUtils'
import { useStrings } from '@ar/frameworks/strings'
import css from './PageTitle.module.scss'

interface CreatedAndModifiedAtProps {
  createdAt: number | undefined
  modifiedAt: number | undefined
}

export default function CreatedAndModifiedAt(props: CreatedAndModifiedAtProps): JSX.Element {
  const { getString } = useStrings()
  const { createdAt, modifiedAt } = props
  return (
    <Container className={css.modificationContainer}>
      <Text font={{ variation: FontVariation.SMALL_BOLD }}>{getString('createdAt')}:</Text>
      <Text font={{ variation: FontVariation.SMALL }}>
        {createdAt ? getReadableDateTime(Number(createdAt), DEFAULT_DATE_TIME_FORMAT) : getString('na')}
      </Text>
      <Text font={{ variation: FontVariation.SMALL_BOLD }}>{getString('modifiedAt')}:</Text>
      <Text font={{ variation: FontVariation.SMALL }}>
        {modifiedAt ? getReadableDateTime(Number(modifiedAt), DEFAULT_DATE_TIME_FORMAT) : getString('na')}
      </Text>
    </Container>
  )
}
