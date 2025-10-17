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
import { get } from 'lodash-es'
import classNames from 'classnames'
import type { FormikProps } from 'formik'
import { FontVariation } from '@harnessio/design-system'
import { Button, FormInput, Layout, Text } from '@harnessio/uicore'

import css from './AddPatternList.module.scss'

interface AddPatternListProps<T> {
  name: string
  formikProps: FormikProps<T>
  onAdd: (val: string) => void
  onRemove: (idx: number) => void
  label: string
  placeholder: string
  addButtonLabel: string
  disabled: boolean
  className?: string
}

export default function AddPatternList<T>({
  name,
  formikProps,
  onRemove,
  onAdd,
  label,
  placeholder,
  addButtonLabel,
  disabled,
  className
}: AddPatternListProps<T>): JSX.Element {
  const value = get(formikProps.values, name)
  return (
    <Layout.Vertical className={classNames(css.patternsContainer, className)}>
      <Text font={{ variation: FontVariation.FORM_LABEL }}>{label}</Text>
      <Layout.Vertical spacing="none" className={css.patternsListContainer}>
        {value?.map((_each: string, index: number) => (
          <Layout.Horizontal key={`${name}[${index}]`} className={css.patternContainer} spacing="small">
            <FormInput.Text
              className={css.patternInput}
              name={`${name}[${index}]`}
              placeholder={placeholder}
              disabled={disabled}
            />
            <Button
              disabled={disabled}
              minimal
              icon="main-trash"
              data-testid={`remove-pattern-${index}`}
              onClick={() => onRemove(index)}
            />
          </Layout.Horizontal>
        ))}
      </Layout.Vertical>
      <Button
        className={css.addPatternBtn}
        icon="plus"
        iconProps={{ size: 12 }}
        minimal
        intent="primary"
        data-testid="add-patter"
        font={{ variation: FontVariation.FORM_LABEL }}
        onClick={() => onAdd('')}
        text={addButtonLabel}
        disabled={disabled}
      />
    </Layout.Vertical>
  )
}
