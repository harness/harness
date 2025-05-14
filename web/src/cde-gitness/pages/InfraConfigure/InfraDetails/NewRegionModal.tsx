import React from 'react'
import { Button, ButtonVariation, Formik, FormikForm, FormInput, ModalDialog } from '@harnessio/uicore'
import * as Yup from 'yup'
import cidrRegex from 'cidr-regex'
import { useFormikContext } from 'formik'
import type { regionProp } from 'cde-gitness/constants'
import { useStrings } from 'framework/strings'
import CustomSelectDropdown from 'cde-gitness/components/CustomSelectDropdown/CustomSelectDropdown'
import { InfraDetails } from './InfraDetails.constants'

interface NewRegionModalProps {
  isOpen: boolean
  setIsOpen: (value: boolean) => void
  onSubmit: (value: NewRegionModalForm) => void
}

type NewRegionModalForm = regionProp

const validationSchema = (context: { domain: string }) =>
  Yup.object().shape({
    location: Yup.string().required('Location is required'),
    defaultSubnet: Yup.string()
      .matches(cidrRegex({ exact: true }), 'Invalid CIDR format')
      .required('Default Subnet is required'),
    proxySubnet: Yup.string()
      .matches(cidrRegex({ exact: true }), 'Invalid CIDR format')
      .required('Proxy Subnet is required'),
    domain: Yup.string()
      .required('Domain is required')
      .test('ends-with-domain', `Domain must end with ${context.domain}`, function (value) {
        return value ? value.endsWith(context.domain) : false
      }),
    dns: Yup.string().required('DNS Managed Zone is required')
  })

const NewRegionModal = ({ isOpen, setIsOpen, onSubmit }: NewRegionModalProps) => {
  const { getString } = useStrings()

  const { values, setFieldValue } = useFormikContext<{ domain: string; location: string }>()

  const regionOptions = Object.keys(InfraDetails.regions).map(item => {
    return {
      label: item,
      value: item
    }
  })

  return (
    <ModalDialog
      isOpen={isOpen}
      onClose={() => setIsOpen(false)}
      width={700}
      title={getString('cde.gitspaceInfraHome.newRegion')}>
      <Formik<NewRegionModalForm>
        validationSchema={validationSchema({ domain: values.domain })}
        onSubmit={onSubmit}
        formName={''}
        initialValues={{
          location: '',
          defaultSubnet: '',
          proxySubnet: '',
          domain: `*.${values?.domain}`,
          dns: '',
          identifier: 0
        }}>
        {() => {
          return (
            <FormikForm>
              <CustomSelectDropdown
                value={regionOptions.find(item => item.label === values?.location)}
                onChange={(data: string) => setFieldValue('location', data)}
                label={getString('cde.gitspaceInfraHome.locationName')}
                options={regionOptions}
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
              <FormInput.Text
                name="domain"
                placeholder="e.g us-west-ga.io"
                label={getString('cde.configureInfra.domain')}
              />
              <FormInput.Text
                placeholder="e.g us-west-ga.io"
                name="dns"
                label={getString('cde.gitspaceInfraHome.dnsManagedZone')}
              />

              <Button variation={ButtonVariation.PRIMARY} type="submit" style={{ marginLeft: '75%' }}>
                {getString('cde.gitspaceInfraHome.addnewRegion')}
              </Button>
            </FormikForm>
          )
        }}
      </Formik>
    </ModalDialog>
  )
}

export default NewRegionModal
