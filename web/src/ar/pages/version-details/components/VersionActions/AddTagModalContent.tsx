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
import { Formik } from 'formik'
import { array, object } from 'yup'
import { Button, ButtonVariation, Container, Layout } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import PatternInput from '@ar/components/Form/PatternInput/PatternInput'

import css from './AddTagMenuItem.module.scss'

function normalizeTagNames(tagNames: string[] | (string | { label: string; value: string })[]): string[] {
  if (!Array.isArray(tagNames)) return []
  return tagNames
    .map(item => (typeof item === 'string' ? item : item?.value ?? ''))
    .map(t => t.trim())
    .filter(Boolean)
}

export interface AddTagModalContentProps {
  onSubmit: (tagNames: string[]) => void
  onClose: () => void
  label?: string
  placeholder?: string
  addButtonText?: string
  disabled?: boolean
  /** Text shown above the tag input bar */
  instructions?: string
}

export function AddTagModalContent({
  onSubmit,
  onClose,
  label,
  placeholder,
  addButtonText,
  disabled = false,
  instructions
}: AddTagModalContentProps): JSX.Element {
  const { getString } = useStrings()
  return (
    <Formik
      initialValues={{ tagNames: [] as string[] }}
      onSubmit={values => onSubmit(normalizeTagNames(values.tagNames))}
      validationSchema={object({
        tagNames: array().min(
          1,
          getString('validationMessages.entityRequired', {
            entity: getString('versionList.actions.tagNameEntity')
          })
        )
      })}>
      {formik => (
        <Container>
          <Layout.Vertical spacing="medium">
            <Container className={css.instructions}>{instructions}</Container>
            <Container className={css.addTagInputWrapper}>
              <PatternInput
                name="tagNames"
                label={label ?? getString('versionList.table.columns.tags')}
                placeholder={placeholder ?? getString('versionList.actions.addTagPlaceholder')}
                disabled={disabled}
              />
            </Container>
          </Layout.Vertical>
          <Layout.Horizontal spacing="medium" margin={{ top: 'large' }}>
            <Button
              variation={ButtonVariation.PRIMARY}
              text={addButtonText || getString('add')}
              onClick={() => formik.submitForm()}
              disabled={disabled}
            />
            <Button
              variation={ButtonVariation.SECONDARY}
              text={getString('cancel')}
              onClick={onClose}
              disabled={disabled}
            />
          </Layout.Horizontal>
        </Container>
      )}
    </Formik>
  )
}
