import React, { useCallback } from 'react'
import * as Yup from 'yup'
import { Button, ButtonVariation, Dialog, FormInput, Layout, Text, FormikForm, Formik } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useFormikContext } from 'formik'
import { useStrings } from 'framework/strings'
import type { AdminSettingsFormValues } from 'cde-gitness/utils/cloudRegionsUtils'
import { useAppContext } from 'AppContext'
import {
  PublicPrivateRegistrySelect,
  type AccessType
} from 'cde-gitness/components/PublicPrivateRegistrySelect/PublicPrivateRegistrySelect'
import { Connectors } from 'cde-gitness/pages/AdminSettings/utils/connectorUtils'
import css from './ProvideDefaultImage.module.scss'

interface ModalFormValues {
  imagePath: string
  connectorRef: any
  accessType: AccessType
}

interface ProvideDefaultImageModalProps {
  isOpen: boolean
  onClose: (formValues?: ModalFormValues) => void
}

export const ProvideDefaultImageModal: React.FC<ProvideDefaultImageModalProps> = ({ isOpen, onClose }) => {
  const { getString } = useStrings()
  const { setFieldValue } = useFormikContext<AdminSettingsFormValues>()
  const { customComponents, accountInfo } = useAppContext()
  const { MultiTypeConnectorField } = customComponents

  const validationSchema = Yup.object({
    imagePath: Yup.string().required(getString('validation.nameIsRequired')),
    connectorRef: Yup.mixed().when('accessType', {
      is: 'private',
      then: () => Yup.mixed().required(getString('validation.connectorRequired')),
      otherwise: () => Yup.mixed().notRequired()
    })
  })

  const handleApply = useCallback(
    (values: ModalFormValues) => {
      const defaultImageData = {
        image_name: values.imagePath,
        image_connector_ref: values.connectorRef
      }

      setFieldValue('gitspaceImages', defaultImageData)
      onClose(values)
    },
    [setFieldValue, onClose]
  )

  if (!isOpen) {
    return null
  }

  return (
    <Dialog
      isOpen={isOpen}
      onClose={() => onClose()}
      title={getString('cde.settings.images.provideDefaultImagePathOrRegistry')}
      className={css.dialogContainer}>
      <Formik
        onSubmit={handleApply}
        initialValues={{
          imagePath: '',
          connectorRef: undefined,
          accessType: 'private'
        }}
        validationSchema={validationSchema}
        formName="provideDefaultImage">
        {formik => {
          return (
            <FormikForm>
              <Layout.Vertical spacing="large">
                <Layout.Vertical spacing="small" margin={{ bottom: 'large' }}>
                  <Text font={{ variation: FontVariation.BODY2_SEMI }} color={Color.GREY_700}>
                    {getString('cde.settings.images.selectImageRegistryAccessType')}
                  </Text>
                  <PublicPrivateRegistrySelect
                    selected={formik.values.accessType}
                    onChange={accessType => formik.setFieldValue('accessType', accessType)}
                  />
                </Layout.Vertical>

                {formik.values.accessType === 'private' && (
                  <Layout.Vertical spacing="small">
                    <Text font={{ variation: FontVariation.BODY2_SEMI }} color={Color.GREY_700}>
                      {getString('cde.settings.images.selectImageRegistryConnector')}
                    </Text>
                    <MultiTypeConnectorField
                      name="connectorRef"
                      formik={formik}
                      selected={formik.initialValues.connectorRef}
                      type={[Connectors.DOCKER, Connectors.AWS, Connectors.NEXUS, Connectors.ARTIFACTORY]}
                      accountIdentifier={accountInfo?.identifier}
                      placeholder={getString('cde.settings.images.selectConnector')}
                      width={'100%'}
                      multiTypeProps={{
                        allowableTypes: ['FIXED']
                      }}
                      setRefValue
                    />
                  </Layout.Vertical>
                )}

                <Layout.Vertical spacing="small" margin={{ bottom: 'medium' }}>
                  <Text font={{ variation: FontVariation.BODY2_SEMI }} color={Color.GREY_700}>
                    {getString('cde.settings.images.imageRegistryOrPath')}
                  </Text>
                  <FormInput.Text name="imagePath" placeholder="e.g mcr.microsoft.com/devcontainers/java" />
                </Layout.Vertical>
              </Layout.Vertical>

              <Layout.Horizontal spacing="medium" className={css.buttonContainer}>
                <Button
                  text={getString('cde.settings.images.apply')}
                  variation={ButtonVariation.PRIMARY}
                  type="submit"
                />
                <Button
                  text={getString('cde.settings.images.cancel')}
                  variation={ButtonVariation.TERTIARY}
                  onClick={() => onClose()}
                />
              </Layout.Horizontal>
            </FormikForm>
          )
        }}
      </Formik>
    </Dialog>
  )
}
