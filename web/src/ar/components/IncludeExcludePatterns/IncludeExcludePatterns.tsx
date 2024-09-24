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

import React, { useState } from 'react'
import { get } from 'lodash-es'
import type { FormikProps } from 'formik'
import { FontVariation } from '@harnessio/design-system'
import { Button, Container } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'

import PatternInput from '@ar/components/Form/PatternInput/PatternInput'
import css from './IncludeExcludePatterns.module.scss'

interface PatternListProps {
  name: string
  label: string
  placeholder: string
  addButtonLabel: string
}

interface IncludeExcludePatternsProps<T> {
  disabled: boolean
  formikProps: FormikProps<T>
  isEdit: boolean
  includePatternListProps: PatternListProps
  excludePatternListProps: PatternListProps
}

interface FormData {
  allowedPattern: string[] | undefined
  blockedPattern: string[] | undefined
}

function shouldShowIncludeExcludeList(values: FormData): boolean {
  const includePatterns = get(values, 'allowedPattern', []) || []
  const excludePatterns = get(values, 'blockedPattern', []) || []
  return !!includePatterns.length || !!excludePatterns.length
}

export default function IncludeExcludePatterns<T>(props: IncludeExcludePatternsProps<T>): JSX.Element {
  const { formikProps, disabled, includePatternListProps, excludePatternListProps } = props
  const { getString } = useStrings()

  const [showList, setShowList] = useState(shouldShowIncludeExcludeList(formikProps.values as FormData))

  if (!showList) {
    return (
      <Container flex={{ justifyContent: 'flex-start' }}>
        <Button
          className={css.includePatternsBtn}
          minimal
          rightIcon="chevron-down"
          iconProps={{ size: 12 }}
          text={getString('repositoryDetails.repositoryForm.addPatterns')}
          intent="primary"
          data-testid="add-pattern"
          font={{ variation: FontVariation.FORM_LABEL }}
          onClick={() => {
            setShowList(true)
          }}
          disabled={disabled}
        />
      </Container>
    )
  }
  return (
    <Container>
      <PatternInput
        label={includePatternListProps.label}
        name={includePatternListProps.name}
        placeholder={includePatternListProps.placeholder}
        disabled={disabled}
      />
      <PatternInput
        label={excludePatternListProps.label}
        name={excludePatternListProps.name}
        placeholder={excludePatternListProps.placeholder}
        disabled={disabled}
      />
    </Container>
  )
}
