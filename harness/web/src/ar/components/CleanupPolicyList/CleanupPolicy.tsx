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
import { Color, FontVariation } from '@harnessio/design-system'
import { Button, ButtonVariation, Card, FormInput, Layout, Text } from '@harnessio/uicore'

import { String, useStrings } from '@ar/frameworks/strings'
import type { StringsMap } from '@ar/frameworks/strings'

import CheckboxWithPatternInput from '@ar/components/Form/CheckboxWithMultitypeInput/CheckboxWithPatternInput'

import css from './CleanupPolicyList.module.scss'

interface LabelElement {
  label: keyof StringsMap
}

function LabelElement({ label }: LabelElement): JSX.Element {
  return <String useRichText stringID={label} />
}

interface CleanupPolicyProps {
  disabled?: boolean
  name: string
  onRemove: () => void
}

export default function CleanupPolicy(props: CleanupPolicyProps): JSX.Element {
  const { disabled, name, onRemove } = props
  const { getString } = useStrings()
  return (
    <Card className={css.cardContainer}>
      <Layout.Vertical spacing="small">
        <FormInput.Text
          name={`${name}.name`}
          label={getString('cleanupPolicy.name')}
          placeholder={getString('enterPlaceholder', { name: getString('cleanupPolicy.name') })}
          disabled={disabled}
        />
        <Text icon="info-messaging" font={{ variation: FontVariation.BODY }} color={Color.GREY_600}>
          {getString('cleanupPolicy.infoMessage')}
        </Text>
        <CheckboxWithPatternInput
          labelElement={<LabelElement label="cleanupPolicy.cleanUpByVersionPrefixes" />}
          name={`${name}.versionPrefix`}
          placeholder={getString('cleanupPolicy.placeholder')}
          disabled={disabled}
        />
        <CheckboxWithPatternInput
          labelElement={<LabelElement label="cleanupPolicy.cleanUpByPackagePrefixes" />}
          name={`${name}.packagePrefix`}
          placeholder={getString('cleanupPolicy.placeholder')}
          disabled={disabled}
        />
        <FormInput.Text
          name={`${name}.expireDays`}
          label={<LabelElement label="cleanupPolicy.cleanUpByArtifactsOlderThan" />}
          placeholder={getString('cleanupPolicy.cleanUpByArtifactsOlderThanPlaceholder')}
          disabled={disabled}
        />
      </Layout.Vertical>
      <Button
        minimal
        disabled={disabled}
        className={css.absoluteDeleteBtn}
        variation={ButtonVariation.ICON}
        icon="main-trash"
        data-testid={`remove-pattern-${name}`}
        onClick={() => onRemove()}
      />
    </Card>
  )
}
