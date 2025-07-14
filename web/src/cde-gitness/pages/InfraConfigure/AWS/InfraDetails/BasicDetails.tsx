import React from 'react'
import { Container, FormInput, Layout, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useFormikContext, type FormikProps } from 'formik'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import CustomSelectDropdown from 'cde-gitness/components/CustomSelectDropdown/CustomSelectDropdown'
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

  const { setFieldValue, values } = useFormikContext<{ instance_type?: string; machine_type?: string }>()

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
          isIdentifierEditable={true}
        />
        <FormInput.Text name="vpc_cidr_block" label={getString('cde.Aws.VpcCidrBlock')} placeholder="10.6.0.0/16" />
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
        />
        <Text font={{ variation: FontVariation.SMALL }}>{getString('cde.configureInfra.basicNoteText')}</Text>
        <br />
        <CustomSelectDropdown
          value={instanceTypeOption.find(item => item.value === values?.machine_type)}
          onChange={(data: { label: string; value: string }) => {
            setFieldValue('machine_type', data.value)
          }}
          allowCustom
          label={getString('cde.Aws.gatewayInstanceType')}
          options={instanceTypeOption}
        />
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
