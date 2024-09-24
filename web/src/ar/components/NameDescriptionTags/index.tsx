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

import React, { useEffect, useState } from 'react'
import cx from 'classnames'
import { defaultTo } from 'lodash-es'
import { Classes } from '@blueprintjs/core'
import { Icon } from '@harnessio/icons'
import { Container, FormInput, HarnessDocTooltip, Label } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import type { DescriptionComponentProps, TagsComponentProps } from './types'
import css from './NameDescriptionTags.module.scss'

export const Description = (props: DescriptionComponentProps): JSX.Element => {
  const { descriptionProps = {}, hasValue, disabled = false } = props
  const { isOptional = true, ...restDescriptionProps } = descriptionProps
  const { getString } = useStrings()
  const [isDescriptionOpen, setDescriptionOpen] = useState<boolean>(hasValue || false)
  const [isDescriptionFocus, setDescriptionFocus] = useState<boolean>(false)

  useEffect(() => {
    setDescriptionOpen(defaultTo(hasValue, false))
  }, [hasValue])

  return (
    <Container style={{ marginBottom: isDescriptionOpen ? '0' : 'var(--spacing-medium)' }}>
      <Label className={cx(Classes.LABEL, css.descriptionLabel)} data-tooltip-id={props.dataTooltipId}>
        {isOptional ? getString('optionalField', { name: getString('description') }) : getString('description')}
        {props.dataTooltipId ? <HarnessDocTooltip useStandAlone={true} tooltipId={props.dataTooltipId} /> : null}
        {!isDescriptionOpen && (
          <Icon
            className={css.editOpen}
            data-name="edit"
            data-testid="description-edit"
            size={12}
            name="Edit"
            onClick={() => {
              setDescriptionOpen(true)
              setDescriptionFocus(true)
            }}
          />
        )}
      </Label>
      {isDescriptionOpen && (
        <FormInput.TextArea
          data-name="description"
          disabled={disabled}
          autoFocus={isDescriptionFocus}
          name="description"
          placeholder={getString('descriptionPlaceholder')}
          {...restDescriptionProps}
        />
      )}
    </Container>
  )
}

export const Tags = (props: TagsComponentProps): JSX.Element => {
  const { tagsProps = {}, hasValue, isOptional = true, disabled, name } = props
  const { getString } = useStrings()
  const [isTagsOpen, setTagsOpen] = useState<boolean>(hasValue || false)

  useEffect(() => {
    setTagsOpen(defaultTo(hasValue, false))
  }, [hasValue])

  return (
    <Container>
      <Label className={cx(Classes.LABEL, css.descriptionLabel)} data-tooltip-id={props.dataTooltipId}>
        {isOptional ? getString('optionalField', { name: getString('tagsLabel') }) : getString('tagsLabel')}
        {props.dataTooltipId ? <HarnessDocTooltip useStandAlone={true} tooltipId={props.dataTooltipId} /> : null}
        {!isTagsOpen && (
          <Icon
            className={css.editOpen}
            data-name="edit"
            data-testid="tags-edit"
            size={12}
            name="Edit"
            onClick={() => {
              setTagsOpen(true)
            }}
          />
        )}
      </Label>
      {isTagsOpen && (
        <FormInput.KVTagInput
          name={name}
          isArray
          tagsProps={{
            ...tagsProps,
            disabled
          }}
          disabled={disabled}
        />
      )}
    </Container>
  )
}
