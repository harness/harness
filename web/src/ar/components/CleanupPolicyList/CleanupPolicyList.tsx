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
import type { FormikProps } from 'formik'
import { FontVariation } from '@harnessio/design-system'
import { Button, Card, Layout } from '@harnessio/uicore'

import CleanupPolicy from './CleanupPolicy'
import css from './CleanupPolicyList.module.scss'

// TODO: update type once BE support cleanup policy
interface CleanupPolicyListProps<T> {
  name: string
  formikProps: FormikProps<T>
  onAdd: (val: string) => void
  onRemove: (idx: number) => void
  addButtonLabel: string
  disabled: boolean
  getDefaultValue: () => any
}
export default function CleanupPolicyList<T>(props: CleanupPolicyListProps<T>): JSX.Element {
  const { name, formikProps, disabled, onAdd, addButtonLabel, getDefaultValue, onRemove } = props
  const values = get(formikProps.values, name)

  return (
    <Layout.Vertical spacing="small">
      {/* TODO: update type once BE support cleanup policy */}
      {values?.map((each: any, index: number) => (
        <CleanupPolicy key={each.id} disabled={disabled} name={`${name}[${index}]`} onRemove={() => onRemove(index)} />
      ))}
      <Card className={css.cardContainer}>
        <Button
          className={css.addBtn}
          font={{ variation: FontVariation.FORM_LABEL }}
          icon="plus"
          iconProps={{ size: 12 }}
          minimal
          intent="primary"
          data-testid="add-patter"
          onClick={() => onAdd(getDefaultValue())}
          text={addButtonLabel}
          disabled={disabled}
        />
      </Card>
    </Layout.Vertical>
  )
}
