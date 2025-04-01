import React from 'react'
import { Container, FormInput, Label, Layout, Text, TextInput } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import type { FormikProps } from 'formik'
import { useStrings } from 'framework/strings'
import css from './InfraDetails.module.scss'

interface GatewayProps {
  formikProps: FormikProps<any>
}

const GatewayDetails = ({ formikProps }: GatewayProps) => {
  const { getString } = useStrings()
  return (
    <Layout.Vertical spacing="medium" className={css.containerSpacing}>
      <Text className={css.basicDetailsHeading}>{getString('cde.configureInfra.gateway')}</Text>
      <Container className={css.basicDetailsBody}>
        <Label>
          <Text
            rightIcon="info"
            className={css.inputLabel}
            rightIconProps={{ color: Color.PRIMARY_7, size: 12, margin: 5 }}>
            {getString('cde.configureInfra.numberOfInstance')}
          </Text>
        </Label>
        <TextInput
          name="instances"
          value={formikProps?.values?.instances ?? 0}
          type="number"
          className={css.inputWithNote}
          onChange={(e: React.FormEvent<any>) => formikProps.setFieldValue('instances', e.currentTarget.value)}
          placeholder={getString('cde.configureInfra.numberOfInstance')}
        />

        <Text className={css.noteText}>{getString('cde.configureInfra.gatewayNoteText')}</Text>
        <FormInput.Text
          className={css.inputWithMargin}
          name="machine_type"
          label={getString('cde.configureInfra.machineType')}
          placeholder={getString('cde.configureInfra.machineType')}
        />
      </Container>
    </Layout.Vertical>
  )
}

export default GatewayDetails
