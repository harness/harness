import React from 'react'
import { Container, FormInput, Layout, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import css from './InfraDetails.module.scss'

const GatewayDetails = () => {
  const { getString } = useStrings()
  return (
    <Layout.Vertical spacing="medium" className={css.containerSpacing}>
      <Text className={css.basicDetailsHeading}>{getString('cde.configureInfra.gateway')}</Text>
      <Container className={css.basicDetailsBody}>
        <FormInput.Text
          name="domain"
          className={css.inputWithNote}
          label={
            <Text
              rightIcon="info"
              className={css.inputLabel}
              rightIconProps={{ color: Color.PRIMARY_7, size: 12, margin: 5 }}>
              {getString('cde.configureInfra.numberOfInstance')}
            </Text>
          }
          placeholder={getString('cde.configureInfra.numberOfInstance')}
        />
        <Text className={css.noteText}>{getString('cde.configureInfra.gatewayNoteText')}</Text>
        <FormInput.Text
          className={css.inputWithMargin}
          name="machineType"
          label={getString('cde.configureInfra.machineType')}
          placeholder={getString('cde.configureInfra.machineType')}
        />
      </Container>
    </Layout.Vertical>
  )
}

export default GatewayDetails
