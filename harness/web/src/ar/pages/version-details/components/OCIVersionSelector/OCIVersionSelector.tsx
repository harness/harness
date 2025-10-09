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
import { Color } from '@harnessio/design-system'
import { PopoverInteractionKind, Position } from '@blueprintjs/core'
import { Button, ButtonVariation, Layout, Popover, Text } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import { getShortDigest } from '@ar/pages/digest-list/utils'

import type { OCIVersionValue } from './type'
import TagOrDigestListSelector from './TagOrDigestListSelector'

import css from './OCIVersionSelector.module.scss'

interface OCIVersionSelectorProps {
  value: OCIVersionValue
  onChange: (val: OCIVersionValue) => void
}

function OCIVersionSelector(props: OCIVersionSelectorProps) {
  const { value, onChange } = props
  const { getString } = useStrings()
  const label = value.tag ? value.tag : getShortDigest(value.manifest)
  const type = value.tag
    ? getString('versionDetails.OCIVersionSelectorTab.tag')
    : getString('versionDetails.OCIVersionSelectorTab.digest')
  return (
    <Button
      className={css.versionSelector}
      rightIcon="main-chevron-down"
      iconProps={{ size: 12, color: Color.GREY_400 }}
      withoutCurrentColor
      text={
        <Layout.Horizontal className={css.versionSelectorText} spacing="xsmall">
          <Text>{type}:</Text>
          <Text lineClamp={1}>{label}</Text>
        </Layout.Horizontal>
      }
      variation={ButtonVariation.TERTIARY}
      tooltip={
        <Popover>
          <TagOrDigestListSelector value={value} onChange={onChange} />
        </Popover>
      }
      tooltipProps={{
        position: Position.BOTTOM_LEFT,
        interactionKind: PopoverInteractionKind.CLICK,
        minimal: true
      }}
    />
  )
}

export default OCIVersionSelector
