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

import React, { PropsWithChildren } from 'react'
import { Color, FontVariation } from '@harnessio/design-system'
import { Button, ButtonVariation, Text } from '@harnessio/uicore'
import { useViolationDetailsModal } from '@ar/pages/violations-list/hooks/useViolationDetailsModal/useViolationDetailsModal'

import css from './ExemptionDetailsSection.module.scss'

export function Label(props: PropsWithChildren<unknown>) {
  return (
    <Text font={{ variation: FontVariation.BODY }} color={Color.GREY_500}>
      {props.children}
    </Text>
  )
}

export function Value(props: PropsWithChildren<unknown>) {
  return (
    <Text font={{ variation: FontVariation.BODY }} color={Color.GREY_800}>
      {props.children}
    </Text>
  )
}

interface VersionActionBtnProps {
  scanId?: string
}
export function VersionActionBtn(props: PropsWithChildren<VersionActionBtnProps>) {
  const { scanId } = props
  const [showModal] = useViolationDetailsModal({ scanId: scanId || '' })
  if (scanId) {
    return (
      <Button className={css.versionBtn} variation={ButtonVariation.LINK} small onClick={showModal}>
        {props.children}
      </Button>
    )
  }
  return <Value>{props.children}</Value>
}
