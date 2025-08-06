import React from 'react'
import { useParams } from 'react-router-dom'
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
  const { infraprovider_identifier } = useParams<{ infraprovider_identifier?: string }>()
  const editMode = infraprovider_identifier !== undefined

  const { setFieldValue, values } = useFormikContext<{ machine_type?: string }>()

  const delegateHandler = (val: string[]) => {
    formikProps.setFieldValue('delegateSelector', val)
  }

  const machineTypeOption = InfraDetails.machine_types.map(machine => {
    return {
      label: machine.name,
      value: machine.name
    }
  })

  return (
    <Layout.Vertical spacing="medium" className={css.containerSpacing}>
      <Text className={css.basicDetailsHeading}>{getString('cde.configureInfra.basicDetails')}</Text>
      <Container className={css.basicDetailsBody}>
        <FormInput.InputWithIdentifier
          inputLabel={getString('cde.configureInfra.infraName')}
          inputName="name"
          isIdentifierEditable={!editMode}
        />
        <FormInput.Text
          name="project"
          label={getString('cde.configureInfra.project')}
          placeholder={getString('cde.configureInfra.project')}
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
          placeholder={getString('cde.configureInfra.domain')}
          disabled={editMode}
        />
        <Text color={Color.GREY_500} font={{ variation: FontVariation.SMALL }}>
          {getString('cde.configureInfra.basicNoteText')}
        </Text>
        <br />
        <CustomSelectDropdown
          value={machineTypeOption.find(item => item.label === values?.machine_type)}
          onChange={(data: { label: string; value: string }) => {
            setFieldValue('machine_type', data.value)
          }}
          allowCustom
          label={getString('cde.configureInfra.gatewayMachineType')}
          options={machineTypeOption}
          // placeholder={getString('cde.configureInfra.machineType')}
        />

        <FormInput.Text
          name="gateway.vm_image_name"
          className={css.inputWithNote}
          label={getString('cde.configureInfra.gatewayImageName')}
          placeholder={getString('cde.configureInfra.gatewayImageNamePlaceholder')}
        />
        <Text color={Color.GREY_500} font={{ variation: FontVariation.SMALL }}>
          {getString('cde.configureInfra.defaultImageNoteText')}
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
