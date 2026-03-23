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
import { Formik } from 'formik'
import type { FormikProps } from 'formik'
import produce from 'immer'
import { object, string } from 'yup'
import { Color } from '@harnessio/design-system'
import { Button, ButtonVariation, Container, FormInput, FormikForm, Heading, Layout } from '@harnessio/uicore'
import { useAppStore } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import type { RepositoryConfigType } from '@ar/common/types'
import { OrganizationSelect, ProjectSelect, RegistrySelect } from '@ar/components/ScopeSelects'

import css from './CloneVersionModalContent.module.scss'

export interface CloneVersionFormValues {
  sourceRegistry: string
  sourcePackage: string
  sourceVersion: string
  organization: string
  project: string
  registry: string
  registryUuid: string
}

export interface CloneVersionModalContentProps {
  registryName: string
  packageName: string
  version: string
  registryType?: RepositoryConfigType
  packageType?: string
  onSubmit: (target: { organization: string; project: string; registry: string; registryUuid: string }) => void
  onClose: () => void
  disabled?: boolean
}

export function CloneVersionModalContent({
  registryName,
  packageName,
  version,
  registryType,
  packageType,
  onSubmit,
  onClose,
  disabled
}: CloneVersionModalContentProps): JSX.Element {
  const { getString } = useStrings()
  const { scope } = useAppStore()
  const initialOrg = scope?.orgIdentifier ?? ''
  const initialProject = scope?.projectIdentifier ?? ''

  const [selectedOrg, setSelectedOrg] = useState<string>(initialOrg)
  const [selectedProject, setSelectedProject] = useState<string>(initialProject)

  function handleOrgChange(value: string, formik: FormikProps<CloneVersionFormValues>): void {
    setSelectedOrg(value)
    const newValues = produce(formik.values, draft => {
      draft.organization = value
      draft.project = ''
      draft.registry = ''
      draft.registryUuid = ''
    })
    formik.setValues(newValues)
    setSelectedProject('')
  }

  function handleProjectChange(value: string, formik: FormikProps<CloneVersionFormValues>): void {
    setSelectedProject(value)
    const newValues = produce(formik.values, draft => {
      draft.project = value
      draft.registry = ''
      draft.registryUuid = ''
    })
    formik.setValues(newValues)
  }

  return (
    <Formik
      initialValues={{
        sourceRegistry: registryName,
        sourcePackage: packageName,
        sourceVersion: version,
        organization: initialOrg ?? '',
        project: initialProject ?? '',
        registry: '',
        registryUuid: ''
      }}
      onSubmit={values =>
        onSubmit({
          organization: values.organization,
          project: values.project,
          registry: values.registry,
          registryUuid: values.registryUuid
        })
      }
      validationSchema={object({
        organization: string(),
        project: string(),
        registry: string().required(
          getString('validationMessages.entityRequired', {
            entity: getString('versionDetails.copyVersionModal.registry')
          })
        )
      })}>
      {formik => {
        const orgValue = formik.values.organization
        const projectValue = formik.values.project
        const registryUuidValue = formik.values.registryUuid
        return (
          <Container>
            <FormikForm>
              <Container className={css.form}>
                <Layout.Vertical spacing="medium">
                  <Heading level={2} className={css.sectionHeading} color={Color.BLACK}>
                    {getString('versionDetails.copyVersionModal.source')}
                  </Heading>
                  <Layout.Vertical spacing="small" className={css.sourceSection}>
                    <FormInput.Text
                      name="sourceRegistry"
                      label={getString('versionDetails.copyVersionModal.registryName')}
                      disabled
                    />
                    <FormInput.Text
                      name="sourcePackage"
                      label={getString('versionDetails.copyVersionModal.packageName')}
                      disabled
                    />
                    <FormInput.Text
                      name="sourceVersion"
                      label={getString('versionDetails.copyVersionModal.version')}
                      disabled
                    />
                  </Layout.Vertical>

                  <div className={css.separator} />
                  <Heading level={2} className={css.sectionHeadingWithTop} color={Color.BLACK}>
                    {getString('versionDetails.copyVersionModal.target')}
                  </Heading>
                  <Layout.Vertical spacing="small">
                    <OrganizationSelect
                      name="organization"
                      value={orgValue}
                      onChange={val => handleOrgChange(val, formik)}
                      disabled={disabled}
                      className={css.inputWithSpinner}
                    />
                    <ProjectSelect
                      name="project"
                      org={selectedOrg}
                      value={projectValue}
                      onChange={val => handleProjectChange(val, formik)}
                      disabled={disabled}
                      className={css.inputWithSpinner}
                    />
                    <RegistrySelect
                      name="registry"
                      value={registryUuidValue}
                      onChange={(identifier, uuid) => {
                        formik.setFieldValue('registry', identifier)
                        formik.setFieldValue('registryUuid', uuid ?? '')
                      }}
                      disabled={disabled}
                      org={selectedOrg}
                      project={selectedProject}
                      registryType={registryType}
                      packageType={packageType}
                      className={css.inputWithSpinner}
                    />
                  </Layout.Vertical>
                </Layout.Vertical>
              </Container>
              <Layout.Horizontal spacing="medium" margin={{ top: 'large' }}>
                <Button
                  variation={ButtonVariation.PRIMARY}
                  text={getString('versionList.actions.copyVersion')}
                  onClick={() => formik.submitForm()}
                  disabled={disabled}
                />
                <Button
                  variation={ButtonVariation.SECONDARY}
                  text={getString('cancel')}
                  onClick={onClose}
                  disabled={disabled}
                />
              </Layout.Horizontal>
            </FormikForm>
          </Container>
        )
      }}
    </Formik>
  )
}
