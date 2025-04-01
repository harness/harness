import React from 'react'
import { Button, ButtonVariation, Container, Layout, Page, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { Link, useHistory } from 'react-router-dom'
import { routes } from 'cde-gitness/RouteDefinitions'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import InfraLogo from '../../../../icons/infra_home_icon.svg?url'
import css from './NoDataCard.module.scss'

const NoDataCard = () => {
  const { getString } = useStrings()
  const history = useHistory()
  const { accountInfo } = useAppContext()

  return (
    <Page.Body className={css.main}>
      <Container className={css.titleContainer}>
        <Layout.Vertical spacing="small" margin={{ bottom: 'medium' }}>
          <img src={InfraLogo} className={css.infraLogo} />
          <Text font={{ size: 'medium', weight: 'bold' }} className={css.containerHeading} color={Color.BLACK}>
            {getString('cde.configureGitspaceInfra')}
          </Text>
          <Text className={css.infraDescription}>{getString('cde.gitspaceInfraHome.description')}</Text>
          <Button
            onClick={() => history.push(routes.toCDEInfraConfigure({ accountId: accountInfo?.identifier }))}
            font={{ size: 'small' }}
            className={css.configureButton}
            variation={ButtonVariation.PRIMARY}>
            {getString('cde.gitspaceInfraHome.configureGCPButton')}
          </Button>
          <Text className={css.supportText} icon="info-messaging">
            {getString('cde.gitspaceInfraHome.gcpSupportText')}
            <Link to={'/'} className={css.learnMoreText} style={{ paddingLeft: '4px' }}>
              {getString('cde.gitspaceInfraHome.learnMore')}
            </Link>
          </Text>
        </Layout.Vertical>
      </Container>
    </Page.Body>
  )
}

export default NoDataCard
