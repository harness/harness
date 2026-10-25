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
import { useMutate } from 'restful-react'
import * as yup from 'yup'
import {
  Button,
  ButtonVariation,
  Dialog,
  FormInput,
  Formik,
  FormikForm,
  Layout,
  SelectOption,
  Text,
  useToaster
} from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import { getErrorMessage } from 'utils/Utils'
import { ShieldFindingUpdateInput, TypesShieldFinding, getShieldApiPath } from 'utils/ShieldUtils'

interface FormData {
  state: SelectOption
  reason: string
}

interface FindingStateDialogProps {
  repoPath?: string
  finding: TypesShieldFinding
  onClose: () => void
  onUpdated: () => void
}

// Dialog to ignore a finding or accept its risk. A reason is required.
export const FindingStateDialog: React.FC<FindingStateDialogProps> = ({ repoPath, finding, onClose, onUpdated }) => {
  const { getString } = useStrings()
  const { showError, showSuccess } = useToaster()
  const { mutate: updateFinding, loading } = useMutate<TypesShieldFinding>({
    verb: 'PATCH',
    path: `${getShieldApiPath(repoPath)}/findings/${finding.id}`
  })

  const onSubmit = (data: FormData) => {
    const payload: ShieldFindingUpdateInput = {
      state: data.state.value as ShieldFindingUpdateInput['state'],
      reason: data.reason.trim()
    }
    updateFinding(payload)
      .then(() => {
        showSuccess(getString('shield.findingUpdated'))
        onUpdated()
      })
      .catch(error => showError(getErrorMessage(error)))
  }

  return (
    <Dialog isOpen enforceFocus={false} onClose={onClose} title={getString('shield.dismissFinding')}>
      <Formik<FormData>
        initialValues={{ state: { label: getString('shield.stateValue.ignored'), value: 'ignored' }, reason: '' }}
        formName="updateShieldFinding"
        validationSchema={yup.object().shape({
          reason: yup
            .string()
            .trim()
            .required(`${getString('shield.reason')} ${getString('isRequired')}`)
        })}
        validateOnChange
        validateOnBlur
        onSubmit={onSubmit}>
        <FormikForm>
          <Layout.Vertical spacing="small">
            <Text color={Color.GREY_700}>
              {finding.vuln_id}
              {finding.package ? ` - ${finding.package}` : ''}
            </Text>
            <FormInput.Select
              name="state"
              label={getString('shield.state')}
              items={[
                { label: getString('shield.stateValue.ignored'), value: 'ignored' },
                { label: getString('shield.stateValue.accepted_risk'), value: 'accepted_risk' }
              ]}
            />
            <FormInput.TextArea name="reason" label={getString('shield.reason')} />
            <Layout.Horizontal spacing="medium" padding={{ top: 'medium' }}>
              <Button variation={ButtonVariation.PRIMARY} text={getString('save')} type="submit" disabled={loading} />
              <Button variation={ButtonVariation.TERTIARY} text={getString('cancel')} onClick={onClose} />
            </Layout.Horizontal>
          </Layout.Vertical>
        </FormikForm>
      </Formik>
    </Dialog>
  )
}
