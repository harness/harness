/*
 * Copyright 2021 Harness Inc. All rights reserved.
 * Use of this source code is governed by the PolyForm Shield 1.0.0 license
 * that can be found in the licenses directory at the root of this repository, also available at
 * https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
 */

import React, { ChangeEvent, useEffect, useState } from 'react'
import { Dialog, Intent } from '@blueprintjs/core'
import * as yup from 'yup'
import {
  Button,
  ButtonProps,
  Container,
  Layout,
  FlexExpander,
  Icon,
  Formik,
  FormikForm,
  Heading,
  useToaster,
  FormInput,
  Label,
  DropDown,
  SelectOption,
  Text
} from '@harness/uicore'
import { Color, FontVariation } from '@harness/design-system'
import { useGet, useMutate } from 'restful-react'
import { get } from 'lodash-es'
import { useModalHook } from '@harness/use-modal'
import { useStrings } from 'framework/strings'
import { BRANCH_PER_PAGE, getErrorMessage } from 'utils/Utils'
import { GitIcon, GitInfoProps, isGitBranchNameValid } from 'utils/GitUtils'
import type { RepoBranch } from 'services/scm'
import css from './CreateBranchModalButton.module.scss'

interface FormData {
  name: string
  sourceBranch: string
}

export interface CreateBranchModalButtonProps extends Omit<ButtonProps, 'onClick'>, Pick<GitInfoProps, 'repoMetadata'> {
  onSuccess: (data: RepoBranch) => void
  showSuccessMessage?: boolean
}

export const CreateBranchModalButton: React.FC<CreateBranchModalButtonProps> = ({
  onSuccess,
  repoMetadata,
  showSuccessMessage,
  ...props
}) => {
  const ModalComponent: React.FC = () => {
    const { getString } = useStrings()
    const [sourceBranch, setSourceBranch] = useState(repoMetadata.defaultBranch as string)
    const { showError, showSuccess } = useToaster()
    const { mutate: createBranch, loading } = useMutate<RepoBranch>({
      verb: 'POST',
      path: `/api/v1/repos/${repoMetadata.path}/+/branches`
    })
    const handleSubmit = (formData?: Unknown): void => {
      const name = get(formData, 'name').trim()
      try {
        createBranch({
          name,
          target: sourceBranch
        })
          .then(response => {
            hideModal()
            onSuccess(response)
            if (showSuccessMessage) {
              showSuccess(getString('branchCreated').replace('__branch__', name), 5000)
            }
          })
          .catch(_error => {
            showError(getErrorMessage(_error), 0, getString('failedToCreateBranch'))
          })
      } catch (exception) {
        showError(getErrorMessage(exception), 0, getString('failedToCreateBranch'))
      }
    }

    return (
      <Dialog
        isOpen
        enforceFocus={false}
        onClose={hideModal}
        title={''}
        style={{ width: 700, maxHeight: '95vh', overflow: 'auto' }}>
        <Layout.Vertical padding={{ left: 'xxlarge' }} style={{ height: '100%' }} className={css.main}>
          <Heading className={css.title} font={{ variation: FontVariation.H3 }} margin={{ bottom: 'xlarge' }}>
            <Icon name={GitIcon.CodeBranch} size={22} /> {getString('createABranch')}
          </Heading>
          <Container margin={{ right: 'xxlarge' }}>
            <Formik<FormData>
              initialValues={{
                name: '',
                sourceBranch: ''
              }}
              formName="createGitBranch"
              enableReinitialize={true}
              validationSchema={yup.object().shape({
                name: yup
                  .string()
                  .trim()
                  .required()
                  .test('valid-branch-name', getString('validation.gitBranchNameInvalid'), value => {
                    const val = value || ''
                    return !!val && isGitBranchNameValid(val)
                  })
              })}
              validateOnChange
              validateOnBlur
              onSubmit={handleSubmit}>
              <FormikForm>
                <FormInput.Text
                  name="name"
                  label={getString('branchName')}
                  placeholder={getString('nameYourBranch')}
                  tooltipProps={{
                    dataTooltipId: 'repositoryBranchTextField'
                  }}
                  inputGroup={{ autoFocus: true }}
                />
                <Container margin={{ top: 'medium', bottom: 'medium' }}>
                  <Label className={css.label}>{getString('branchSource')}</Label>
                  <Text className={css.branchSourceDesc}>{getString('branchSourceDesc')}</Text>
                  <Layout.Horizontal spacing="medium" padding={{ top: 'xsmall' }}>
                    <BranchDropdown
                      repoMetadata={repoMetadata}
                      currentBranchName={sourceBranch}
                      onSelect={name => setSourceBranch(name)}
                    />
                    <FlexExpander />
                  </Layout.Horizontal>
                </Container>

                <Layout.Horizontal
                  spacing="small"
                  padding={{ right: 'xxlarge', top: 'xxxlarge', bottom: 'large' }}
                  style={{ alignItems: 'center' }}>
                  <Button type="submit" text={getString('createBranch')} intent={Intent.PRIMARY} disabled={loading} />
                  <Button text={getString('cancel')} minimal onClick={hideModal} />
                  <FlexExpander />

                  {loading && <Icon intent={Intent.PRIMARY} name="spinner" size={16} />}
                </Layout.Horizontal>
              </FormikForm>
            </Formik>
          </Container>
        </Layout.Vertical>
      </Dialog>
    )
  }

  const [openModal, hideModal] = useModalHook(ModalComponent, [onSuccess])

  return <Button onClick={openModal} {...props} />
}

interface BranchDropdownProps extends Pick<GitInfoProps, 'repoMetadata'> {
  currentBranchName: string
  onSelect: (branchName: string) => void
}

const BranchDropdown: React.FC<BranchDropdownProps> = ({ repoMetadata, onSelect }) => {
  const { getString } = useStrings()
  const [activeBranch, setActiveBranch] = useState(repoMetadata.defaultBranch)
  const [query, setQuery] = useState('')
  const [branches, setBranches] = useState<SelectOption[]>([])
  const { data, loading } = useGet<RepoBranch[]>({
    path: `/api/v1/repos/${repoMetadata.path}/+/branches`,
    queryParams: { sort: 'date', direction: 'desc', per_page: BRANCH_PER_PAGE, page: 1, query }
  })

  useEffect(() => {
    if (data?.length) {
      setBranches(
        data
          .map(e => e.name)
          .map(_branch => ({
            label: _branch,
            value: _branch
          })) as SelectOption[]
      )
    }
  }, [data])

  return (
    <DropDown
      icon={GitIcon.CodeBranch}
      value={activeBranch}
      items={branches}
      {...{
        inputProps: {
          leftElement: <Icon name={loading ? 'steps-spinner' : 'thinner-search'} size={12} color={Color.GREY_500} />,
          placeholder: getString('searchBranch'),
          onInput: (event: ChangeEvent<HTMLInputElement>) => {
            if (event.target.value !== query) {
              setQuery(event.target.value)
            }
          },
          onBlur: (event: ChangeEvent<HTMLInputElement>) => {
            setTimeout(() => {
              setQuery(event.target.value || '')
            }, 250)
          }
        }
      }}
      onChange={({ value: switchBranch }) => {
        setActiveBranch(switchBranch as string)
        onSelect(switchBranch as string)
      }}
      popoverClassName={css.branchDropdown}
      usePortal
    />
  )
}
