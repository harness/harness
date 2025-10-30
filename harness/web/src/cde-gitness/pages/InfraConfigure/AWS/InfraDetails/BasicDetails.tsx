import React from 'react'
import { useParams } from 'react-router-dom'
import { Container, FormInput, Layout, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useFormikContext, type FormikProps } from 'formik'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import CustomSelectDropdown from 'cde-gitness/components/CustomSelectDropdown/CustomSelectDropdown'
import CustomInput from 'cde-gitness/components/CustomInput/CustomInput'
import { InfraDetails } from './InfraDetails.constants'
import css from './InfraDetails.module.scss'

interface BasicDetailProps {
  formikProps: FormikProps<any>
}

const BasicDetails = ({ formikProps }: BasicDetailProps) => {
  const { getString } = useStrings()
  const { hooks, accountInfo, customComponents } = useAppContext()
  const { DelegateSelectorsV2 } = customComponents
  const queryParams = { accountId: accountInfo?.identifier }
  const { data: delegateData } = hooks.useGetDelegateSelectorsUpTheHierarchyV2({
    queryParams
  })
  const { infraprovider_identifier } = useParams<{ infraprovider_identifier?: string }>()
  const editMode = infraprovider_identifier !== undefined

  const { setFieldValue, values, errors } = useFormikContext<{ instance_type?: string; instances?: string }>()

  const delegateHandler = (val: string[]) => {
    formikProps.setFieldValue('delegateSelector', val)
  }

  const instanceTypeOption = InfraDetails.instance_types.map(instance => {
    return {
      label: instance.name,
      value: instance.name
    }
  })

  return (
    <Layout.Vertical spacing="medium" className={css.containerSpacing}>
      <Text className={css.basicDetailsHeading}>{getString('overview')}</Text>
      <Container className={css.basicDetailsBody}>
        <FormInput.InputWithIdentifier
          inputLabel={getString('cde.configureInfra.infraName')}
          inputName="name"
          isIdentifierEditable={!editMode}
        />
        <FormInput.Text
          name="vpc_cidr_block"
          label={getString('cde.Aws.VpcCidrBlock')}
          placeholder="10.6.0.0/16"
          disabled={editMode}
        />
        <FormInput.Text
          name="domain"
          className={css.inputWithNote}
          label={
            <Text
              rightIcon="info"
              className={css.inputLabel}
              rightIconProps={{ color: Color.PRIMARY_7, size: 12, margin: 5 }}>
              {getString('cde.configureInfra.domain')}
            </Text>
          }
          placeholder={getString('cde.Aws.domainExample')}
          disabled={editMode}
        />
        <Text color={Color.GREY_500} font={{ variation: FontVariation.SMALL }}>
          {getString('cde.configureInfra.basicNoteText')}
        </Text>
        <br />
        <CustomSelectDropdown
          value={instanceTypeOption.find(item => item.value === values?.instance_type)}
          onChange={(data: { label: string; value: string }) => {
            setFieldValue('instance_type', data.value)
          }}
          allowCustom
          label={getString('cde.Aws.gatewayInstanceType')}
          options={instanceTypeOption}
        />
        <CustomInput
          marginBottom={false}
          label={getString('cde.configureInfra.NumberOfInstance')}
          name="instances"
          type="text"
          value={values.instances || ''}
          autoComplete="off"
          onChange={(form: { value: string }) => {
            if (form.value === '' || /[0-9]+/.test(form.value)) {
              const numValue = parseInt(form.value, 10)
              if (form.value !== '' && numValue < 1) {
                return
              }
              const valueWithoutLeadingZeros = form.value === '' ? '' : String(numValue)
              setFieldValue('instances', valueWithoutLeadingZeros)
            }
          }}
          placeholder="default: 3"
          error={errors.instances}
        />
        <Text color={Color.GREY_500} font={{ variation: FontVariation.SMALL }}>
          {getString('cde.configureInfra.instanceNoteText')}
        </Text>
        <Container className={css.delegateContainer}>
          <Text className={css.delegateSelector}>{getString('cde.delegate.DelegateSelector')}</Text>
          <DelegateSelectorsV2
            data={delegateData?.resource ?? []}
            selectedItems={formikProps?.values?.delegateSelector}
            onTagInputChange={delegateHandler}
          />
        </Container>
      </Container>
    </Layout.Vertical>
  )
}

export default BasicDetails
