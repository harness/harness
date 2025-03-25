import React from 'react'
import { Container, Layout, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import css from './InfraDetails.module.scss'

const ConfigureLocations = () => {
  const { getString } = useStrings()
  return (
    <Layout.Vertical spacing="small" className={css.containerSpacing}>
      <Text className={css.basicDetailsHeading}>{getString('cde.configureInfra.configureLocations')}</Text>
      <Layout.Horizontal spacing="small">
        <Text color={Color.GREY_400} className={css.headerLinkText}>
          {getString('cde.configureInfra.configureLocationNote')}
        </Text>
        <Text color={Color.PRIMARY_7} className={css.headerLinkText}>
          {getString('cde.configureInfra.learnMore')}
        </Text>
      </Layout.Horizontal>

      <Container className={css.basicDetailsBody}></Container>
    </Layout.Vertical>
  )
}

export default ConfigureLocations
