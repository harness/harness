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

import React from 'react'
import { FormInput, Text } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import { ProtectionRulesType } from 'utils/GitUtils'
import css from '../ProtectionRulesForm.module.scss'

const TagRulesForm = () => {
  const { getString } = useStrings()

  return (
    <>
      <FormInput.CheckBox
        className={css.checkboxLabel}
        label={getString('protectionRules.blockCreation', { ruleType: ProtectionRulesType.TAG })}
        name={'blockCreation'}
      />
      <Text padding={{ left: 'xlarge' }} className={css.checkboxText}>
        {getString('protectionRules.blockCreationText', { refs: 'tags' })}
      </Text>

      <hr className={css.dividerContainer} />
      <FormInput.CheckBox
        className={css.checkboxLabel}
        label={getString('protectionRules.blockDeletion', { ruleType: ProtectionRulesType.TAG })}
        name={'blockDeletion'}
      />
      <Text padding={{ left: 'xlarge' }} className={css.checkboxText}>
        {getString('protectionRules.blockDeletionText', { refs: 'tags' })}
      </Text>

      <hr className={css.dividerContainer} />
      <FormInput.CheckBox
        className={css.checkboxLabel}
        label={getString('protectionRules.blockUpdate', { ruleType: ProtectionRulesType.TAG })}
        name={'blockUpdate'}
      />
      <Text padding={{ left: 'xlarge' }} className={css.checkboxText}>
        {getString('protectionRules.blockUpdateText', { refs: 'tags' })}
      </Text>
    </>
  )
}

export default TagRulesForm
