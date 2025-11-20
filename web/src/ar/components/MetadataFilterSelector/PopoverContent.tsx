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
import type { FormikProps } from 'formik'
import { Button, ButtonSize, ButtonVariation, Container } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'

import PropertiesForm from '../PropertiesForm/PropertiesForm'
import type { PropertiesFormValues, PropertySpec } from '../PropertiesForm/types'
import css from './MetadataFilterSelector.module.scss'

interface PopoverContentProps {
  value: PropertySpec[]
  onSubmit: (data: PropertySpec[]) => void
  onClose: () => void
}

function PopoverContent(props: PopoverContentProps) {
  const [dirty, setDirty] = React.useState(false)
  const formikRef = React.useRef<FormikProps<unknown> | null>(null)
  const { getString } = useStrings()

  const handleApplyFilter = (data: PropertiesFormValues) => {
    props.onSubmit(data.value.filter(each => !!each.key && !!each.value))
  }

  return (
    <Container padding="medium">
      <PropertiesForm
        minItems={1}
        readonly={false}
        value={{ value: props.value }}
        onChangeDirty={setDirty}
        onSubmit={handleApplyFilter}
        ref={formikRef}
        supportQuery={true}
      />
      <Container className={css.actionBtnContainer}>
        <Button
          text={getString('apply')}
          onClick={() => formikRef.current?.submitForm()}
          disabled={!dirty}
          size={ButtonSize.SMALL}
          variation={ButtonVariation.PRIMARY}
        />
        <Button
          text={getString('cancel')}
          onClick={() => props.onClose()}
          size={ButtonSize.SMALL}
          disabled={!dirty}
          variation={ButtonVariation.SECONDARY}
        />
      </Container>
    </Container>
  )
}

export default PopoverContent
