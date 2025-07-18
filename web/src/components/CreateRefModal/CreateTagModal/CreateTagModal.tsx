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

import React, { useCallback, useState } from 'react'
import { Dialog, Intent } from '@blueprintjs/core'
import * as yup from 'yup'
import {
  Button,
  ButtonProps,
  Container,
  Layout,
  FlexExpander,
  Formik,
  FormikForm,
  Heading,
  useToaster,
  Text,
  FormInput,
  Label,
  ButtonVariation,
  StringSubstitute
} from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { useMutate } from 'restful-react'
import { get } from 'lodash-es'
import { Render } from 'react-jsx-match'
import { useModalHook } from 'hooks/useModalHook'
import { useStrings } from 'framework/strings'
import { getErrorMessage, permissionProps } from 'utils/Utils'
import { GitInfoProps, normalizeGitRef, isGitBranchNameValid, GitRefType } from 'utils/GitUtils'
import { BranchTagSelect } from 'components/BranchTagSelect/BranchTagSelect'
import type { TypesBranchExtended } from 'services/code'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useAppContext } from 'AppContext'
import { useRuleViolationCheck } from 'hooks/useRuleViolationCheck'
import css from '../CreateRefModal.module.scss'

interface FormData {
  name: string
  sourceBranch: string
  description: string
}

interface UseCreateTagModalProps extends Pick<GitInfoProps, 'repoMetadata'> {
  suggestedTagName?: string
  suggestedSourceBranch?: string
  onSuccess: (data: TypesBranchExtended) => void
  showSuccessMessage?: boolean
}

interface CreateTagModalButtonProps extends Omit<ButtonProps, 'onClick'>, UseCreateTagModalProps {
  onSuccess: (data: TypesBranchExtended) => void
  showSuccessMessage?: boolean
}

export function useCreateTagModal({
  suggestedTagName = '',
  suggestedSourceBranch = '',
  onSuccess,
  repoMetadata,
  showSuccessMessage
}: CreateTagModalButtonProps) {
  const [tagName, setTagName] = useState(suggestedTagName)
  const ModalComponent: React.FC = () => {
    const { getString } = useStrings()
    const [sourceBranch, setSourceBranch] = useState(suggestedSourceBranch || (repoMetadata.default_branch as string))
    const { showError, showSuccess } = useToaster()
    const { violation, bypassable, bypassed, setAllStates } = useRuleViolationCheck()
    const { mutate: createTag, loading } = useMutate<TypesBranchExtended>({
      verb: 'POST',
      path: `/api/v1/repos/${repoMetadata.path}/+/tags`
    })

    const handleSubmit = (formData: FormData) => {
      const name = get(formData, 'name').trim()
      const description = get(formData, 'description').trim()

      try {
        createTag({
          name,
          message: description,
          target: normalizeGitRef(sourceBranch),
          bypass_rules: bypassed
        })
          .then(response => {
            hideModal()
            onSuccess(response)
            if (showSuccessMessage) {
              showSuccess(
                <StringSubstitute
                  str={getString('tagCreated')}
                  vars={{
                    tag: name
                  }}
                />,
                5000
              )
            }
          })
          .catch(_error => {
            if (_error.status === 422) {
              setAllStates({
                violation: true,
                bypassed: true,
                bypassable: _error?.data?.violations?.[0]?.bypassable
              })
            } else showError(getErrorMessage(_error), 0, 'failedToCreateTag')
          })
      } catch (exception) {
        showError(getErrorMessage(exception), 0, 'failedToCreateTag')
      }
    }

    return (
      <Dialog
        isOpen
        enforceFocus={false}
        onClose={hideModal}
        title={''}
        style={{ width: 585, maxHeight: '95vh', overflow: 'auto' }}>
        <Layout.Vertical padding={{ left: 'xxlarge' }} style={{ height: '100%' }} className={css.main}>
          <Heading className={css.title} font={{ variation: FontVariation.H3 }} margin={{ bottom: 'xlarge' }}>
            {getString('createATag')}
          </Heading>
          <Container className={css.container} margin={{ right: 'xxlarge' }}>
            <Formik<FormData>
              initialValues={{
                name: tagName,
                sourceBranch: suggestedSourceBranch,
                description: ''
              }}
              formName="createGitTag"
              enableReinitialize
              validationSchema={yup.object().shape({
                name: yup
                  .string()
                  .trim()
                  .required()
                  .test('valid-tag-name', getString('validation.gitTagNameInvalid'), value => {
                    const val = value || ''
                    return !!val && isGitBranchNameValid(val)
                  }),
                description: yup.string().required()
              })}
              validateOnChange
              validateOnBlur
              onSubmit={handleSubmit}>
              <FormikForm>
                <FormInput.Text
                  name="name"
                  label={getString('name')}
                  placeholder={getString('enterTagPlaceholder')}
                  tooltipProps={{
                    dataTooltipId: 'repositoryTagTextField'
                  }}
                  inputGroup={{ autoFocus: true }}
                />
                <Container margin={{ top: 'medium', bottom: 'medium' }}>
                  <Label className={css.label}>{getString('basedOn')}</Label>
                  <Layout.Horizontal className={css.selectContainer} padding={{ top: 'xsmall' }}>
                    <BranchTagSelect
                      className={css.branchTagSelect}
                      repoMetadata={repoMetadata}
                      disableBranchCreation
                      disableViewAllBranches
                      gitRef={sourceBranch}
                      onSelect={setSourceBranch}
                      popoverClassname={css.popoverContainer}
                    />
                    <FlexExpander />
                  </Layout.Horizontal>
                </Container>

                <FormInput.TextArea
                  label={getString('description')}
                  className={css.extendedDescription}
                  name="description"
                  placeholder={getString('tagDescription')}
                />
                <Container margin={{ bottom: 'large' }}>
                  <Layout.Horizontal
                    spacing="small"
                    padding={{ right: 'xxlarge', top: 'xxlarge' }}
                    style={{ alignItems: 'center' }}>
                    {!bypassable ? (
                      <Button
                        type="submit"
                        text={getString('createTag')}
                        variation={ButtonVariation.PRIMARY}
                        disabled={loading}
                      />
                    ) : (
                      <Button
                        intent={Intent.DANGER}
                        disabled={loading}
                        type="submit"
                        variation={ButtonVariation.SECONDARY}
                        text={getString('protectionRules.createRefAlertBtn', { ref: GitRefType.TAG })}
                      />
                    )}
                    <Button text={getString('cancel')} variation={ButtonVariation.LINK} onClick={hideModal} />
                    <FlexExpander />
                    {loading && <Icon intent={Intent.PRIMARY} name="steps-spinner" size={16} />}
                  </Layout.Horizontal>
                  <Render when={violation}>
                    <Layout.Horizontal className={css.warningMessage}>
                      <Icon intent={Intent.WARNING} name="danger-icon" size={16} />
                      <Text font={{ variation: FontVariation.BODY2 }} color={Color.RED_800}>
                        {bypassable
                          ? getString('protectionRules.createRefAlertText', { ref: GitRefType.TAG })
                          : getString('protectionRules.createRefBlockText', { ref: GitRefType.TAG })}
                      </Text>
                    </Layout.Horizontal>
                  </Render>
                </Container>
              </FormikForm>
            </Formik>
          </Container>
        </Layout.Vertical>
      </Dialog>
    )
  }
  const [openModal, hideModal] = useModalHook(ModalComponent, [
    onSuccess,
    suggestedTagName,
    suggestedSourceBranch,
    showSuccessMessage
  ])
  const fn = useCallback(
    (_tagName?: string) => {
      if (_tagName) {
        setTagName(_tagName)
      }
      openModal()
    },
    [setTagName, openModal]
  )

  return fn
}

export const CreateTagModalButton: React.FC<CreateTagModalButtonProps> = ({
  onSuccess,
  repoMetadata,
  showSuccessMessage,
  ...props
}) => {
  const openModal = useCreateTagModal({ repoMetadata, onSuccess, showSuccessMessage })
  const { standalone } = useAppContext()
  const { hooks } = useAppContext()
  const space = useGetSpaceParam()

  const permPushResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY',
        resourceIdentifier: repoMetadata?.identifier as string
      },
      permissions: ['code_repo_push']
    },
    [space]
  )
  return <Button onClick={() => openModal()} {...props} {...permissionProps(permPushResult, standalone)} />
}
