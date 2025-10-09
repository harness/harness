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

import React, { useEffect } from 'react'
import { get } from 'lodash-es'
import classNames from 'classnames'
import { useFormikContext } from 'formik'
import { FormGroup } from '@blueprintjs/core'
import { Container, FormInput, Label, Layout } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import { useAppStore, useParentComponents } from '@ar/hooks'

import css from './MultiTypeSecretInput.module.scss'

export enum SecretValueType {
  TEXT = 'TEXT',
  ENCRYPTED = 'ENCRYPTED'
}

interface MultiTypeSecretInputProps {
  name: string
  label: string
  typeField: string
  secretField: string
  secretSpaceIdField: string
  helperText?: string
  className?: string
  labelClassName?: string
  placeholder?: string
  disabled?: boolean
  onChangeType?: (type: SecretValueType) => void
  onChangeValue?: (e: React.FormEvent<HTMLElement>, type: SecretValueType) => void
}

export default function MultiTypeSecretInput(props: MultiTypeSecretInputProps) {
  const {
    name,
    className,
    labelClassName,
    label,
    typeField,
    onChangeType,
    onChangeValue,
    placeholder,
    disabled,
    secretField,
    secretSpaceIdField
  } = props

  const formik = useFormikContext()
  const valueType = get(formik.values, typeField)

  const { scope } = useAppStore()
  const { getString } = useStrings()
  const { SecretFormInput } = useParentComponents()

  useEffect(() => {
    if (!valueType) {
      formik.setFieldValue(typeField, SecretValueType.TEXT)
    }
  }, [])

  return (
    <FormGroup>
      <Layout.Vertical className={className}>
        <Layout.Horizontal className={classNames(labelClassName, css.labelContainer)}>
          <Label>{label}</Label>
          <FormInput.DropDown
            className={css.dropdownSelect}
            name={typeField}
            items={[
              { label: getString('plaintext'), value: SecretValueType.TEXT },
              { label: getString('encrypted'), value: SecretValueType.ENCRYPTED }
            ]}
            onChange={option => {
              onChangeType?.(option.value as SecretValueType)
              formik.setFieldValue(name, undefined)
              formik.setFieldValue(secretField, undefined)
              formik.setFieldValue(secretSpaceIdField, undefined)
            }}
            dropDownProps={{
              isLabel: true,
              filterable: false,
              minWidth: 'unset'
            }}
          />
        </Layout.Horizontal>
        <Container>
          {valueType === SecretValueType.TEXT && (
            <FormInput.Text
              name={name}
              placeholder={placeholder}
              onChange={e => onChangeValue?.(e, valueType)}
              disabled={disabled}
            />
          )}
          {valueType === SecretValueType.ENCRYPTED && (
            <SecretFormInput
              name={secretField}
              spaceIdFieldName={secretSpaceIdField}
              placeholder={placeholder}
              scope={scope}
              disabled={disabled}
              formik={formik}
            />
          )}
        </Container>
      </Layout.Vertical>
    </FormGroup>
  )
}
