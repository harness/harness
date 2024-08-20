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
import { Intent } from '@blueprintjs/core'
import * as yup from 'yup'
import { useGet } from 'restful-react'
import { FontVariation } from '@harnessio/design-system'

import { Color } from '@harnessio/design-system'
import {
  Button,
  Container,
  Layout,
  FlexExpander,
  Formik,
  FormikForm,
  FormInput,
  Text,
  ButtonSize,
  ButtonVariation,
  stringSubstitute
} from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import type { RepoRepositoryOutput } from 'services/code'

import { useStrings } from 'framework/strings'
import type { ExportFormDataExtended } from 'utils/GitUtils'
import Upgrade from '../../../icons/Upgrade.svg?url'
import css from '../GeneralSettings/GeneralSpaceSettings.module.scss'

interface ExportFormProps {
  handleSubmit: (data: ExportFormDataExtended) => void
  loading: boolean
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  hideModal: any
  step: number
  setStep: React.Dispatch<React.SetStateAction<number>>
  space: string
}

const ExportForm = (props: ExportFormProps) => {
  const { handleSubmit, loading, hideModal, step, setStep, space } = props
  const { getString } = useStrings()
  // const [auth, setAuth] = useState(false)
  const formInitialValues: ExportFormDataExtended = {
    accountId: '',
    token: '',
    organization: '',
    name: '',
    repoCount: 0
  }

  const validationSchemaStepOne = yup.object().shape({
    accountId: yup.string().trim().required(getString('exportSpace.accIdRequired')),
    token: yup.string().trim().required(getString('exportSpace.accesstokenReq'))
  })

  const validationSchemaStepTwo = yup.object().shape({
    organization: yup.string().trim().required(getString('importSpace.orgRequired')),
    name: yup.string().trim().required(getString('importSpace.spaceNameRequired'))
  })
  const { data: repositories } = useGet<RepoRepositoryOutput[]>({
    path: `/api/v1/spaces/${space}/+/repos`
  })

  return (
    <Formik
      enableReinitialize={true}
      validateOnBlur
      initialValues={formInitialValues}
      formName="exportForm"
      onSubmit={handleSubmit}>
      {formik => {
        const handleValidationClick = async () => {
          try {
            if (step === 0) {
              await validationSchemaStepOne.validate(formik.values, { abortEarly: false })
              setStep(1)
            } else if (step === 1) {
              await validationSchemaStepTwo.validate(formik.values, { abortEarly: false })
              setStep(2)
            } // eslint-disable-next-line @typescript-eslint/no-explicit-any
          } catch (err: any) {
            formik.setErrors(
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              err.inner.reduce((acc: { [x: string]: any }, current: { path: string | number; message: string }) => {
                acc[current.path] = current.message
                return acc
              }, {})
            )
          }
        }
        return (
          <Container width={'97%'}>
            <FormikForm className={css.mainContainer}>
              {step === 0 ? (
                <>
                  <Container className={css.textContainer} width={'90%'}>
                    <FormInput.Text
                      name="accountId"
                      label={getString('exportSpace.accIdLabel')}
                      placeholder={getString('exportSpace.accIdPlaceholder')}
                      tooltipProps={{
                        dataTooltipId: 'accIdTextField'
                      }}
                    />
                    {formik.errors.accountId ? (
                      <Text
                        margin={{ top: 'small', bottom: 'small' }}
                        color={Color.RED_500}
                        icon="circle-cross"
                        iconProps={{ color: Color.RED_500 }}>
                        {formik.errors.accountId}
                      </Text>
                    ) : null}
                    <FormInput.Text
                      name="token"
                      label={
                        <>
                          {getString('exportSpace.tokenLabel')}
                          <a
                            target="_blank"
                            rel="noreferrer"
                            href="https://developer.harness.io/docs/platform/automation/api/add-and-manage-api-keys/">
                            <Icon padding={{ left: 'small' }} className={css.icon} name="code-info" size={16} />
                          </a>
                        </>
                      }
                      placeholder={getString('exportSpace.tokenPlaceholder')}
                      tooltipProps={{
                        dataTooltipId: 'tokenTextField'
                      }}
                      inputGroup={{ type: 'password' }}
                    />
                    {formik.errors.token ? (
                      <Text
                        margin={{ top: 'small', bottom: 'small' }}
                        color={Color.RED_500}
                        icon="circle-cross"
                        iconProps={{ color: Color.RED_500 }}>
                        {formik.errors.token}
                      </Text>
                    ) : null}
                  </Container>
                </>
              ) : null}
              {step === 1 ? (
                <>
                  <Container className={css.textContainer} width={'90%'}>
                    <FormInput.Text
                      name="organization"
                      label={getString('exportSpace.organization')}
                      placeholder={getString('exportSpace.orgIdPlaceholder')}
                      tooltipProps={{
                        dataTooltipId: 'importSpaceOrgName'
                      }}
                    />
                    {formik.errors.organization ? (
                      <Text
                        margin={{ bottom: 'small' }}
                        color={Color.RED_500}
                        icon="circle-cross"
                        iconProps={{ color: Color.RED_500 }}>
                        {formik.errors.organization}
                      </Text>
                    ) : null}
                    <Container>
                      <FormInput.Text
                        name="name"
                        label={getString('exportSpace.projectName')}
                        placeholder={getString('exportSpace.projectIdPlaceholder')}
                        tooltipProps={{
                          dataTooltipId: 'exportProjectName'
                        }}
                      />
                      {formik.errors.name ? (
                        <Text
                          margin={{ bottom: 'small' }}
                          color={Color.RED_500}
                          icon="circle-cross"
                          iconProps={{ color: Color.RED_500 }}>
                          {formik.errors.name}
                        </Text>
                      ) : null}
                    </Container>
                    {/* <Layout.Horizontal>
                      <Label>{getString('exportSpace.entitiesLabel')}</Label>
                      <a href="">
                        <Icon padding={{ left: 'small' }} className={css.icon} name="code-info" size={16} />
                      </a>
                    </Layout.Horizontal> */}

                    {/* <Container className={css.importContainer} padding={'medium'}>
                      <Layout.Horizontal>
                        <FormInput.CheckBox
                          name="repositories"
                          label={getString('pageTitle.repositories')}
                          tooltipProps={{
                            dataTooltipId: 'authorization'
                          }}
                          defaultChecked
                          onClick={() => {
                            setAuth(!auth)
                          }}
                          disabled
                          padding={{ right: 'small' }}
                          className={css.checkbox}
                        />
                        <Container padding={{ left: 'xxxlarge' }}>
                          <FormInput.CheckBox
                            name="pipelines"
                            label={getString('pageTitle.pipelines')}
                            tooltipProps={{
                              dataTooltipId: 'pipelines'
                            }}
                            onClick={() => {
                              setAuth(!auth)
                            }}
                          />
                        </Container>
                      </Layout.Horizontal>
                    </Container> */}
                  </Container>
                </>
              ) : null}

              {step === 2 && (
                <>
                  <Container className={css.textContainer} width={'90%'}>
                    <FormInput.Text
                      name="organization"
                      label={getString('exportSpace.projectOrg')}
                      tooltipProps={{
                        dataTooltipId: 'accIdTextField'
                      }}
                    />
                    {formik.errors.organization ? (
                      <Text
                        margin={{ top: 'small', bottom: 'small' }}
                        color={Color.RED_500}
                        icon="circle-cross"
                        iconProps={{ color: Color.RED_500 }}>
                        {formik.errors.organization}
                      </Text>
                    ) : null}
                    <FormInput.Text
                      name="name"
                      label={getString('exportSpace.projectName')}
                      tooltipProps={{
                        dataTooltipId: 'tokenTextField'
                      }}
                    />
                    {formik.errors.name ? (
                      <Text
                        margin={{ top: 'small', bottom: 'small' }}
                        color={Color.RED_500}
                        icon="circle-cross"
                        iconProps={{ color: Color.RED_500 }}>
                        {formik.errors.name}
                      </Text>
                    ) : null}
                    <Container padding={'small'} className={css.repoInfo}>
                      <Text
                        font={{
                          variation: FontVariation.BODY2
                        }}>
                        {
                          stringSubstitute(getString('exportSpace.repoToConvert'), {
                            length: repositories?.length
                          }) as string
                        }
                      </Text>
                    </Container>
                  </Container>
                </>
              )}

              <hr className={css.dividerContainer} />

              <Layout.Horizontal
                spacing="small"
                padding={{ right: 'xxlarge', bottom: 'large' }}
                style={{ alignItems: 'center' }}>
                {step === 0 && (
                  <Button
                    text={getString('importSpace.next')}
                    intent={Intent.PRIMARY}
                    onClick={() => {
                      handleValidationClick()
                      if (formik.values.accountId !== '' || formik.values.token !== '') {
                        setStep(1)
                      }
                      formik.setErrors({})
                    }}
                  />
                )}
                {step === 1 && (
                  <Button
                    text={getString('importSpace.next')}
                    intent={Intent.PRIMARY}
                    onClick={() => {
                      handleValidationClick()
                      formik.setFieldValue('repoCount', repositories?.length)

                      if (formik.values.organization !== '' || formik.values.name !== '') {
                        formik.setErrors({})
                        setStep(2)
                      }
                    }}
                  />
                )}
                {step === 2 && (
                  <Button
                    size={ButtonSize.MEDIUM}
                    className={css.upgradeButton}
                    variation={ButtonVariation.PRIMARY}
                    text={
                      <Layout.Horizontal>
                        <img width={16} height={16} src={Upgrade} />

                        <Text className={css.buttonText} color={Color.GREY_0}>
                          {getString('exportSpace.startUpgrade')}
                        </Text>
                      </Layout.Horizontal>
                    }
                    intent={Intent.PRIMARY}
                    onClick={() => {
                      handleValidationClick()
                      if (formik.values.accountId !== '' || formik.values.token !== '') {
                        formik.setErrors({})
                        handleSubmit(formik.values)
                      }
                    }}
                  />
                )}

                <Button text={getString('cancel')} minimal onClick={hideModal} />
                <FlexExpander />

                {loading && <Icon intent={Intent.PRIMARY} name="steps-spinner" size={16} />}
              </Layout.Horizontal>
            </FormikForm>
          </Container>
        )
      }}
    </Formik>
  )
}

export default ExportForm
