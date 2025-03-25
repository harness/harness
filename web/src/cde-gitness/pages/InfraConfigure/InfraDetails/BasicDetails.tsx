import React from 'react'
import { Container, FormInput, Label, Layout, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import css from './InfraDetails.module.scss'

const BasicDetails = () => {
  const { getString } = useStrings()
  return (
    <Layout.Vertical spacing="medium" className={css.containerSpacing}>
      <Text className={css.basicDetailsHeading}>{getString('cde.configureInfra.basicDetails')}</Text>
      <Container className={css.basicDetailsBody}>
        <Label>
          <Layout.Horizontal className={css.inlineContainer}>
            <Text className={css.inputLabel}>{getString('cde.configureInfra.name')}</Text>
            <Layout.Horizontal>
              <Text className={css.inputLabel}>{getString('cde.configureInfra.id')}: </Text>
              <Text
                color={Color.PRIMARY_7}
                rightIcon="code-edit"
                className={css.valueText}
                rightIconProps={{ size: 16 }}>
                Dhruv Test
              </Text>
            </Layout.Horizontal>
          </Layout.Horizontal>
        </Label>
        <FormInput.Text name="name" placeholder={getString('cde.configureInfra.name')} />
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
        />
        <Text className={css.noteText}>{getString('cde.configureInfra.basicNoteText')}</Text>
      </Container>
    </Layout.Vertical>
  )
}

export default BasicDetails
