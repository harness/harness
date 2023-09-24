/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useState } from 'react'
import { Button, ButtonSize, ButtonVariation, Container, Layout, Text, TextInput } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import css from './UserProfile.module.scss'

enum ACCESS_MODES {
  VIEW,
  EDIT
}

const EditableTextField = ({ onSave, value }: { value: string; onSave: (text: string) => void }) => {
  const { getString } = useStrings()
  const [viewMode, setViewMode] = useState(ACCESS_MODES.VIEW)
  const [text, setText] = useState(value)

  return (
    <Container className={css.editableTextWrapper}>
      {viewMode === ACCESS_MODES.EDIT ? (
        <Layout.Horizontal spacing="medium" width="100%" style={{ alignItems: 'center' }}>
          <TextInput
            defaultValue={value}
            onChange={e => setText((e.target as HTMLInputElement).value)}
            wrapperClassName={css.textInput}
          />
          <Button
            text={getString('save')}
            variation={ButtonVariation.SECONDARY}
            size={ButtonSize.SMALL}
            onClick={() => {
              onSave(text)
              setViewMode(ACCESS_MODES.VIEW)
            }}
          />
          <Button
            text={getString('cancel')}
            variation={ButtonVariation.TERTIARY}
            size={ButtonSize.SMALL}
            onClick={() => {
              setViewMode(ACCESS_MODES.VIEW)
            }}
          />
        </Layout.Horizontal>
      ) : (
        <Text color={Color.GREY_800} font={{ variation: FontVariation.SMALL_SEMI }}>
          {value}
          <Button
            iconProps={{ size: 12 }}
            text={getString('edit')}
            icon="Edit"
            variation={ButtonVariation.LINK}
            onClick={() => {
              setViewMode(ACCESS_MODES.EDIT)
            }}
          />
        </Text>
      )}
    </Container>
  )
}

export default EditableTextField
