import React from 'react'
import { Container, Layout, Page, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import css from './InfraLoaderCard.module.scss'

function InfraLoaderCard() {
  const { getString } = useStrings()
  return (
    <Page.Body className={css.mainContainer}>
      <Layout.Vertical spacing={'xlarge'}>
        <Container className={css.loadingCard}>
          <Layout.Vertical spacing={'xlarge'}>
            <Text
              icon={'steps-spinner'}
              iconProps={{ color: Color.BLUE_500, size: 16 }}
              className={css.connectionHeader}
              color={Color.PRIMARY_7}>
              {getString('cde.gitspaceInfraHome.waitingForConnection')}...
            </Text>
            <Text className={css.connectionMessage}>
              {getString('cde.gitspaceInfraHome.waitingMessage')}{' '}
              <span className={css.linkStyle}>{getString('cde.gitspaceInfraHome.troubleshoot')}</span>
            </Text>
          </Layout.Vertical>
        </Container>
      </Layout.Vertical>
    </Page.Body>
  )
}

export default InfraLoaderCard
