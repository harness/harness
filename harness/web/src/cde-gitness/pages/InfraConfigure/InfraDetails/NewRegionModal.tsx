import React from 'react'
import {
  Button,
  ButtonVariation,
  Container,
  Formik,
  FormikForm,
  FormInput,
  ModalDialog,
  SelectOption,
  Text
} from '@harnessio/uicore'
import * as Yup from 'yup'
import cidrRegex from 'cidr-regex'
import { useFormikContext } from 'formik'
import { Color, FontVariation } from '@harnessio/design-system'
import type { regionProp } from 'cde-gitness/constants'
import { useStrings } from 'framework/strings'
import CustomSelectDropdown from 'cde-gitness/components/CustomSelectDropdown/CustomSelectDropdown'
import { InfraDetails } from './InfraDetails.constants'
import css from './NewRegionModal.module.scss'

interface NewRegionModalProps {
  isOpen: boolean
  setIsOpen: (value: boolean) => void
  onSubmit: (value: NewRegionModalForm) => void
  initialValues?: regionProp | null
  isEditMode?: boolean
  existingRegions?: string[]
}

type NewRegionModalForm = regionProp

const validationSchema = () =>
  Yup.object().shape({
    location: Yup.string().required('Location is required'),
    defaultSubnet: Yup.string()
      .matches(cidrRegex({ exact: true }), 'Invalid CIDR format')
      .required('Default Subnet is required'),
    proxySubnet: Yup.string()
      .matches(cidrRegex({ exact: true }), 'Invalid CIDR format')
      .required('Proxy Subnet is required'),
    domain: Yup.string().required('Domain is required')
  })

const NewRegionModal = ({
  isOpen,
  setIsOpen,
  onSubmit,
  initialValues,
  isEditMode = false,
  existingRegions = []
}: NewRegionModalProps) => {
  const { getString } = useStrings()

  const { values } = useFormikContext<{ domain: string }>()

  const regionOptions = Object.keys(InfraDetails.regions)
    .filter(region => {
      // Include the current region if in edit mode, exclude if already in existingRegions
      return isEditMode
        ? initialValues?.location === region || !existingRegions.includes(region)
        : !existingRegions.includes(region)
    })
    .map(item => {
      return {
        label: item,
        value: item
      }
    })
  const getInitialValues = (): NewRegionModalForm => {
    if (initialValues) {
      const domainPrefix = initialValues.domain ? initialValues.domain.replace(`.${values?.domain}`, '') : ''

      return {
        location: initialValues.location || '',
        defaultSubnet: initialValues.defaultSubnet || '',
        proxySubnet: initialValues.proxySubnet || '',
        domain: domainPrefix,
        identifier: initialValues.identifier || 0
      }
    }
    return {
      location: '',
      defaultSubnet: '',
      proxySubnet: '',
      domain: '',
      identifier: 0
    }
  }

  return (
    <ModalDialog
      isOpen={isOpen}
      onClose={() => setIsOpen(false)}
      width={800}
      title={isEditMode ? 'Edit Region' : getString('cde.gitspaceInfraHome.newRegion')}>
      <Formik<NewRegionModalForm>
        validationSchema={validationSchema()}
        onSubmit={formValues => {
          const fullDomain = formValues.domain ? `${formValues.domain}.${values.domain}` : values.domain
          onSubmit({
            ...formValues,
            domain: fullDomain
          })
        }}
        formName={''}
        initialValues={getInitialValues()}>
        {formikProps => {
          return (
            <FormikForm>
              <CustomSelectDropdown
                value={regionOptions.find(item => item.label === formikProps?.values?.location)}
                onChange={(data: SelectOption) => {
                  formikProps.setFieldValue('location', data?.value as string)
                }}
                label={getString('cde.gitspaceInfraHome.locationName')}
                options={regionOptions}
                error={formikProps.errors.location}
                disabled={isEditMode}
                // placeholder="e.g us-west1"
              />
              <FormInput.Text
                placeholder="e.g 10.6.0.0/16"
                name="defaultSubnet"
                label={getString('cde.gitspaceInfraHome.defaultSubnet')}
              />
              <FormInput.Text
                placeholder="e.g 10.3.0.0/16"
                name="proxySubnet"
                label={getString('cde.gitspaceInfraHome.proxySubnet')}
              />
              <Container className="form-group">
                <Text className="form-group--label" font={{ variation: FontVariation.BODY }} color={Color.GREY_500}>
                  {getString('cde.configureInfra.domain')}
                </Text>
                <Container className={css.inputContainer}>
                  <Container className={css.inputWrapper}>
                    <FormInput.Text name="domain" placeholder="e.g us-west-ga.io" />
                    <span className={css.domainSuffix}>.{values?.domain}</span>
                  </Container>
                </Container>
              </Container>

              {/*<Button*/}
              {/*  variation={ButtonVariation.PRIMARY}*/}
              {/*  type="submit"*/}
              {/*  style={{ marginLeft: '75%' }}*/}
              {/*  margin={{ top: 'medium' }}>*/}
              {/*  {getString('cde.gitspaceInfraHome.addnewRegion')}*/}
              {/*</Button>*/}

              <Container className={css.buttonContainer}>
                <Button variation={ButtonVariation.PRIMARY} type="submit" className={css.actionButton}>
                  {isEditMode ? getString('save') : getString('cde.gitspaceInfraHome.addnewRegion')}
                </Button>
                <Button
                  variation={ButtonVariation.TERTIARY}
                  text={getString('cancel')}
                  onClick={() => setIsOpen(false)}
                  className={css.cancelButton}
                />
              </Container>
            </FormikForm>
          )
        }}
      </Formik>
    </ModalDialog>
  )
}

export default NewRegionModal
