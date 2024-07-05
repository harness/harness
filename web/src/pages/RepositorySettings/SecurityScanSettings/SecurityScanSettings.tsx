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
import {
  Container,
  Layout,
  Text,
  Button,
  ButtonVariation,
  Formik,
  useToaster,
  FormInput,
  FormikForm
} from '@harnessio/uicore'
import cx from 'classnames'
import { useGet, useMutate } from 'restful-react'
import { Render } from 'react-jsx-match'
import type { FormikState } from 'formik'
import { Color, FontVariation } from '@harnessio/design-system'
import type { RepoRepositoryOutput } from 'services/code'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { VulnerabilityScanningType } from 'utils/GitUtils'
import { getErrorMessage, permissionProps } from 'utils/Utils'
import { NavigationCheck } from 'components/NavigationCheck/NavigationCheck'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import css from './SecurityScanSettings.module.scss'

interface SecurityScanProps {
  repoMetadata: RepoRepositoryOutput | undefined
  activeTab: string
}

interface FormData {
  secretScanEnable: boolean
  vulnerabilityScanEnable: boolean
  vulnerabilityScanningType: VulnerabilityScanningType
}

const SecurityScanSettings = (props: SecurityScanProps) => {
  const { repoMetadata, activeTab } = props
  const { hooks, standalone, routingId } = useAppContext()
  const { CODE_SECURITY_SCANNING_ON_PUSH } = hooks?.useFeatureFlags()
  const { getString } = useStrings()
  const { showError, showSuccess } = useToaster()
  const space = useGetSpaceParam()
  const permPushResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY',
        resourceIdentifier: repoMetadata?.identifier as string
      },
      permissions: ['code_repo_edit']
    },
    [space]
  )
  const { data: securitySettings, loading: securitySettingsLoading } = useGet({
    path: `/api/v1/repos/${repoMetadata?.path}/+/settings/security`,
    queryParams: { routingId: routingId },
    lazy: !activeTab
  })
  const { mutate: updateSecuritySettings, loading: isUpdating } = useMutate({
    verb: 'PATCH',
    path: `/api/v1/repos/${repoMetadata?.path}/+/settings/security`,
    queryParams: { routingId: routingId }
  })

  const handleSubmit = async (
    formData: FormData,
    resetForm: (nextState?: Partial<FormikState<FormData>> | undefined) => void
  ) => {
    try {
      const payload = {
        secret_scanning_enabled: !!formData?.secretScanEnable,
        vulnerability_scanning_mode: formData?.vulnerabilityScanEnable
          ? formData?.vulnerabilityScanningType
          : VulnerabilityScanningType.DISABLED
      }
      const response = await updateSecuritySettings(payload)
      showSuccess(getString('securitySettings.updateSuccess'), 1500)
      resetForm({
        values: {
          secretScanEnable: !!response?.secret_scanning_enabled,
          vulnerabilityScanEnable: !(response?.vulnerability_scanning_mode === VulnerabilityScanningType.DISABLED),
          vulnerabilityScanningType:
            response?.vulnerability_scanning_mode === VulnerabilityScanningType.DISABLED
              ? VulnerabilityScanningType.DETECT
              : response?.vulnerability_scanning_mode
        }
      })
    } catch (exception) {
      showError(getErrorMessage(exception), 1500, getString('securitySettings.failedToUpdate'))
    }
  }
  return (
    <Container className={css.main}>
      <LoadingSpinner visible={securitySettingsLoading || isUpdating} />
      {securitySettings && (
        <Formik<FormData>
          formName="securityScanSettings"
          initialValues={{
            secretScanEnable: !!securitySettings?.secret_scanning_enabled,
            vulnerabilityScanEnable: !(
              securitySettings?.vulnerability_scanning_mode === VulnerabilityScanningType.DISABLED
            ),
            vulnerabilityScanningType:
              securitySettings?.vulnerability_scanning_mode === VulnerabilityScanningType.DISABLED
                ? VulnerabilityScanningType.DETECT
                : securitySettings?.vulnerability_scanning_mode
          }}
          onSubmit={(formData, { resetForm }) => {
            handleSubmit(formData, resetForm)
          }}>
          {formik => {
            return (
              <FormikForm>
                <Layout.Vertical padding={{ top: 'medium' }}>
                  <Container padding="medium" margin="medium" className={css.generalContainer}>
                    <Layout.Horizontal
                      spacing={'medium'}
                      padding={{ left: 'medium' }}
                      flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
                      <FormInput.Toggle
                        {...permissionProps(permPushResult, standalone)}
                        key={'secretScanEnable'}
                        style={{ margin: '0px' }}
                        label=""
                        name="secretScanEnable"></FormInput.Toggle>
                      <Text className={css.title}>{getString('securitySettings.scanningSecret')}</Text>
                      <Text className={css.text}>{getString('securitySettings.scanningSecretDesc')}</Text>
                    </Layout.Horizontal>
                  </Container>
                  <Render when={!standalone && CODE_SECURITY_SCANNING_ON_PUSH}>
                    <Container padding="medium" margin="medium" className={css.generalContainer}>
                      <Layout.Horizontal
                        spacing={'medium'}
                        padding={{ left: 'medium' }}
                        flex={{ justifyContent: 'flex-start' }}
                        className={cx(formik.values.vulnerabilityScanEnable ? css.expand : css.collapse)}>
                        <FormInput.Toggle
                          {...permissionProps(permPushResult, standalone)}
                          key={'vulnerabilityScanEnable'}
                          label=""
                          style={{ margin: '0px' }}
                          name="vulnerabilityScanEnable"></FormInput.Toggle>
                        <Layout.Vertical padding={{ left: 'medium' }}>
                          <Container className={cx(formik.values.vulnerabilityScanEnable && css.toggle)}>
                            <Layout.Horizontal spacing={'medium'} flex={{ alignItems: 'center' }}>
                              <Text className={css.title}>{getString('securitySettings.vulnerabilityScanning')}</Text>
                              <Text className={css.text}>
                                {getString('securitySettings.vulnerabilityScanningDesc')}
                              </Text>
                            </Layout.Horizontal>
                          </Container>

                          {formik.values.vulnerabilityScanEnable && (
                            <Container margin={{ top: 'medium' }}>
                              <FormInput.RadioGroup
                                {...permissionProps(permPushResult, standalone)}
                                name="vulnerabilityScanningType"
                                key={formik.values.vulnerabilityScanningType}
                                label=""
                                className={css.radioContainer}
                                items={[
                                  {
                                    label: (
                                      <Container>
                                        <Layout.Horizontal spacing={'small'}>
                                          <Text font={{ variation: FontVariation.SMALL_SEMI }}>
                                            {getString('securitySettings.detect')}
                                          </Text>
                                          <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_400}>
                                            {getString('securitySettings.detectDesc')}
                                          </Text>
                                        </Layout.Horizontal>
                                      </Container>
                                    ),
                                    value: VulnerabilityScanningType.DETECT
                                  },
                                  {
                                    label: (
                                      <Container>
                                        <Layout.Horizontal spacing={'small'}>
                                          <Text font={{ variation: FontVariation.SMALL_SEMI }}>
                                            {getString('securitySettings.block')}
                                          </Text>
                                          <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_400}>
                                            {getString('securitySettings.blockDesc')}
                                          </Text>
                                        </Layout.Horizontal>
                                      </Container>
                                    ),
                                    value: VulnerabilityScanningType.BLOCK
                                  }
                                ]}
                              />
                            </Container>
                          )}
                        </Layout.Vertical>
                      </Layout.Horizontal>
                    </Container>
                  </Render>
                </Layout.Vertical>
                <Layout.Horizontal margin={'medium'} spacing={'medium'}>
                  <Button
                    variation={ButtonVariation.PRIMARY}
                    text={getString('save')}
                    onClick={() => formik.submitForm()}
                    disabled={formik.isSubmitting}
                    {...permissionProps(permPushResult, standalone)}
                  />
                </Layout.Horizontal>
                <NavigationCheck when={formik.dirty} />
              </FormikForm>
            )
          }}
        </Formik>
      )}
    </Container>
  )
}

export default SecurityScanSettings
